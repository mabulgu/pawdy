package prompt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mabulgu/pawdy/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("./test_prompt.md")
	assert.NotNil(t, builder)
	assert.Equal(t, "./test_prompt.md", builder.systemPromptPath)
}

func TestBuilder_BuildRAGPrompt(t *testing.T) {
	builder := NewBuilder("")
	
	// Test with context
	docs := []*types.Document{
		{
			ID:      "doc1",
			Content: "OpenShift Bare Metal requires careful network configuration.",
			Metadata: map[string]any{
				"title": "Networking Guide",
				"path":  "/docs/networking.md",
			},
		},
		{
			ID:      "doc2", 
			Content: "Use metal3 for bare metal provisioning.",
			Metadata: map[string]any{
				"title": "Provisioning Guide",
			},
		},
	}
	
	prompt := builder.BuildRAGPrompt("How do I configure networking?", docs)
	
	assert.Contains(t, prompt, "Based on the following context")
	assert.Contains(t, prompt, "Source 1 - Networking Guide")
	assert.Contains(t, prompt, "Source 2 - Provisioning Guide")
	assert.Contains(t, prompt, "OpenShift Bare Metal requires")
	assert.Contains(t, prompt, "Question: How do I configure networking?")
	assert.Contains(t, prompt, "based on the provided context")
}

func TestBuilder_BuildRAGPrompt_NoContext(t *testing.T) {
	builder := NewBuilder("")
	
	prompt := builder.BuildRAGPrompt("What is OpenShift?", nil)
	
	assert.NotContains(t, prompt, "Based on the following context")
	assert.Contains(t, prompt, "Question: What is OpenShift?")
	assert.Contains(t, prompt, "OpenShift Bare Metal operations")
}

func TestBuilder_BuildSystemPrompt_File(t *testing.T) {
	// Create a temporary system prompt file
	tempDir := t.TempDir()
	promptFile := filepath.Join(tempDir, "system_prompt.md")
	testPrompt := "You are a test assistant."
	
	err := os.WriteFile(promptFile, []byte(testPrompt), 0644)
	require.NoError(t, err)
	
	builder := NewBuilder(promptFile)
	prompt, err := builder.BuildSystemPrompt()
	
	assert.NoError(t, err)
	assert.Equal(t, testPrompt, prompt)
	
	// Test caching
	prompt2, err := builder.BuildSystemPrompt()
	assert.NoError(t, err)
	assert.Equal(t, testPrompt, prompt2)
}

func TestBuilder_BuildSystemPrompt_Default(t *testing.T) {
	builder := NewBuilder("")
	prompt, err := builder.BuildSystemPrompt()
	
	assert.NoError(t, err)
	assert.Contains(t, prompt, "Pawdy")
	assert.Contains(t, prompt, "OpenShift Bare Metal")
	assert.Contains(t, prompt, "🐾")
}

func TestBuilder_BuildSystemPrompt_FileNotFound(t *testing.T) {
	builder := NewBuilder("/nonexistent/file.md")
	_, err := builder.BuildSystemPrompt()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read system prompt file")
}

func TestBuilder_FormatResponse(t *testing.T) {
	builder := NewBuilder("")
	
	sources := []*types.Document{
		{
			ID:    "doc1",
			Score: 0.85,
			Metadata: map[string]any{
				"title": "Network Configuration",
				"path":  "/docs/network.md",
			},
		},
		{
			ID:    "doc2",
			Score: 0.72,
			Metadata: map[string]any{
				"path": "/docs/troubleshooting.md",
			},
		},
	}
	
	response := "Configure the network using the metal3 operator."
	formatted := builder.FormatResponse(response, sources)
	
	assert.Contains(t, formatted, "Configure the network using the metal3 operator.")
	assert.Contains(t, formatted, "**Sources:**")
	assert.Contains(t, formatted, "[1] Network Configuration")
	assert.Contains(t, formatted, "[2] /docs/troubleshooting.md")
	assert.Contains(t, formatted, "relevance: 85.0%")
	assert.Contains(t, formatted, "relevance: 72.0%")
}

func TestBuilder_FormatResponse_NoSources(t *testing.T) {
	builder := NewBuilder("")
	
	response := "This is a response without sources."
	formatted := builder.FormatResponse(response, nil)
	
	assert.Equal(t, response, formatted)
	assert.NotContains(t, formatted, "**Sources:**")
}
