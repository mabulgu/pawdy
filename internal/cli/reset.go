package cli

import (
	"context"
	"fmt"

	"github.com/mabulgu/pawdy/internal/app"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the vector database",
	Long: `Reset the vector database by deleting all indexed documents. This will 
remove all ingested content and you'll need to run 'pawdy ingest' again to 
re-index your documents.`,
	RunE: runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().String("collection", "", "specific collection to reset (default: use config)")
	resetCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
}

func runReset(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	
	if !force {
		fmt.Print("⚠️  This will delete all indexed documents. Continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println("Reset cancelled.")
			return nil
		}
	}

	// Initialize the application
	pawdy, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Pawdy: %w", err)
	}
	defer pawdy.Close()

	ctx := context.Background()
	
	collection, _ := cmd.Flags().GetString("collection")
	
	fmt.Println("🗑️  Resetting vector database...")
	
	err = pawdy.Reset(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	fmt.Println("✅ Vector database reset successfully!")
	fmt.Println("💡 Run 'pawdy ingest ./materials' to re-index your documents")

	return nil
}
