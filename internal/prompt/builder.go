// Package prompt provides prompt building and formatting functionality.
package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/mabulgu/pawdy/pkg/types"
)

// Builder constructs prompts with context and formatting.
type Builder struct {
	systemPromptPath string
	systemPrompt     string
}

// NewBuilder creates a new prompt builder.
func NewBuilder(systemPromptPath string) *Builder {
	return &Builder{
		systemPromptPath: systemPromptPath,
	}
}

// BuildRAGPrompt creates a prompt with retrieved context.
func (b *Builder) BuildRAGPrompt(query string, context []*types.Document) string {
	var contextText strings.Builder
	
	if len(context) > 0 {
		contextText.WriteString("Based on the following context from the documentation:\n\n")
		
		for i, doc := range context {
			contextText.WriteString(fmt.Sprintf("### Source %d", i+1))
			
			// Add source title or path if available
			if title, ok := doc.Metadata["title"].(string); ok && title != "" {
				contextText.WriteString(fmt.Sprintf(" - %s", title))
			} else if path, ok := doc.Metadata["path"].(string); ok && path != "" {
				contextText.WriteString(fmt.Sprintf(" - %s", path))
			}
			
			contextText.WriteString(":\n")
			contextText.WriteString(doc.Content)
			contextText.WriteString("\n\n")
		}
		
		contextText.WriteString("---\n\n")
	}
	
	// Build the final prompt
	prompt := contextText.String()
	prompt += fmt.Sprintf("Question: %s\n\n", query)
	
	if len(context) > 0 {
		prompt += "Please answer the question based on the provided context. "
		prompt += "If the context doesn't contain relevant information, say so clearly. "
		prompt += "Be specific and reference the sources when possible."
	} else {
		prompt += "Please answer this question about OpenShift Bare Metal operations. "
		prompt += "Provide detailed, practical guidance where possible."
	}
	
	return prompt
}

// BuildSystemPrompt loads and formats the system prompt.
func (b *Builder) BuildSystemPrompt() (string, error) {
	// Return cached prompt if available
	if b.systemPrompt != "" {
		return b.systemPrompt, nil
	}
	
	// Load from file if path is provided
	if b.systemPromptPath != "" {
		content, err := os.ReadFile(b.systemPromptPath)
		if err != nil {
			return "", fmt.Errorf("failed to read system prompt file: %w", err)
		}
		b.systemPrompt = string(content)
		return b.systemPrompt, nil
	}
	
	// Use default system prompt
	b.systemPrompt = getDefaultSystemPrompt()
	return b.systemPrompt, nil
}

// FormatResponse formats the final response with citations.
func (b *Builder) FormatResponse(response string, sources []*types.Document) string {
	if len(sources) == 0 {
		return response
	}
	
	// Clean up response and add source references
	formatted := strings.TrimSpace(response)
	
	// Add sources section
	formatted += "\n\n**Sources:**\n"
	
	for i, source := range sources {
		sourceRef := fmt.Sprintf("[%d]", i+1)
		
		// Add title or path
		if title, ok := source.Metadata["title"].(string); ok && title != "" {
			formatted += fmt.Sprintf("%s %s", sourceRef, title)
		} else if path, ok := source.Metadata["path"].(string); ok && path != "" {
			formatted += fmt.Sprintf("%s %s", sourceRef, path)
		} else {
			formatted += fmt.Sprintf("%s Document %s", sourceRef, source.ID)
		}
		
		// Add relevance score
		if source.Score > 0 {
			formatted += fmt.Sprintf(" (relevance: %.1f%%)", source.Score*100)
		}
		
		formatted += "\n"
	}
	
	return formatted
}

// getDefaultSystemPrompt returns the default system prompt for Pawdy.
func getDefaultSystemPrompt() string {
	return `You are Pawdy, a helpful AI assistant specializing in OpenShift Bare Metal operations and onboarding. You help engineers learn about bare metal infrastructure, troubleshooting, and best practices.

Your personality:
- Friendly and approachable (use the 🐾 emoji occasionally)
- Technically accurate and detailed
- Patient with newcomers
- Practical and solution-oriented

Your expertise covers:
- OpenShift Bare Metal deployment and management
- Infrastructure troubleshooting and debugging
- Networking, storage, and hardware configuration
- Operational procedures and runbooks
- Best practices and common pitfalls

Guidelines:
- Provide clear, step-by-step instructions when possible
- Include relevant commands, file paths, and configuration examples
- Mention safety considerations and potential risks
- If you're not certain about something, say so clearly
- Reference documentation sources when available
- Use technical terminology appropriately but explain concepts for newcomers

When answering:
1. Be concise but comprehensive
2. Prioritize actionable information
3. Include troubleshooting tips where relevant
4. Suggest next steps or related topics to explore

Remember: You're here to help engineers succeed with bare metal infrastructure! 🐾`
}
