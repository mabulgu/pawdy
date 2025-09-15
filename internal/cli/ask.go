package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/mabulgu/pawdy/internal/app"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a one-shot question",
	Long: `Ask a single question and get an answer with context from your team documentation.
	
Examples:
  pawdy ask "How do I gather initramfs logs?"
  pawdy ask "What are the bare metal networking requirements?"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

func init() {
	rootCmd.AddCommand(askCmd)
	askCmd.Flags().Float64("temperature", 0, "override temperature for this question")
}

func runAsk(cmd *cobra.Command, args []string) error {
	// Join all arguments as the question
	question := strings.Join(args, " ")

	// Initialize the application
	pawdy, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Pawdy: %w", err)
	}
	defer pawdy.Close()

	ctx := context.Background()

	// Get temperature override from flags
	temperature, _ := cmd.Flags().GetFloat64("temperature")

	fmt.Printf("Question: %s\n\n", question)
	fmt.Print("ʕ•ᴥ•ʔ ")

	response, sources, err := pawdy.Ask(ctx, question, temperature)
	if err != nil {
		return fmt.Errorf("failed to get answer: %w", err)
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

	return nil
}
