package rag

import (
	"context"
	"testing"

	"github.com/mabulgu/pawdy/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmbeddingProvider is a mock implementation for testing
type MockEmbeddingProvider struct {
	mock.Mock
}

func (m *MockEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	args := m.Called(ctx, texts)
	return args.Get(0).([][]float32), args.Error(1)
}

func (m *MockEmbeddingProvider) GetDimensions() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockEmbeddingProvider) IsHealthy(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestOllamaEmbeddings_GetDimensions(t *testing.T) {
	embeddings := NewOllamaEmbeddings("http://localhost:11434", "nomic-embed-text")
	assert.Equal(t, 768, embeddings.GetDimensions())
}

func TestQdrantRetriever_NewQdrantRetriever(t *testing.T) {
	mockEmbeddings := &MockEmbeddingProvider{}
	mockEmbeddings.On("GetDimensions").Return(768)

	retriever, err := NewQdrantRetriever("http://localhost:6333", "test_collection", mockEmbeddings)
	
	// Note: This will fail in CI without Qdrant running, but shows the test structure
	if err != nil {
		t.Skip("Skipping test that requires Qdrant connection")
	}
	
	assert.NotNil(t, retriever)
	assert.Equal(t, "test_collection", retriever.collection)
}

func TestDocumentProcessing(t *testing.T) {
	// Test document creation and metadata handling
	doc := &types.Document{
		ID:      "test-doc-1",
		Content: "This is test content for the document.",
		Metadata: map[string]any{
			"title": "Test Document",
			"path":  "/test/doc.md",
			"type":  ".md",
		},
	}

	assert.Equal(t, "test-doc-1", doc.ID)
	assert.Contains(t, doc.Content, "test content")
	assert.Equal(t, "Test Document", doc.Metadata["title"])
}
