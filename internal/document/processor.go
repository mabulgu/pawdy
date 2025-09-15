// Package document provides document processing and chunking functionality.
package document

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/mabulgu/pawdy/pkg/types"
)

// Processor handles document parsing and chunking.
type Processor struct {
	chunkTokens  int
	chunkOverlap int
}

// NewProcessor creates a new document processor.
func NewProcessor(chunkTokens, chunkOverlap int) *Processor {
	return &Processor{
		chunkTokens:  chunkTokens,
		chunkOverlap: chunkOverlap,
	}
}

// Process extracts text content from a document and splits it into chunks.
func (p *Processor) Process(ctx context.Context, reader io.Reader, source types.DocumentSource) ([]*types.Document, error) {
	var text string
	var err error

	// Handle PDF files specially (require file path)
	if strings.ToLower(source.Type) == ".pdf" {
		text, err = p.extractPDF(source.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract PDF text: %w", err)
		}
	} else {
		// Read all content for other file types
		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read document: %w", err)
		}

		// Extract text based on file type
		text, err = p.extractText(string(content), source.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to extract text: %w", err)
		}
	}

	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("document contains no extractable text")
	}

	// Split into chunks
	chunks := p.chunkText(text, p.chunkTokens, p.chunkOverlap)

	// Create document objects
	documents := make([]*types.Document, len(chunks))
	for i, chunk := range chunks {
		docID := fmt.Sprintf("%x-%d", md5.Sum([]byte(source.Path)), i)

		documents[i] = &types.Document{
			ID:      docID,
			Content: chunk,
			Metadata: map[string]any{
				"path":         source.Path,
				"title":        source.Title,
				"type":         source.Type,
				"size":         source.Size,
				"modified":     source.Modified,
				"chunk_id":     i,
				"total_chunks": len(chunks),
			},
		}
	}

	return documents, nil
}

// SupportedTypes returns the file types this processor can handle.
func (p *Processor) SupportedTypes() []string {
	return []string{".md", ".txt", ".html", ".pdf"}
}

// extractText extracts plain text from various document formats.
func (p *Processor) extractText(content, fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case ".md", ".markdown":
		return p.extractMarkdown(content), nil
	case ".txt":
		return content, nil
	case ".html", ".htm":
		return p.extractHTML(content), nil
	default:
		// Treat as plain text
		return content, nil
	}
}

// extractPDF extracts text from PDF files.
func (p *Processor) extractPDF(filePath string) (string, error) {
	file, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	var text strings.Builder
	totalPages := r.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// Extract text from the page with empty font map
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		text.WriteString(pageText)
		text.WriteString("\n") // Add newline between pages
	}

	result := text.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("no text could be extracted from PDF")
	}

	// Clean up excessive whitespace that's common in PDF extraction
	// Replace multiple spaces with single spaces
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.TrimSpace(result)

	return result, nil
}

// extractMarkdown removes markdown formatting while preserving structure.
func (p *Processor) extractMarkdown(content string) string {
	text := content

	// Remove code blocks (preserve content but remove formatting)
	codeBlockRe := regexp.MustCompile("(?s)```[a-zA-Z]*\n(.*?)\n```")
	text = codeBlockRe.ReplaceAllString(text, "$1")

	// Remove inline code formatting
	inlineCodeRe := regexp.MustCompile("`([^`]+)`")
	text = inlineCodeRe.ReplaceAllString(text, "$1")

	// Remove links but keep text
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	text = linkRe.ReplaceAllString(text, "$1")

	// Remove image syntax
	imageRe := regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	text = imageRe.ReplaceAllString(text, "")

	// Remove headers but keep content
	headerRe := regexp.MustCompile(`^#{1,6}\s+(.+)$`)
	text = headerRe.ReplaceAllStringFunc(text, func(match string) string {
		return headerRe.ReplaceAllString(match, "$1")
	})

	// Remove bold/italic formatting
	boldItalicRe := regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`)
	text = boldItalicRe.ReplaceAllString(text, "$1")

	// Remove strikethrough
	strikeRe := regexp.MustCompile(`~~([^~]+)~~`)
	text = strikeRe.ReplaceAllString(text, "$1")

	// Clean up multiple whitespace
	whitespaceRe := regexp.MustCompile(`\s+`)
	text = whitespaceRe.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// extractHTML removes HTML tags and extracts text content.
func (p *Processor) extractHTML(content string) string {
	// Remove script and style tags completely
	scriptRe := regexp.MustCompile(`(?i)<(script|style)[^>]*>.*?</\1>`)
	text := scriptRe.ReplaceAllString(content, "")

	// Remove HTML tags but preserve content
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text = tagRe.ReplaceAllString(text, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Clean up multiple whitespace
	whitespaceRe := regexp.MustCompile(`\s+`)
	text = whitespaceRe.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// chunkText splits text into overlapping chunks based on approximate token count.
func (p *Processor) chunkText(text string, maxTokens, overlap int) []string {
	// Rough approximation: 1 token ≈ 4 characters for English text
	maxChars := maxTokens * 4
	overlapChars := overlap * 4

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var chunks []string
	currentChunk := ""

	for _, word := range words {
		// Check if adding this word would exceed the chunk size
		testChunk := currentChunk
		if testChunk != "" {
			testChunk += " "
		}
		testChunk += word

		if len(testChunk) > maxChars && currentChunk != "" {
			// Save current chunk
			chunks = append(chunks, strings.TrimSpace(currentChunk))

			// Start new chunk with overlap
			if overlapChars > 0 && len(chunks) > 0 {
				currentChunk = p.getOverlapText(currentChunk, overlapChars) + " " + word
			} else {
				currentChunk = word
			}
		} else {
			currentChunk = testChunk
		}
	}

	// Add the final chunk if it has content
	if strings.TrimSpace(currentChunk) != "" {
		chunks = append(chunks, strings.TrimSpace(currentChunk))
	}

	return chunks
}

// getOverlapText returns the last N characters of text for overlap.
func (p *Processor) getOverlapText(text string, overlapChars int) string {
	if len(text) <= overlapChars {
		return text
	}

	// Find a good break point (word boundary) near the overlap size
	startPos := len(text) - overlapChars

	// Look for the nearest word boundary
	for j := startPos; j < len(text); j++ {
		if text[j] == ' ' {
			return strings.TrimSpace(text[j:])
		}
	}

	// If no word boundary found, just truncate
	return strings.TrimSpace(text[startPos:])
}

// ProcessFile processes a single file and returns document chunks.
func ProcessFile(ctx context.Context, filePath string, chunkTokens, chunkOverlap int) ([]*types.Document, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create document source
	source := types.DocumentSource{
		Path:     filePath,
		Title:    extractTitle(filePath),
		Size:     fileInfo.Size(),
		Modified: fileInfo.ModTime(),
		Type:     filepath.Ext(filePath),
	}

	// Create processor
	processor := NewProcessor(chunkTokens, chunkOverlap)

	// Process the document
	return processor.Process(ctx, file, source)
}

// extractTitle attempts to extract a meaningful title from the file path or content.
func extractTitle(filePath string) string {
	// Get filename without extension
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Convert to title case and replace separators
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Capitalize words
	words := strings.Fields(name)
	for j, word := range words {
		if len(word) > 0 {
			words[j] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// CountTokens provides a rough estimate of token count for text.
func CountTokens(text string) int {
	// Rough approximation: 1 token ≈ 4 characters for English text
	// This is a simplified approach - real tokenization would use the model's tokenizer
	return len(text) / 4
}
