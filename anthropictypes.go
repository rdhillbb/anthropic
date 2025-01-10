package anthropic

import (
    "encoding/json"
    "net/http"
)

// Core API configuration constants
const (
    defaultAPIEndpoint = "https://api.anthropic.com/v1/messages"
    defaultModel      = "claude-3-5-sonnet-20241022"
    defaultSystemPrompt = `You are Mr. PeeBody. You are an expert search agent. If the user requests research, you are to search the internet if you do not have the information available. You have access to the following tools:

1. 'get_weather'
   - Gets current weather for a specified location
   - Requires location parameter (e.g., city, country, region)
   - Optional unit parameter (celsius or fahrenheit)
   - Returns temperature, conditions, and humidity in both Celsius and Fahrenheit

2. 'get_stock_price'
   - Gets current stock price for a given stock symbol
   - Requires stock symbol parameter (e.g., AAPL)

3. 'SearchInternet'
   - Performs a general internet search for information
   - Requires a search query parameter

4. 'DeepSearch'
   - Performs a more comprehensive, detailed search analysis
   - Requires a search query parameter
   - Best used when detailed analysis is needed

Any request that you get from the user, you are to develop a step by step plan to execute. Once Developed, you are to review and execute.`
)

// Role and content type constants
const (
    RoleSystem    = "system"
    RoleUser      = "user"
    RoleAssistant = "assistant"
    
    ContentTypeText       = "text"
    ContentTypeToolUse    = "tool_use"
    ContentTypeToolResult = "tool_result"
    ContentTypeThinking   = "thinking"  
    
    StopReasonToolUse      = "tool_use"
    StopReasonEndTurn      = "end_turn"
    StopReasonMaxTokens    = "max_tokens"
    StopReasonStopSequence = "stop_sequence"  
    
    ToolChoiceAuto = "auto"
    ToolChoiceNone = "none"
    ToolChoiceTool = "tool"
)

// ClientOption defines functions that can modify client configuration
type ClientOption func(*AnthropicClient)

// AnthropicClient handles communication with the Anthropic API
type AnthropicClient struct {
    apiKey          string
    defaultParams   MessageParams
    httpClient      *http.Client
    conversation    []Message
    maxConvLength   int
    systemPrompt    string    // System prompt that defines assistant behavior
}

// Message represents a single message in the conversation
type Message struct {
    Role    string           `json:"role"`    
    Content []MessageContent `json:"content"` 
}

// MessageContent represents different types of content within a message
type MessageContent struct {
    Type       string          `json:"type"`               
    Text       string          `json:"text,omitempty"`     
    ID         string          `json:"id,omitempty"`       
    Name       string          `json:"name,omitempty"`     
    Input      json.RawMessage `json:"input,omitempty"`    
    ToolUseID  string          `json:"tool_use_id,omitempty"`  
    Content    string          `json:"content,omitempty"`      
    IsError    bool            `json:"is_error,omitempty"`     
}

// ToolUse represents a tool call from the assistant
type ToolUse struct {
    ID    string          `json:"id"`
    Name  string          `json:"name"`
    Input json.RawMessage `json:"input"`
}

// MessageParams contains all possible parameters for a message request
type MessageParams struct {
    Model       string                 `json:"model"`
    MaxTokens   int                    `json:"max_tokens"`
    Temperature float64                `json:"temperature,omitempty"`
    TopP        float64                `json:"top_p,omitempty"`
    TopK        int                    `json:"top_k,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    System      string                 `json:"system,omitempty"`
    Tools       []Tool                 `json:"tools,omitempty"`
    ToolChoice  *ToolChoice            `json:"tool_choice,omitempty"`
}

// Request represents the complete structure sent to the Anthropic API
type Request struct {
    Model       string      `json:"model"`
    Messages    []Message   `json:"messages"`
    MaxTokens   int         `json:"max_tokens"`
    Temperature float64     `json:"temperature,omitempty"`
    TopP        float64     `json:"top_p,omitempty"`
    TopK        int         `json:"top_k,omitempty"`
    System      string      `json:"system,omitempty"`
    Tools       []Tool      `json:"tools,omitempty"`
    ToolChoice  *ToolChoice `json:"tool_choice,omitempty"`
}

// Tool-related types
type Tool struct {
    Name         string      `json:"name"`
    Description  string      `json:"description"`
    InputSchema  InputSchema `json:"input_schema"`
}

type InputSchema struct {
    Type       string              `json:"type"`
    Properties map[string]Property `json:"properties"`
    Required   []string           `json:"required"`
}

type Property struct {
    Type        string   `json:"type"`
    Description string   `json:"description"`
    Enum        []string `json:"enum,omitempty"`
}

type ToolChoice struct {
    Type string `json:"type"`
    Name string `json:"name,omitempty"`
}

// Response types
type AnthropicResponse struct {
    ID          string           `json:"id"`
    Type        string           `json:"type"`
    Role        string           `json:"role"`
    Content     []MessageContent `json:"content"`
    Model       string           `json:"model"`
    StopReason  string           `json:"stop_reason"`
    Usage       Usage            `json:"usage"`
}

type Usage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
}

// GetDefaultTools returns the default set of tools available to Mr. PeeBody
func GetDefaultTools() []Tool {
    return []Tool{
        {
            Name:        "get_weather",
            Description: "Gets current weather for a specified location",
            InputSchema: InputSchema{
                Type: "object",
                Properties: map[string]Property{
                    "location": {
                        Type:        "string",
                        Description: "Location (city, country, or region)",
                    },
                    "unit": {
                        Type:        "string",
                        Description: "Temperature unit (celsius or fahrenheit)",
                        Enum:        []string{"celsius", "fahrenheit"},
                    },
                },
                Required: []string{"location"},
            },
        },
        {
            Name:        "get_stock_price",
            Description: "Gets current stock price for a given stock symbol",
            InputSchema: InputSchema{
                Type: "object",
                Properties: map[string]Property{
                    "symbol": {
                        Type:        "string",
                        Description: "Stock symbol (e.g., AAPL)",
                    },
                },
                Required: []string{"symbol"},
            },
        },
        {
            Name:        "SearchInternet",
            Description: "Performs a general internet search for information",
            InputSchema: InputSchema{
                Type: "object",
                Properties: map[string]Property{
                    "query": {
                        Type:        "string",
                        Description: "Search query",
                    },
                },
                Required: []string{"query"},
            },
        },
        {
            Name:        "DeepSearch",
            Description: "Performs a more comprehensive, detailed search analysis",
            InputSchema: InputSchema{
                Type: "object",
                Properties: map[string]Property{
                    "query": {
                        Type:        "string",
                        Description: "Search query for detailed analysis",
                    },
                },
                Required: []string{"query"},
            },
        },
    }
}
