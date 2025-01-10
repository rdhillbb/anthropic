package main

import (
    "bufio"
    "context"
    "flag"
    "fmt"
    "os"
    "strings"
    "github.com/rdhillbb/messagefile"    
    "anthropic"
)

const defaultModel = "claude-3-5-sonnet-20241022"

func main() {
    apiKey := flag.String("api-key", "", "Anthropic API key")
    debug := flag.Bool("debug", false, "Enable debug logging")
    flag.Parse()

    if *apiKey == "" {
        fmt.Println("Error: API key is required")
        flag.Usage()
        os.Exit(1)
    }

    if *debug {
        anthropic.EnableDebug()
        defer anthropic.DisableDebug()
    }

    // Get system prompt and handle potential error
    systemPrompt, err := messagefile.GetMSG("systemprompt:systempromptmain")
    if err != nil {
        fmt.Printf("Warning: Failed to load system prompt: %v\nUsing default prompt\n", err)
        systemPrompt = "You are Mr. PeeBody, an expert search agent." // Fallback prompt
    }

    client := anthropic.NewClient(*apiKey, 
        anthropic.WithSystemPrompt(systemPrompt),  // Add system prompt here
        anthropic.WithDefaultParams(anthropic.MessageParams{
            Model:      defaultModel,
            MaxTokens:  8000,
            Tools:      GetDefaultTools(),
            ToolChoice: &anthropic.ToolChoice{Type: anthropic.ToolChoiceAuto},
        }),
        anthropic.WithMaxConversationLength(10),
    )

    handlers := GetDefaultHandlers()
    scanner := bufio.NewScanner(os.Stdin)
    ctx := context.Background()

    fmt.Println("Chat initialized with tools. Type 'exit' to quit.")
    fmt.Println("Available tools:")
    for _, tool := range GetDefaultTools() {
        fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
    }
    fmt.Println("\nEnter your message:")

    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "exit" {
            break
        }

        if input == "" {
            continue
        }

        response, err := client.ChatWithTools(
            ctx,
            input,
            &anthropic.MessageParams{
                Model:      defaultModel,
                MaxTokens:  8000,
                Tools:      GetDefaultTools(),
                ToolChoice: &anthropic.ToolChoice{Type: anthropic.ToolChoiceAuto},
            },
            handlers,
        )

        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }

        fmt.Println("\nAssistant:")
        for _, content := range response.Content {
            if content.Type == anthropic.ContentTypeText {
                fmt.Println(content.Text)
            }
        }
        fmt.Println()
    }

    if err := scanner.Err(); err != nil {
        fmt.Printf("Error reading input: %v\n", err)
        os.Exit(1)
    }
}
