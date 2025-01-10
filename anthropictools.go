package anthropic

import (
    "context"
    "encoding/json"
    "fmt"
)

// ChatWithTools implements the core tool interaction loop, allowing the assistant
// to use tools while maintaining conversation context. It includes comprehensive
// logging throughout the tool execution process.
func (c *AnthropicClient) ChatWithTools(
    ctx context.Context,
    message string,
    params *MessageParams,
    handlers map[string]func(context.Context, json.RawMessage) (string, error),
) (*AnthropicResponse, error) {
    logMessage("Starting tool-enabled chat interaction")
    logJSON("Initial message", message)
    logJSON("Tool parameters", params)
    
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

        // Record assistant's response in conversation
        c.addMessageToConversation(RoleAssistant, resp.Content)
        logJSON("Updated conversation with assistant response", c.conversation)

        // Check if tool use is required or if we're done
        if resp.StopReason != StopReasonToolUse {
            logMessage("Tool interaction complete - Stop reason: %s", resp.StopReason)
            return resp, nil
        }

        // Extract and validate tool calls from the response
        toolCalls := extractToolCalls(resp)
        logJSON("Extracted tool calls from response", toolCalls)
        
        if len(toolCalls) == 0 {
            logMessage("Error: No valid tool calls found despite tool_use stop reason")
            return nil, fmt.Errorf("received tool_use stop reason but no valid tool calls found")
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

        // Add tool results to conversation history
        c.addMessageToConversation(RoleUser, resultContents)
        logJSON("Updated conversation with tool results", c.conversation)

        // Adjust tool choice after first iteration
        if iterations == 0 {
            logMessage("Clearing ToolChoice after first iteration")
            //params.ToolChoice = nil
        }
        
        iterations++
    }
}

// extractToolCalls processes the assistant's response to identify and validate
// any tool calls. It includes detailed logging of the extraction process.
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
            
            // Validate required fields
            if content.ID == "" || content.Name == "" || content.Input == nil {
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

// validateToolParams ensures the tool configuration is valid and includes
// appropriate logging of the validation process.
func validateToolParams(params *MessageParams) error {
    logMessage("Validating tool parameters")
    
    if params.Tools != nil && len(params.Tools) > 0 {
        logMessage("Tools are specified (%d tools configured)", len(params.Tools))
        
        if params.ToolChoice == nil {
            logMessage("Error: ToolChoice must be specified when tools are provided")
            return fmt.Errorf("tool_choice must be specified when tools are provided")
        }
        
        logMessage("Validating ToolChoice type: %s", params.ToolChoice.Type)
        if params.ToolChoice.Type != ToolChoiceAuto && 
           params.ToolChoice.Type != ToolChoiceNone && 
           params.ToolChoice.Type != ToolChoiceTool {
            logMessage("Error: Invalid ToolChoice type: %s", params.ToolChoice.Type)
            return fmt.Errorf("invalid tool_choice type: %s", params.ToolChoice.Type)
        }
        
        if params.ToolChoice.Type == ToolChoiceTool && params.ToolChoice.Name == "" {
            logMessage("Error: ToolChoice name is required for type 'tool'")
            return fmt.Errorf("tool_choice name must be specified when type is 'tool'")
        }
    }
    
    logMessage("Tool parameter validation successful")
    return nil
}
