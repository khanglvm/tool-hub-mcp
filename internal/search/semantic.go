package search

import (
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

// EmbeddingModel provides semantic search via vector embeddings (optional).
type EmbeddingModel struct {
	storage storage.Storage
	cache   map[string][]float32
	mu      sync.RWMutex
	enabled bool
}

// NewEmbeddingModel creates a new embedding model wrapper.
// Embeddings are optional - if not available, search falls back to BM25.
func NewEmbeddingModel(store storage.Storage) *EmbeddingModel {
	return &EmbeddingModel{
		storage: store,
		cache:   make(map[string][]float32),
		enabled: true,
	}
}

// Embed generates an embedding for text (placeholder for future integration).
// Currently returns nil since we don't have an embedding model integrated yet.
func (e *EmbeddingModel) Embed(text string) ([]float32, error) {
	if !e.enabled {
		return nil, nil
	}

	// Check cache first
	e.mu.RLock()
	if vec, exists := e.cache[text]; exists {
		e.mu.RUnlock()
		return vec, nil
	}
	e.mu.RUnlock()

	// TODO: Integrate actual embedding model
	// Options for future integration:
	// - fastembed-go (lightweight, pure Go)
	// - OpenAI embeddings API (requires API key)
	// - Local model via Ollama
	// For now, we return nil to disable semantic search

	return nil, fmt.Errorf("embedding model not yet integrated")
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SearchSemantic performs semantic search (placeholder).
func (i *Indexer) SearchSemantic(query string, limit int) ([]SearchResult, error) {
	// Semantic search not yet implemented
	// This is a placeholder for future integration
	return nil, nil
}

// SaveEmbedding caches an embedding vector for a tool.
func (e *EmbeddingModel) SaveEmbedding(toolName string, vector []float32, version string) error {
	if !e.enabled || e.storage == nil {
		return nil
	}

	// Cache in memory
	e.mu.Lock()
	e.cache[toolName] = vector
	e.mu.Unlock()

	// Persist to storage
	if err := e.storage.SaveEmbedding(toolName, vector, version); err != nil {
		log.Printf("Warning: failed to save embedding to storage: %v", err)
	}

	return nil
}

// GetEmbedding retrieves a cached embedding for a tool.
func (e *EmbeddingModel) GetEmbedding(toolName string) ([]float32, error) {
	if !e.enabled {
		return nil, nil
	}

	// Check memory cache first
	e.mu.RLock()
	if vec, exists := e.cache[toolName]; exists {
		e.mu.RUnlock()
		return vec, nil
	}
	e.mu.RUnlock()

	// Check persistent storage
	if e.storage != nil {
		vector, _, err := e.storage.GetEmbedding(toolName)
		if err == nil && vector != nil {
			// Cache it
			e.mu.Lock()
			e.cache[toolName] = vector
			e.mu.Unlock()
			return vector, nil
		}
	}

	return nil, fmt.Errorf("embedding not found for tool: %s", toolName)
}

// ClearCache clears the in-memory embedding cache.
func (e *EmbeddingModel) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cache = make(map[string][]float32)
}
