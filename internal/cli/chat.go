package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mabulgu/pawdy/internal/app"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with Pawdy. Type your questions and 
get answers with context from your team documentation. Use 'exit' or 'quit' to end the session.`,
	RunE: runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.Flags().Float64("temperature", 0, "override temperature for this session")
}

func runChat(cmd *cobra.Command, args []string) error {
	// Initialize the application
	pawdy, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Pawdy: %w", err)
	}
	defer pawdy.Close()

	// Print backend information
	fmt.Printf("Backend: %s\n", pawdy.Config.Backend)
	if pawdy.Config.Backend == "llamacpp" {
		fmt.Printf("Model: %s\n", pawdy.Config.ModelPath)
	} else {
		fmt.Printf("Ollama URL: %s\n", pawdy.Config.OllamaURL)
	}
	fmt.Printf("Safety: %s\n", pawdy.Config.Safety)
	fmt.Println("\nType your questions (or 'exit'/'quit' to end):")
	fmt.Println("─────────────────────────────────────────────")

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("\n >")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("\n👋 Goodbye!")
			break
		}

		fmt.Print("ʕ•ᴥ•ʔ ")

		// Get temperature override from flags
		temperature, _ := cmd.Flags().GetFloat64("temperature")

		response, sources, err := pawdy.Ask(ctx, input, temperature)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		fmt.Println(response)

		// Print sources if any
		if len(sources) > 0 {
			fmt.Println("\n📚 Sources:")
			for i, source := range sources {
				fmt.Printf("  [%d] %s (score: %.3f)\n", i+1,
					getSourceTitle(source), source.Score)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

func getSourceTitle(source *app.Source) string {
	if title, ok := source.Metadata["title"].(string); ok && title != "" {
		return title
	}
	if path, ok := source.Metadata["path"].(string); ok && path != "" {
		return path
	}
	return fmt.Sprintf("Document %s", source.ID)
}
