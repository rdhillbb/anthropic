# Anthropic ChatWithTools Quick Start Guide

## 1. Create a New Project

First, create a new Go project and install the required dependencies:

```bash
mkdir anthropic-chat
cd anthropic-chat
go mod init anthropic-chat
go get anthropic
```

## 2. Create Your Main File (main.go)

```go
package main

import (
    "bufio"
    "context"
    "flag"
    "fmt"
    "os"
    "strings"
    "anthropic"
)

const defaultModel = "claude-3-5-sonnet-20241022"

func main() {
    // Get API key from command line
    apiKey := flag.String("api-key", "", "Anthropic API key")
    flag.Parse()

    if *apiKey == "" {
        fmt.Println("Error: API key is required")
        flag.Usage()
        os.Exit(1)
    }

    // Initialize client
    client := anthropic.NewClient(*apiKey,
        anthropic.WithSystemPrompt(getSystemPrompt()),
        anthropic.WithDefaultParams(anthropic.MessageParams{
            Model:      defaultModel,
            MaxTokens:  8000,
            Tools:      GetDefaultTools(),
            ToolChoice: &anthropic.ToolChoice{Type: anthropic.ToolChoiceAuto},
        }),
    )

    // Start chat loop
    runChat(client)
}

// System prompt definition
func getSystemPrompt() string {
    return `You are an AI assistant with access to weather and stock information.
    When asked about weather or stocks, use the appropriate tools to get real-time data.`
}

// Chat loop implementation
func runChat(client *anthropic.AnthropicClient) {
    handlers := GetDefaultHandlers()
    scanner := bufio.NewScanner(os.Stdin)
    ctx := context.Background()

    fmt.Println("Chat initialized. Type 'exit' to quit.")
    fmt.Print("> ")

    for scanner.Scan() {
        input := strings.TrimSpace(scanner.Text())
        if input == "exit" {
            break
        }

        response, err := client.ChatWithTools(
            ctx,
            input,
            nil, // Use default params
            handlers,
        )

        if err != nil {
            fmt.Printf("Error: %v\n", err)
            fmt.Print("> ")
            continue
        }

        fmt.Println("\nAssistant:")
        for _, content := range response.Content {
            if content.Type == anthropic.ContentTypeText {
                fmt.Println(content.Text)
            }
        }
        fmt.Print("\n> ")
    }
}
```

## 3. Create Tools File (tools.go)

```go
package main

import (
    "context"
    "encoding/json"
    "anthropic"
)

// Define weather tool
func GetWeather() anthropic.Tool {
    return anthropic.Tool{
        Name: "get_weather",
        Description: "Get current weather for a location",
        InputSchema: anthropic.InputSchema{
            Type: "object",
            Properties: map[string]anthropic.Property{
                "location": {
                    Type:        "string",
                    Description: "Location name (city, country)",
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

// Define stock tool
func GetStock() anthropic.Tool {
    return anthropic.Tool{
        Name: "get_stock_price",
        Description: "Get current stock price",
        InputSchema: anthropic.InputSchema{
            Type: "object",
            Properties: map[string]anthropic.Property{
                "symbol": {
                    Type:        "string",
                    Description: "Stock symbol (e.g., AAPL)",
                },
            },
            Required: []string{"symbol"},
        },
    }
}

// Combine tools into default set
func GetDefaultTools() []anthropic.Tool {
    return []anthropic.Tool{
        GetWeather(),
        GetStock(),
    }
}

// Implement weather handler
func HandleWeather(ctx context.Context, args json.RawMessage) (string, error) {
    var params struct {
        Location string `json:"location"`
        Unit     string `json:"unit"`
    }
    if err := json.Unmarshal(args, &params); err != nil {
        return "", err
    }

    // Simulated weather data (replace with actual API call)
    weather := map[string]interface{}{
        "temperature_c": 22,
        "temperature_f": 71,
        "condition":    "sunny",
        "location":     params.Location,
    }
    
    jsonBytes, err := json.Marshal(weather)
    if err != nil {
        return "", err
    }
    return string(jsonBytes), nil
}

// Implement stock handler
func HandleStock(ctx context.Context, args json.RawMessage) (string, error) {
    var params struct {
        Symbol string `json:"symbol"`
    }
    if err := json.Unmarshal(args, &params); err != nil {
        return "", err
    }

    // Simulated stock data (replace with actual API call)
    stock := map[string]interface{}{
        "symbol": params.Symbol,
        "price":  "150.00",
    }
    
    jsonBytes, err := json.Marshal(stock)
    if err != nil {
        return "", err
    }
    return string(jsonBytes), nil
}

// Create handler map
func GetDefaultHandlers() map[string]func(context.Context, json.RawMessage) (string, error) {
    return map[string]func(context.Context, json.RawMessage) (string, error){
        "get_weather":     HandleWeather,
        "get_stock_price": HandleStock,
    }
}
```

## 4. Run the Application

```bash
go run . -api-key="your-api-key-here"
```

Example usage:
```
Chat initialized. Type 'exit' to quit.
> What's the weather in London?
Assistant: Let me check the weather in London for you...
[Weather information appears here]

> What's the stock price for AAPL?
Assistant: Let me check Apple's stock price...
[Stock information appears here]

> exit
```

## Next Steps

1. Replace the simulated data in handlers with real API calls
2. Add error handling for API failures
3. Add more tools as needed
4. Customize the system prompt for your use case

## Common Issues

1. Make sure your API key is valid
2. Ensure all imports are correct
3. Check that tool names match between definitions and handlers
4. Verify JSON responses are properly formatted

That's it! You now have a working chat application with tool support. The assistant can use the weather and stock tools to provide real-time information in response to user queries.
