# Anthropic Types Documentation

## Overview
The `anthropictypes.go` file defines the core data structures and constants used throughout the Anthropic API client implementation. This document explains how to use these types effectively when working with the Anthropic API client.

## Configuration Constants

### API Configuration
```go
defaultAPIEndpoint = "https://api.anthropic.com/v1/messages"
defaultModel = "claude-3-5-sonnet-20241022"
```
These constants define the default API endpoint and model. They are used automatically by the client but can be overridden through MessageParams.

### Role Constants
```go
RoleSystem    = "system"
RoleUser      = "user"
RoleAssistant = "assistant"
```
These constants are used when constructing messages in a conversation. They define the role of each participant in the conversation.

### Content Type Constants
```go
ContentTypeText       = "text"
ContentTypeToolUse    = "tool_use"
ContentTypeToolResult = "tool_result"
ContentTypeThinking   = "thinking"
```
These constants define the different types of content that can be included in messages.

## Core Types

### AnthropicClient
```go
type AnthropicClient struct {
    apiKey          string
    defaultParams   MessageParams
    httpClient      *http.Client
    conversation    []Message
    maxConvLength   int
    systemPrompt    string
}
```

The AnthropicClient is the main entry point for interacting with the API. To create a new client:

```go
client := NewClient(apiKey, WithMaxConversationLength(100))
```

### MessageParams
```go
type MessageParams struct {
    Model       string
    MaxTokens   int
    Temperature float64
    TopP        float64
    TopK        int
    Metadata    map[string]interface{}
    System      string
    Tools       []Tool
    ToolChoice  *ToolChoice
}
```

MessageParams allows you to customize each request. Example usage:

```go
params := MessageParams{
    Model: "claude-3-5-sonnet-20241022",
    MaxTokens: 1000,
    Temperature: 0.7,
    Tools: GetDefaultTools(),
    ToolChoice: &ToolChoice{
        Type: ToolChoiceAuto,
    },
}
```

### Message and MessageContent
```go
type Message struct {
    Role    string
    Content []MessageContent
}

type MessageContent struct {
    Type       string
    Text       string
    ID         string
    Name       string
    Input      json.RawMessage
    ToolUseID  string
    Content    string
    IsError    bool
}
```

These types represent the structure of messages in a conversation. Example:

```go
message := Message{
    Role: RoleUser,
    Content: []MessageContent{
        {
            Type: ContentTypeText,
            Text: "What's the weather like?",
        },
    },
}
```

## Tool-Related Types

### Tool
```go
type Tool struct {
    Name         string
    Description  string
    InputSchema  InputSchema
}
```

Tools define the available actions that can be performed. The client comes with default tools accessible via `GetDefaultTools()`.

### ToolChoice
```go
type ToolChoice struct {
    Type string
    Name string
}
```

ToolChoice controls how tools are selected:
- `ToolChoiceAuto`: Automatic tool selection
- `ToolChoiceNone`: Disable tool usage
- `ToolChoiceTool`: Force use of a specific tool

Example:
```go
toolChoice := &ToolChoice{
    Type: ToolChoiceTool,
    Name: "get_weather",
}
```

## Response Types

### AnthropicResponse
```go
type AnthropicResponse struct {
    ID          string
    Type        string
    Role        string
    Content     []MessageContent
    Model       string
    StopReason  string
    Usage       Usage
}
```

This type represents the response from the API. The StopReason field can be one of:
- `StopReasonToolUse`
- `StopReasonEndTurn`
- `StopReasonMaxTokens`
- `StopReasonStopSequence`

## Best Practices

1. Always use the provided constants instead of hardcoding strings:
```go
// Good
role := RoleAssistant
// Bad
role := "assistant"
```

2. When creating new tools, ensure the InputSchema is properly defined:
```go
tool := Tool{
    Name: "custom_tool",
    Description: "Does something useful",
    InputSchema: InputSchema{
        Type: "object",
        Properties: map[string]Property{
            "param1": {
                Type: "string",
                Description: "First parameter",
            },
        },
        Required: []string{"param1"},
    },
}
```

3. Handle responses appropriately based on StopReason:
```go
if response.StopReason == StopReasonToolUse {
    // Process tool usage
} else if response.StopReason == StopReasonEndTurn {
    // Normal conversation end
}
```

4. Use ClientOption functions for configuration:
```go
client := NewClient(apiKey,
    WithMaxConversationLength(100),
    WithSystemPrompt(customPrompt),
    WithDefaultParams(defaultParams),
)
```

## Error Handling

When working with tool results, implement comprehensive error handling:
```go
// Example of comprehensive tool result handling
for _, content := range response.Content {
    if content.Type == ContentTypeToolResult {
        // Always check the ToolUseID to match with the original request
        if content.ToolUseID == "" {
            log.Printf("Warning: Tool result missing ToolUseID")
            continue
        }

        if content.IsError {
            // Handle different types of errors appropriately
            switch {
            case strings.Contains(content.Content, "permission denied"):
                // Handle permission errors
                log.Printf("Tool access denied: %s (ID: %s)", 
                    content.Content, content.ToolUseID)
                return nil, fmt.Errorf("permission denied for tool: %s", content.ToolUseID)
                
            case strings.Contains(content.Content, "rate limit"):
                // Handle rate limiting
                log.Printf("Rate limit exceeded for tool: %s", content.ToolUseID)
                // Implement exponential backoff or retry logic
                time.Sleep(time.Second * 2)
                return nil, ErrRateLimit
                
            case strings.Contains(content.Content, "invalid input"):
                // Handle invalid input errors
                log.Printf("Invalid tool input: %s (ID: %s)", 
                    content.Content, content.ToolUseID)
                return nil, fmt.Errorf("invalid input for tool: %s", content.ToolUseID)
                
            default:
                // Handle generic errors
                log.Printf("Tool execution error: %s (ID: %s)", 
                    content.Content, content.ToolUseID)
                return nil, fmt.Errorf("tool execution failed: %s", content.Content)
            }
            continue
        }

        // Process successful result
        log.Printf("Tool executed successfully (ID: %s)", content.ToolUseID)
        
        // Parse the tool result based on expected format
        var result interface{}
        if err := json.Unmarshal([]byte(content.Content), &result); err != nil {
            log.Printf("Failed to parse tool result: %v", err)
            continue
        }

        // Handle specific tool results
        switch content.ToolUseID {
        case "get_weather":
            if weather, ok := result.(map[string]interface{}); ok {
                // Process weather data
                temperature := weather["temperature"]
                conditions := weather["conditions"]
                // Use the weather data...
            }
            
        case "get_stock_price":
            if stockData, ok := result.(map[string]interface{}); ok {
                // Process stock price data
                price := stockData["price"]
                symbol := stockData["symbol"]
                // Use the stock data...
            }
            
        case "SearchInternet":
            if searchResult, ok := result.(map[string]interface{}); ok {
                // Process search results
                results := searchResult["results"]
                // Process search data...
            }
            
        default:
            log.Printf("Unknown tool ID: %s", content.ToolUseID)
        }
    }
}
