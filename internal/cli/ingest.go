package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mabulgu/pawdy/internal/app"
	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest [directory]",
	Short: "Ingest documents from a directory",
	Long: `Ingest and index documents from the specified directory. Supports Markdown (.md), 
plain text (.txt), PDF (.pdf), and HTML (.html) files. Documents are chunked, embedded, 
and stored in the vector database for retrieval.`,
	Args: cobra.ExactArgs(1),
	RunE: runIngest,
}

func init() {
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.Flags().Int("chunk-size", 0, "override chunk size in tokens")
	ingestCmd.Flags().Int("overlap", 0, "override chunk overlap in tokens")
}

func runIngest(cmd *cobra.Command, args []string) error {
	directory := args[0]

	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", directory)
	}

	// Initialize the application
	pawdy, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Pawdy: %w", err)
	}
	defer pawdy.Close()

	// Get override values from flags
	chunkSize, _ := cmd.Flags().GetInt("chunk-size")
	overlap, _ := cmd.Flags().GetInt("overlap")

	fmt.Printf("📂 Ingesting documents from: %s\n", directory)
	fmt.Println("Supported formats: .md, .txt, .html, .pdf")
	fmt.Println()

	ctx := context.Background()

	// Walk through directory and collect files
	var files []string
	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".txt" || ext == ".pdf" || ext == ".html" {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("⚠️  No supported files found in directory")
		return nil
	}

	fmt.Printf("📄 Found %d files to process\n\n", len(files))

	// Process files
	totalChunks := 0
	for i, file := range files {
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(files), filepath.Base(file))

		chunks, err := pawdy.IngestFile(ctx, file, chunkSize, overlap)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ Created %d chunks\n", chunks)
		totalChunks += chunks
	}

	fmt.Printf("\n🎉 Ingestion complete!\n")
	fmt.Printf("📊 Total files processed: %d\n", len(files))
	fmt.Printf("📊 Total chunks created: %d\n", totalChunks)
	fmt.Printf("📊 Embeddings generated: %d\n", totalChunks)

	return nil
}
