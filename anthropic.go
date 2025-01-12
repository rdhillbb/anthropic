package anthropic

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "github.com/rdhillbb/anthropic/logging"
)

// Package-level logging control functions allow users to enable/disable logging
// throughout the entire Anthropic client implementation.
func StartMessageLogging() error {
    return logging.EnableLogging()
}

func StopMessageLogging() {
    logging.DisableLogging()
}

// Internal logging helpers ensure consistent log formatting and conditional logging
func logMessage(format string, args ...interface{}) {
    if logging.IsLoggingEnabled() {
        msg := fmt.Sprintf(format, args...)
        logging.WriteLogs(fmt.Sprintf("[Anthropic] %s", msg))
    }
}

func logJSON(prefix string, data interface{}) {
    if logging.IsLoggingEnabled() {
        jsonBytes, err := json.MarshalIndent(data, "", "  ")
        if err != nil {
            logMessage("%s: failed to marshal JSON: %v", prefix, err)
            return
        }
        logging.WriteLogs(fmt.Sprintf("[Anthropic] %s: %s", prefix, string(jsonBytes)))
    }
}

// System prompt management methods
func (c *AnthropicClient) UpdateSystemPrompt(prompt string) {
    logMessage("Updating system prompt")
    c.systemPrompt = prompt
    logJSON("Updated client configuration", map[string]interface{}{
        "systemPrompt": prompt,
    })
}

func (c *AnthropicClient) GetSystemPrompt() string {
    return c.systemPrompt
}

// Client option functions that configure the AnthropicClient

func WithMaxConversationLength(length int) ClientOption {
    return func(c *AnthropicClient) {
        if length > 0 {
            c.maxConvLength = length
        }
    }
}

func WithDefaultParams(params MessageParams) ClientOption {
    return func(c *AnthropicClient) {
        c.defaultParams = params
    }
}

func WithHTTPClient(client *http.Client) ClientOption {
    return func(c *AnthropicClient) {
        if client != nil {
            c.httpClient = client
        }
    }
}

func WithSystemPrompt(prompt string) ClientOption {
    return func(c *AnthropicClient) {
        c.systemPrompt = prompt
    }
}

// NewClient creates a new AnthropicClient with the provided API key and options.
// It initializes the client with logging enabled if configured.
func NewClient(apiKey string, opts ...ClientOption) *AnthropicClient {
    logMessage("Creating new AnthropicClient")
    client := &AnthropicClient{
        apiKey:       apiKey,
        httpClient:   &http.Client{},
        systemPrompt: defaultSystemPrompt,
        defaultParams: MessageParams{
            Tools: GetDefaultTools(),
            ToolChoice: &ToolChoice{
                Type: ToolChoiceAuto,
            },
        },
    }
    
    for _, opt := range opts {
        opt(client)
    }
    
    logJSON("Client configuration", map[string]interface{}{
        "maxConvLength": client.maxConvLength,
        "hasDefaults":   len(client.defaultParams.Tools) > 0 || 
                        client.defaultParams.MaxTokens > 0 ||
                        client.defaultParams.Model != "",
        "hasSystemPrompt": client.systemPrompt != defaultSystemPrompt,
        "availableTools": len(client.defaultParams.Tools),
    })
    return client
}

// sendRequest handles all HTTP communication with the Anthropic API.
// It includes comprehensive logging of requests, responses, and errors.
func (c *AnthropicClient) sendRequest(ctx context.Context, reqBody Request) (*AnthropicResponse, error) {
    logMessage("Preparing API request")
    logJSON("Request payload", reqBody)

    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        logMessage("Error marshaling request: %v", err)
        return nil, fmt.Errorf("error marshaling request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", defaultAPIEndpoint, bytes.NewBuffer(jsonData))
    if err != nil {
        logMessage("Error creating HTTP request: %v", err)
        return nil, fmt.Errorf("error creating request: %w", err)
    }

    // Set required headers for Anthropic API
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("x-api-key", c.apiKey)

    logMessage("Sending request to Anthropic API")
    resp, err := c.httpClient.Do(req)
    if err != nil {
        logMessage("API request failed: %v", err)
        return nil, fmt.Errorf("error sending request: %w", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        logMessage("Error reading response body: %v", err)
        return nil, fmt.Errorf("error reading response: %w", err)
    }

    // Handle non-200 responses with proper error parsing
    if resp.StatusCode != http.StatusOK {
        logMessage("Received error response (status %d)", resp.StatusCode)
        var errorResp struct {
            Error struct {
                Type    string `json:"type"`
                Message string `json:"message"`
            } `json:"error"`
        }
        if err := json.Unmarshal(body, &errorResp); err != nil {
            logMessage("Failed to parse error response: %v", err)
            return nil, fmt.Errorf("error response status %d: %s", resp.StatusCode, body)
        }
        logMessage("API error: %s - %s", errorResp.Error.Type, errorResp.Error.Message)
        return nil, fmt.Errorf("API error: %s - %s", errorResp.Error.Type, errorResp.Error.Message)
    }

    var anthropicResp AnthropicResponse
    if err := json.Unmarshal(body, &anthropicResp); err != nil {
        logMessage("Error parsing response JSON: %v", err)
        return nil, fmt.Errorf("error parsing response: %w", err)
    }

    logJSON("API response", anthropicResp)
    return &anthropicResp, nil
}

// ChatMe handles a single message interaction while maintaining conversation history.
// It manages the conversation state and handles logging of the entire interaction.
func (c *AnthropicClient) ChatMe(ctx context.Context, message string, params *MessageParams) (*AnthropicResponse, error) {
    logMessage("Starting chat interaction with message: %s", message)
    
    content := []MessageContent{{
        Type: ContentTypeText,
        Text: message,
    }}

    // Add user message to conversation history
    c.addMessageToConversation(RoleUser, content)
    logMessage("Added user message to conversation")
    logJSON("Current conversation state", c.conversation)
    
    c.trimConversationHistory()

    // Use system prompt hierarchy: params > client > default
    systemPrompt := c.systemPrompt
    if params != nil && params.System != "" {
        systemPrompt = params.System
    }

    // Prepare request with complete message history
    reqBody := Request{
        Model:       params.Model,
        System:      systemPrompt,
        Messages:    c.conversation,
        MaxTokens:   params.MaxTokens,
        Temperature: params.Temperature,
        TopP:        params.TopP,
        TopK:        params.TopK,
        Tools:       params.Tools,
        ToolChoice:  params.ToolChoice,
    }

    // Send request and handle any errors
    response, err := c.sendRequest(ctx, reqBody)
    if err != nil {
        logMessage("Chat request failed: %v", err)
        return nil, err
    }

    // Process and store assistant's response
    if len(response.Content) > 0 {
        logMessage("Adding assistant response to conversation")
        c.addMessageToConversation(RoleAssistant, response.Content)
        c.trimConversationHistory()
        logJSON("Updated conversation state", c.conversation)
    }

    return response, nil
}

// Conversation management methods with logging

func (c *AnthropicClient) addMessageToConversation(role string, content []MessageContent) {
    logMessage("Adding message to conversation (role: %s)", role)
    c.conversation = append(c.conversation, Message{
        Role:    role,
        Content: content,
    })
}

func (c *AnthropicClient) trimConversationHistory() {
    if c.maxConvLength > 0 && len(c.conversation) > c.maxConvLength {
        logMessage("Trimming conversation to max length: %d", c.maxConvLength)
        c.conversation = c.conversation[len(c.conversation)-c.maxConvLength:]
    }
}
