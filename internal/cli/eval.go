package cli

import (
	"context"
	"fmt"

	"github.com/mabulgu/pawdy/internal/app"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate model performance against test set",
	Long: `Evaluate the model's performance against a test dataset. The test file should 
contain questions and expected answers in JSONL format. This helps measure the 
quality of responses and RAG performance.`,
	RunE: runEval,
}

func init() {
	rootCmd.AddCommand(evalCmd)
	evalCmd.Flags().String("test-file", "eval.jsonl", "path to test file in JSONL format")
	evalCmd.Flags().String("output", "", "output file for detailed results")
}

func runEval(cmd *cobra.Command, args []string) error {
	testFile, _ := cmd.Flags().GetString("test-file")
	outputFile, _ := cmd.Flags().GetString("output")

	// Initialize the application
	pawdy, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Pawdy: %w", err)
	}
	defer pawdy.Close()

	ctx := context.Background()
	
	fmt.Printf("📊 Running evaluation with test file: %s\n", testFile)
	
	results, err := pawdy.Evaluate(ctx, testFile, outputFile)
	if err != nil {
		return fmt.Errorf("evaluation failed: %w", err)
	}

	fmt.Println("\n📈 Evaluation Results:")
	fmt.Println("═════════════════════")
	fmt.Printf("Questions processed: %d\n", results.Total)
	fmt.Printf("Average response time: %.2fs\n", results.AvgResponseTime)
	fmt.Printf("Average relevance score: %.3f\n", results.AvgRelevanceScore)
	
	if results.SafetyBlocks > 0 {
		fmt.Printf("Safety blocks: %d\n", results.SafetyBlocks)
	}
	
	if outputFile != "" {
		fmt.Printf("\n💾 Detailed results saved to: %s\n", outputFile)
	}

	return nil
}
