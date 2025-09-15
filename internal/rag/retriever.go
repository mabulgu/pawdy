// Package rag provides retrieval-augmented generation functionality.
package rag

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/mabulgu/pawdy/pkg/types"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantRetriever implements document retrieval using Qdrant vector database.
type QdrantRetriever struct {
	collection   string
	embeddings   types.EmbeddingProvider
	client       *qdrant.Client
	pointsClient qdrant.PointsClient
}

// NewQdrantRetriever creates a new Qdrant-based retriever.
func NewQdrantRetriever(qdrantURL, collection string, embeddings types.EmbeddingProvider) (*QdrantRetriever, error) {
	// Parse the Qdrant URL to extract host and port
	parsedURL, err := url.Parse(qdrantURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Qdrant URL: %w", err)
	}

	host := parsedURL.Hostname()
	port := 6334 // Default Qdrant gRPC port
	if parsedURL.Port() != "" {
		// If HTTP port is specified, use gRPC port (HTTP port + 1)
		if httpPort, err := strconv.Atoi(parsedURL.Port()); err == nil {
			port = httpPort + 1
		}
	}

	// Create Qdrant client
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	retriever := &QdrantRetriever{
		collection:   collection,
		embeddings:   embeddings,
		client:       client,
		pointsClient: client.GetPointsClient(),
	}

	// Ensure collection exists
	if err := retriever.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure collection exists: %w", err)
	}

	return retriever, nil
}

// ensureCollection creates the collection if it doesn't exist.
func (r *QdrantRetriever) ensureCollection(ctx context.Context) error {
	// Check if collection exists first
	exists, err := r.client.CollectionExists(ctx, r.collection)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil // Collection already exists
	}

	// Create collection
	dimensions := r.embeddings.GetDimensions()
	err = r.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: r.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(dimensions),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// Search finds the most relevant documents for a query.
func (r *QdrantRetriever) Search(ctx context.Context, query string, topK int) ([]*types.Document, error) {
	// Generate embedding for query
	queryEmbeddings, err := r.embeddings.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	if len(queryEmbeddings) == 0 {
		return []*types.Document{}, nil
	}

	// Perform vector search in Qdrant using the low-level client
	searchResult, err := r.pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: r.collection,
		Vector:         queryEmbeddings[0],
		Limit:          uint64(topK),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search in Qdrant: %w", err)
	}

	// Convert Qdrant results to documents
	results := make([]*types.Document, 0, len(searchResult.GetResult()))
	for _, point := range searchResult.GetResult() {
		// Get ID as string (works for both UUID and numeric IDs)
		var docID string
		if uuid := point.GetId().GetUuid(); uuid != "" {
			docID = uuid
		} else {
			docID = fmt.Sprintf("%d", point.GetId().GetNum())
		}

		doc := &types.Document{
			ID:       docID,
			Score:    float64(point.GetScore()),
			Metadata: make(map[string]any),
		}

		// Extract content and metadata from payload
		if payload := point.GetPayload(); payload != nil {
			if content, exists := payload["content"]; exists {
				if contentStr, ok := content.GetKind().(*qdrant.Value_StringValue); ok {
					doc.Content = contentStr.StringValue
				}
			}

			// Copy all payload fields to metadata
			for key, value := range payload {
				if key != "content" {
					doc.Metadata[key] = convertQdrantValue(value)
				}
			}
		}

		results = append(results, doc)
	}

	return results, nil
}

// AddDocuments ingests and indexes new documents.
func (r *QdrantRetriever) AddDocuments(ctx context.Context, docs []*types.Document) error {
	if len(docs) == 0 {
		return nil
	}

	// Extract text content for embedding
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	// Generate embeddings
	embeddings, err := r.embeddings.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Prepare points for Qdrant
	points := make([]*qdrant.PointStruct, len(docs))
	for i, doc := range docs {
		// Create payload with content and metadata
		payload := map[string]interface{}{
			"content": doc.Content,
		}

		// Add metadata to payload, converting unsupported types
		for key, value := range doc.Metadata {
			// Convert time.Time to string format
			if t, ok := value.(time.Time); ok {
				payload[key] = t.Format(time.RFC3339)
			} else {
				payload[key] = value
			}
		}

		// Convert payload to Qdrant values
		qdrantPayload := qdrant.NewValueMap(payload)

		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(uint64(i + 1)), // Use numeric IDs instead of UUIDs
			Vectors: qdrant.NewVectors(embeddings[i]...),
			Payload: qdrantPayload,
		}
	}

	// Upsert points to Qdrant
	_, err = r.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.collection,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert points to Qdrant: %w", err)
	}

	return nil
}

// DeleteCollection removes all documents from the collection.
func (r *QdrantRetriever) DeleteCollection(ctx context.Context) error {
	err := r.client.DeleteCollection(ctx, r.collection)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	// Recreate the collection
	return r.ensureCollection(ctx)
}

// IsHealthy checks if the vector database is accessible.
func (r *QdrantRetriever) IsHealthy(ctx context.Context) error {
	exists, err := r.client.CollectionExists(ctx, r.collection)
	if err != nil {
		return fmt.Errorf("qdrant health check failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("collection %s does not exist", r.collection)
	}
	return nil
}

// convertQdrantValue converts a Qdrant value to a Go interface{}.
func convertQdrantValue(value *qdrant.Value) interface{} {
	switch v := value.GetKind().(type) {
	case *qdrant.Value_StringValue:
		return v.StringValue
	case *qdrant.Value_IntegerValue:
		return v.IntegerValue
	case *qdrant.Value_DoubleValue:
		return v.DoubleValue
	case *qdrant.Value_BoolValue:
		return v.BoolValue
	default:
		return nil
	}
}
