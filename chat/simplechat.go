package main

import (
	"anthropic" // Import the local package
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Initialize the client with your API key
	client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))

	// Optional: Set a custom system prompt
	client.UpdateSystemPrompt("You are a helpful assistant.")

	// Create a scanner for reading user input
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Chat started. Type 'exit' to quit.")

	// Start chat loop
	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "exit" {
			break
		}

		// Create message parameters
		params := &anthropic.MessageParams{
			Model:       "claude-3-5-sonnet-20241022", // Latest Claude model
			MaxTokens:   1000,                         // Adjust as needed
			Temperature: 0.7,                          // Adjust for creativity vs determinism
		}

		// Send message to Claude
		response, err := client.ChatMe(context.Background(), input, params)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Print assistant's response
		if len(response.Content) > 0 {
			fmt.Print("Assistant: ")
			for _, content := range response.Content {
				if content.Type == anthropic.ContentTypeText {
					fmt.Println(content.Text)
				}
			}
		}
	}

	fmt.Println("\nChat ended.")
}
