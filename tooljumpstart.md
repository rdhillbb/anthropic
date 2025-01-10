# Step-by-Step Guide: Setting Up Tools in Anthropic

## Step 1: Define Individual Tool Structures

First, create functions that return individual tool definitions. Create a new file called `tools.go`:

```go
package main

import "anthropic"

// Step 1a: Define the weather tool
func GetWeather() anthropic.Tool {
    return anthropic.Tool{
        Name: "get_weather",
        Description: "Get the current weather in a given location",
        InputSchema: anthropic.InputSchema{
            Type: "object",
            Properties: map[string]anthropic.Property{
                "location": {
                    Type:        "string",
                    Description: "The location name (city, country)",
                },
                "unit": {
                    Type:        "string",
                    Description: "Temperature unit",
                    Enum:        []string{"celsius", "fahrenheit"},
                },
            },
            Required: []string{"location"},
        },
    }
}

// Step 1b: Define the stock price tool
func GetStock() anthropic.Tool {
    return anthropic.Tool{
        Name: "get_stock_price",
        Description: "Get the current stock price",
        InputSchema: anthropic.InputSchema{
            Type: "object",
            Properties: map[string]anthropic.Property{
                "symbol": {
                    Type:        "string",
                    Description: "The stock symbol, e.g. AAPL",
                },
            },
            Required: []string{"symbol"},
        },
    }
}
```

## Step 2: Combine Tools into Default Set

Create a function that returns all available tools:

```go
// Step 2: Combine all tools
func GetDefaultTools() []anthropic.Tool {
    return []anthropic.Tool{
        GetWeather(),
        GetStock(),
        // Add more tools as needed
    }
}
```

## Step 3: Implement Tool Handlers

Create handlers for each tool that process the actual requests:

```go
package main

import (
    "context"
    "encoding/json"
)

// Step 3a: Implement weather handler
func HandleWeather(ctx context.Context, args json.RawMessage) (string, error) {
    // Parse the input parameters
    var params struct {
        Location string `json:"location"`
        Unit     string `json:"unit"`
    }
    if err := json.Unmarshal(args, &params); err != nil {
        return "", err
    }

    // Your weather API logic here
    weatherData := map[string]interface{}{
        "temperature_c": 22,
        "temperature_f": 71,
        "condition":    "sunny",
        "location":     params.Location,
    }
    
    // Return JSON string
    jsonBytes, err := json.Marshal(weatherData)
    if err != nil {
        return "", err
    }
    return string(jsonBytes), nil
}

// Step 3b: Implement stock handler
func HandleStock(ctx context.Context, args json.RawMessage) (string, error) {
    var params struct {
        Symbol string `json:"symbol"`
    }
    if err := json.Unmarshal(args, &params); err != nil {
        return "", err
    }
    
    // Your stock API logic here
    stockData := map[string]interface{}{
        "symbol": params.Symbol,
        "price":  "150.00",
    }
    
    jsonBytes, err := json.Marshal(stockData)
    if err != nil {
        return "", err
    }
    return string(jsonBytes), nil
}
```

## Step 4: Create Handler Map

Create a function that maps tool names to their handlers:

```go
// Step 4: Create handler map
func GetDefaultHandlers() map[string]func(context.Context, json.RawMessage) (string, error) {
    return map[string]func(context.Context, json.RawMessage) (string, error){
        "get_weather":     HandleWeather,
        "get_stock_price": HandleStock,
    }
}
```

## Step 5: Initialize Client with Tools

In your main application, initialize the Anthropic client with the tools:

```go
func main() {
    // Step 5a: Create client with tools
    client := anthropic.NewClient("your-api-key",
        anthropic.WithDefaultParams(anthropic.MessageParams{
            Model:      "claude-3-5-sonnet-20241022",
            MaxTokens:  8000,
            Tools:      GetDefaultTools(),
            ToolChoice: &anthropic.ToolChoice{Type: anthropic.ToolChoiceAuto},
        }),
    )

    // Step 5b: Set up handlers
    handlers := GetDefaultHandlers()
    
    // Step 5c: Use ChatWithTools
    response, err := client.ChatWithTools(
        context.Background(),
        "What's the weather like in London?",
        nil, // Use default params
        handlers,
    )
    
    // Step 5d: Process response
    if err != nil {
        log.Fatal(err)
    }
    
    for _, content := range response.Content {
        if content.Type == anthropic.ContentTypeText {
            fmt.Println(content.Text)
        }
    }
}
```

## Step 6: Full Directory Structure

Your project should now look like this:

```
project/
├── main.go           # Contains main() and client initialization
├── tools.go          # Contains tool definitions (GetWeather, GetStock, etc.)
├── handlers.go       # Contains tool handlers
└── go.mod           # Module definition
```

## Step 7: Testing Your Tools

Create a simple test to verify tool functionality:

```go
func TestWeatherTool(t *testing.T) {
    // Get the tool definition
    weatherTool := GetWeather()
    if weatherTool.Name != "get_weather" {
        t.Errorf("Expected tool name 'get_weather', got %s", weatherTool.Name)
    }

    // Test the handler
    ctx := context.Background()
    input := `{"location": "London", "unit": "celsius"}`
    result, err := HandleWeather(ctx, json.RawMessage(input))
    if err != nil {
        t.Errorf("Handler failed: %v", err)
    }

    // Verify result structure
    var weatherData map[string]interface{}
    if err := json.Unmarshal([]byte(result), &weatherData); err != nil {
        t.Errorf("Failed to parse result: %v", err)
    }
}
```

## Summary
1. Define individual tools with proper schemas
2. Combine tools into a default set
3. Implement handlers for each tool
4. Create a handler map
5. Initialize client with tools
6. Structure your project files
7. Test your implementation

This structure allows you to:
- Easily add new tools by creating new tool definitions and handlers
- Maintain clean separation between tool definitions and their implementations
- Test tools independently
- Scale your application with additional tools as needed
