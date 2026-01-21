package search

import (
	"fmt"
	"testing"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

func TestCosineSimilarity_Identical(t *testing.T) {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{1.0, 2.0, 3.0}

	similarity := cosineSimilarity(a, b)

	// Identical vectors should have similarity 1.0
	if similarity != 1.0 {
		t.Errorf("expected similarity 1.0 for identical vectors, got %f", similarity)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1.0, 0.0}
	b := []float32{0.0, 1.0}

	similarity := cosineSimilarity(a, b)

	// Orthogonal vectors should have similarity 0.0
	if similarity != 0.0 {
		t.Errorf("expected similarity 0.0 for orthogonal vectors, got %f", similarity)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{-1.0, -2.0, -3.0}

	similarity := cosineSimilarity(a, b)

	// Opposite vectors should have similarity -1.0
	if similarity != -1.0 {
		t.Errorf("expected similarity -1.0 for opposite vectors, got %f", similarity)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{1.0, 2.0}

	similarity := cosineSimilarity(a, b)

	// Different lengths should return 0.0
	if similarity != 0.0 {
		t.Errorf("expected similarity 0.0 for different lengths, got %f", similarity)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0.0, 0.0, 0.0}
	b := []float32{1.0, 2.0, 3.0}

	similarity := cosineSimilarity(a, b)

	// Zero vector should return 0.0 (avoid division by zero)
	if similarity != 0.0 {
		t.Errorf("expected similarity 0.0 for zero vector, got %f", similarity)
	}
}

func TestCosineSimilarity_RealExample(t *testing.T) {
	// Example: document embeddings
	doc1 := []float32{0.5, 0.2, 0.8}
	doc2 := []float32{0.6, 0.3, 0.7}

	similarity := cosineSimilarity(doc1, doc2)

	// Should be positive and close to 1.0 (similar documents)
	if similarity <= 0.0 || similarity > 1.0 {
		t.Errorf("expected similarity in (0, 1], got %f", similarity)
	}
}

func TestNewEmbeddingModel(t *testing.T) {
	mockStore := newMockStorage()
	model := NewEmbeddingModel(mockStore)

	if model == nil {
		t.Fatal("NewEmbeddingModel returned nil")
	}

	if model.storage == nil {
		t.Error("expected storage to be initialized")
	}

	if model.cache == nil {
		t.Error("expected cache to be initialized")
	}

	if !model.enabled {
		t.Error("expected model to be enabled")
	}
}

func TestEmbeddingModel_Embed_NotImplemented(t *testing.T) {
	mockStore := newMockStorage()
	model := NewEmbeddingModel(mockStore)

	_, err := model.Embed("test text")

	if err == nil {
		t.Error("expected error for unimplemented Embed, got nil")
	}
}

func TestEmbeddingModel_SaveEmbedding(t *testing.T) {
	mockStore := newMockStorage()
	model := NewEmbeddingModel(mockStore)

	vector := []float32{0.1, 0.2, 0.3}
	err := model.SaveEmbedding("tool_a", vector, "v1")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Check cache
	cached, exists := model.cache["tool_a"]
	if !exists {
		t.Error("embedding not cached")
	}

	if len(cached) != len(vector) {
		t.Errorf("cached vector length mismatch: expected %d, got %d", len(vector), len(cached))
	}
}

func TestEmbeddingModel_GetEmbedding_CacheHit(t *testing.T) {
	mockStore := newMockStorage()
	model := NewEmbeddingModel(mockStore)

	vector := []float32{0.1, 0.2, 0.3}
	model.cache["tool_a"] = vector

	result, err := model.GetEmbedding("tool_a")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(result) != len(vector) {
		t.Errorf("vector length mismatch: expected %d, got %d", len(vector), len(result))
	}

	for i := range vector {
		if result[i] != vector[i] {
			t.Errorf("vector value mismatch at index %d: expected %f, got %f", i, vector[i], result[i])
		}
	}
}

func TestEmbeddingModel_GetEmbedding_NotFound(t *testing.T) {
	mockStore := newMockStorage()
	model := NewEmbeddingModel(mockStore)

	_, err := model.GetEmbedding("nonexistent")

	if err == nil {
		t.Error("expected error for nonexistent embedding, got nil")
	}
}

func TestEmbeddingModel_ClearCache(t *testing.T) {
	mockStore := newMockStorage()
	model := NewEmbeddingModel(mockStore)

	// Add some embeddings
	model.cache["tool_a"] = []float32{0.1, 0.2, 0.3}
	model.cache["tool_b"] = []float32{0.4, 0.5, 0.6}

	model.ClearCache()

	if len(model.cache) != 0 {
		t.Errorf("expected empty cache after clear, got %d items", len(model.cache))
	}
}

func TestEmbeddingModel_Disabled(t *testing.T) {
	mockStore := newMockStorage()
	model := NewEmbeddingModel(mockStore)
	model.enabled = false

	// Embed should return nil
	vector, err := model.Embed("test")
	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}
	if vector != nil {
		t.Error("expected nil vector when disabled")
	}

	// GetEmbedding should return nil
	vector, err = model.GetEmbedding("tool_a")
	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}
	if vector != nil {
		t.Error("expected nil vector when disabled")
	}

	// SaveEmbedding should be no-op
	err = model.SaveEmbedding("tool_a", []float32{0.1}, "v1")
	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}
}

// mockStorage for search tests (reusing from learning tests)
type mockStorage struct {
	embeddings map[string][]float32
	history    map[string][]storage.UsageEvent
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		embeddings: make(map[string][]float32),
		history:    make(map[string][]storage.UsageEvent),
	}
}

func (m *mockStorage) Init() error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func (m *mockStorage) RecordUsage(event storage.UsageEvent) error {
	if m.history == nil {
		m.history = make(map[string][]storage.UsageEvent)
	}
	m.history[event.ToolName] = append(m.history[event.ToolName], event)
	return nil
}

func (m *mockStorage) GetUsageHistory(toolName string, since time.Time) ([]storage.UsageEvent, error) {
	if hist, ok := m.history[toolName]; ok {
		return hist, nil
	}
	return []storage.UsageEvent{}, nil
}

func (m *mockStorage) RecordSearch(search storage.SearchRecord) error {
	return nil
}

func (m *mockStorage) Cleanup(retention time.Duration) error {
	return nil
}

func (m *mockStorage) SaveEmbedding(toolName string, vector []float32, version string) error {
	if m.embeddings == nil {
		m.embeddings = make(map[string][]float32)
	}
	m.embeddings[toolName] = vector
	return nil
}

func (m *mockStorage) GetEmbedding(toolName string) ([]float32, string, error) {
	if vec, ok := m.embeddings[toolName]; ok {
		return vec, "v1", nil
	}
	return nil, "", fmt.Errorf("embedding not found")
}

