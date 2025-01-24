package anthropic

import (
    "context"
    "encoding/json"
    "fmt"
    "regexp"
)

var (
    // Tool name must match regex ^[a-zA-Z0-9_-]{1,64}$
    toolNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
)
func (c *AnthropicClient) ChatWithTools(
    ctx context.Context, 
    message string,
    params *MessageParams,
    handlers map[string]func(context.Context, json.RawMessage) (string, error),
) (*AnthropicResponse, error) {
    var finalAnswer string
    var toolResults []MessageContent
    
    // Initialize conversation with user message
    messages := []Message{{
        Role: RoleUser,
        Content: []MessageContent{{
            Type: ContentTypeText,
            Text: message,
        }},
    }}

    // Track all responses for final result
    var allResponses []MessageContent

    // Main conversation loop
    for {
        // Send request with current messages
        resp, err := c.sendRequest(ctx, Request{
            Model:       params.Model,
            Messages:    messages,
            MaxTokens:   params.MaxTokens,
            Tools:       params.Tools,
            ToolChoice:  &ToolChoice{
                Type: ToolChoiceAuto, 
                DisableParallel: true,
            },
        })
        if err != nil {
            return nil, fmt.Errorf("request error: %w", err)
        }

        // Process response blocks
        for _, block := range resp.Content {
            switch block.Type {
            case ContentTypeText:
                finalAnswer += block.Text
                allResponses = append(allResponses, block)
                
            case ContentTypeToolUse:
                // Execute tool
                handler, exists := handlers[block.Name]
                if !exists {
                    return nil, fmt.Errorf("no handler for tool: %s", block.Name)
                }
                
                result, err := handler(ctx, block.Input)
                if err != nil {
                    return nil, fmt.Errorf("tool execution error: %w", err)
                }
                
                // Store tool result
                toolResults = append(toolResults, MessageContent{
                    Type:      ContentTypeToolResult,
                    ToolUseID: block.ID,
                    Content:   result,
                })
                allResponses = append(allResponses, toolResults...)
            }
        }

        // Add tool results to conversation if any
        if len(toolResults) > 0 {
            messages = append(messages, Message{
                Role:    RoleUser,
                Content: toolResults,
            })
            toolResults = nil // Reset for next iteration
        }

        // Return final answer with complete response history
        if resp.StopReason != StopReasonToolUse {
            return &AnthropicResponse{
                Content:    allResponses,
                StopReason: resp.StopReason,
                Usage:     resp.Usage,
            }, nil
        }
    }
}
// ChatWithTools implements the core tool interaction loop according to Anthropic's
// documented flow:
// 1. Provide Claude with tools and user prompt
// 2. Claude decides to use tool (returns stop_reason: tool_use)
// 3. Extract tool input, run code, return results
// 4. Claude uses tool result to formulate final response
func (c *AnthropicClient) AChatWithTools(
    ctx context.Context,
    message string,
    params *MessageParams,
    handlers map[string]func(context.Context, json.RawMessage) (string, error),
) (*AnthropicResponse, error) {
    logMessage("Starting tool-enabled chat interaction")
    logJSON("Initial message", message)
    logJSON("Tool parameters", params)
    
       // ADD THIS SECTION
    // Set default tool_choice if not provided
    if params.Tools != nil && len(params.Tools) > 0 && params.ToolChoice == nil {
        params.ToolChoice = &ToolChoice{Type: ToolChoiceAuto}
    }

    // Validate tool configuration before proceeding
    if err := validateToolParams(params); err != nil {
        logMessage("Tool parameter validation failed: %v", err)
        return nil, fmt.Errorf("invalid tool parameters: %w", err)
    }

    // Initialize conversation with user's message
    initialContent := []MessageContent{{
        Type: ContentTypeText,
        Text: message,
    }}
    c.addMessageToConversation(RoleUser, initialContent)
    logJSON("Initial conversation state", c.conversation)

    // Configure iteration limits to prevent infinite loops
    const maxIterations = 10
    iterations := 0

    // Store original tool choice for later reset if needed
    originalToolChoice := params.ToolChoice
    disableParallel := false
    if params.ToolChoice != nil && params.ToolChoice.DisableParallel {
        disableParallel = true
    }

    // Main tool interaction loop
    for {
        logMessage("Starting tool interaction iteration %d/%d", iterations+1, maxIterations)
        
        if iterations >= maxIterations {
            logMessage("Tool interaction loop exceeded maximum iterations")
            return nil, fmt.Errorf("exceeded maximum number of tool call iterations (%d)", maxIterations)
        }

        // Prepare request with current conversation state
        reqBody := Request{
            Model:       params.Model,
            System:      params.System,
            Messages:    c.conversation,
            MaxTokens:   params.MaxTokens,
            Temperature: params.Temperature,
            TopP:        params.TopP,
            TopK:        params.TopK,
            Tools:       params.Tools,
            ToolChoice:  params.ToolChoice,
        }
        logJSON("Outgoing request for tool interaction", reqBody)

        // Get assistant's response
        resp, err := c.sendRequest(ctx, reqBody)
        if err != nil {
            logMessage("Failed to get assistant response: %v", err)
            return nil, fmt.Errorf("chat request error (iteration %d): %w", iterations, err)
        }
        logJSON("Received assistant response", resp)

        // Process any initial text or chain-of-thought from Claude
        if len(resp.Content) > 0 {
            c.addMessageToConversation(RoleAssistant, resp.Content)
            logJSON("Updated conversation with assistant response", c.conversation)
        }

        // If not a tool use response, this is the final response
        if resp.StopReason != StopReasonToolUse {
            logMessage("Tool interaction complete - Final response received")
            // Ensure the response content is added to conversation before returning
            return resp, nil
        }

        // Extract and validate tool calls from the response
        toolCalls := extractToolCalls(resp)
        logJSON("Extracted tool calls from response", toolCalls)
        
        if len(toolCalls) == 0 {
            logMessage("Error: No valid tool calls found despite tool_use stop reason")
            return nil, fmt.Errorf("received tool_use stop reason but no valid tool calls found")
        }

        // Enforce single tool use if parallel is disabled
        if disableParallel && len(toolCalls) > 1 {
            logMessage("Warning: Multiple tool calls received with parallel disabled, using only first call")
            toolCalls = toolCalls[:1]
        }

        // Process each tool call and collect results
        var resultContents []MessageContent
        for _, call := range toolCalls {
            logMessage("Processing tool call - Tool: %s, ID: %s", call.Name, call.ID)
            logJSON("Tool call input parameters", string(call.Input))

            // Find the appropriate handler for this tool
            handler, exists := handlers[call.Name]
            if !exists {
                logMessage("Error: No handler found for tool '%s'", call.Name)
                return nil, fmt.Errorf("no handler for tool: %s", call.Name)
            }

            // Execute the tool and handle any errors
            logMessage("Executing tool '%s'", call.Name)
            result, err := handler(ctx, call.Input)
            if err != nil {
                logMessage("Tool execution failed: %v", err)
                // Return error result according to Anthropic's format
                resultContents = append(resultContents, MessageContent{
                    Type:      ContentTypeToolResult,
                    ToolUseID: call.ID,
                    Content:   fmt.Sprintf("Error executing tool: %v", err),
                    IsError:   true,
                })
                continue
            }
            
            logMessage("Tool execution successful")
            logJSON("Tool execution result", result)
            
            // Record successful tool execution result
            resultContents = append(resultContents, MessageContent{
                Type:      ContentTypeToolResult,
                ToolUseID: call.ID,
                Content:   result,
            })
        }

        // Add tool results to conversation history as user message
        c.addMessageToConversation(RoleUser, resultContents)
        logJSON("Updated conversation with tool results", c.conversation)

        // After first iteration:
        // 1. Clear tool choice to allow Claude to formulate final response
        // 2. Reset to original tool choice for subsequent iterations if needed
        if iterations == 0 {
            logMessage("Clearing tool choice after first iteration")
            params.ToolChoice = nil
        } else {
            params.ToolChoice = originalToolChoice
        }
        
        iterations++
    }
}

// extractToolCalls processes the assistant's response to identify and validate
// tool calls according to Anthropic's specification
func extractToolCalls(resp *AnthropicResponse) []ToolUse {
    logMessage("Extracting tool calls from response")
    var calls []ToolUse
    
    if resp == nil {
        logMessage("Warning: Response is nil, returning empty tool calls")
        return calls
    }

    // Process each content item for potential tool calls
    for i, content := range resp.Content {
        if content.Type == ContentTypeToolUse {
            logMessage("Processing potential tool call %d", i+1)
            
            // Validate required fields according to Anthropic's spec
            if !isValidToolCall(content) {
                logMessage("Skipping invalid tool call - Missing required fields (ID: %s, Name: %s)", 
                    content.ID, content.Name)
                continue
            }
            
            // Create and record valid tool call
            call := ToolUse{
                ID:    content.ID,
                Name:  content.Name,
                Input: content.Input,
            }
            logJSON("Valid tool call found", call)
            calls = append(calls, call)
        }
    }
    
    logMessage("Extracted %d valid tool calls", len(calls))
    return calls
}

// validateToolParams ensures the tool configuration is valid according to
// Anthropic's requirements
func validateToolParams(params *MessageParams) error {
    logMessage("Validating tool parameters")
    
    if params.Tools != nil && len(params.Tools) > 0 {
        logMessage("Tools are specified (%d tools configured)", len(params.Tools))
        
        // Validate each tool definition
        for _, tool := range params.Tools {
            // Validate tool name format
            if !toolNameRegex.MatchString(tool.Name) {
                return fmt.Errorf("invalid tool name format: %s - must match %s", 
                    tool.Name, toolNameRegex.String())
            }
            
            // Validate tool has description
            if tool.Description == "" {
                return fmt.Errorf("tool %s missing required description", tool.Name)
            }
            
            // Validate input schema
            if err := validateInputSchema(tool.InputSchema); err != nil {
                return fmt.Errorf("invalid input schema for tool %s: %w", tool.Name, err)
            }
        }

        // Validate tool choice configuration
        if err := validateToolChoice(params.ToolChoice); err != nil {
            return fmt.Errorf("invalid tool choice configuration: %w", err)
        }
    }
    
    logMessage("Tool parameter validation successful")
    return nil
}

// validateToolChoice ensures the tool choice configuration is valid
func validateToolChoice(choice *ToolChoice) error {
    if choice == nil {
        return fmt.Errorf("tool_choice must be specified when tools are provided")
    }

    // Validate choice type
    switch choice.Type {
    case ToolChoiceAuto, ToolChoiceNone:
        return nil
    case ToolChoiceTool:
        if choice.Name == "" {
            return fmt.Errorf("tool_choice name must be specified when type is 'tool'")
        }
        return nil
    default:
        return fmt.Errorf("invalid tool_choice type: %s", choice.Type)
    }
}

// validateInputSchema ensures the tool's input schema is valid JSON Schema
func validateInputSchema(schema InputSchema) error {
    if schema.Type != "object" {
        return fmt.Errorf("input schema type must be 'object'")
    }
    
    if len(schema.Properties) == 0 {
        return fmt.Errorf("input schema must define at least one property")
    }
    
    return nil
}

// isValidToolCall checks if a tool call content block has all required fields
func isValidToolCall(content MessageContent) bool {
    return content.ID != "" && 
           content.Name != "" && 
           content.Input != nil && 
           toolNameRegex.MatchString(content.Name)
}
