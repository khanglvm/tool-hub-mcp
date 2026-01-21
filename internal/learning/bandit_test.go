package learning

import (
	"math/rand"
	"testing"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

// mockStorage is a mock implementation of storage.Storage for testing.
type mockStorage struct {
	history map[string][]storage.UsageEvent
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		history: make(map[string][]storage.UsageEvent),
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

func (m *mockStorage) SaveEmbedding(toolName string, vector []float32, version string) error {
	return nil
}

func (m *mockStorage) GetEmbedding(toolName string) ([]float32, string, error) {
	return nil, "", nil
}

func (m *mockStorage) ClearHistory(toolName string) error {
	delete(m.history, toolName)
	return nil
}

func (m *mockStorage) RecordSearch(search storage.SearchRecord) error {
	return nil
}

func (m *mockStorage) Cleanup(retention time.Duration) error {
	return nil
}

func TestNewEpsilonGreedy(t *testing.T) {
	bandit := NewEpsilonGreedy()

	if bandit == nil {
		t.Fatal("NewEpsilonGreedy returned nil")
	}

	if bandit.Epsilon != epsilon {
		t.Errorf("expected Epsilon=%f, got %f", epsilon, bandit.Epsilon)
	}

	if bandit.Seed == 0 {
		t.Error("expected Seed to be initialized, got 0")
	}
}

func TestSelectTool_SingleTool(t *testing.T) {
	bandit := NewEpsilonGreedy()
	mockStore := newMockStorage()
	tools := []string{"tool_a"}

	result := bandit.SelectTool(tools, mockStore)

	if result != "tool_a" {
		t.Errorf("expected 'tool_a', got '%s'", result)
	}
}

func TestSelectTool_EmptyList(t *testing.T) {
	bandit := NewEpsilonGreedy()
	mockStore := newMockStorage()
	tools := []string{}

	result := bandit.SelectTool(tools, mockStore)

	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

func TestSelectTool_Exploitation(t *testing.T) {
	// Set seed to force exploitation (epsilon = 0.1, so we need rand.Float64() >= 0.1)
	rand.Seed(42) // This seed produces rand.Float64() >= 0.1

	bandit := NewEpsilonGreedy()
	bandit.Seed = 42
	bandit.Epsilon = 0.0 // Force exploitation

	mockStore := newMockStorage()
	now := time.Now()

	// Add history: tool_a used more recently than tool_b
	mockStore.RecordUsage(storage.UsageEvent{
		ToolName:  "tool_a",
		Timestamp: now.Add(-1 * time.Hour),
	})
	mockStore.RecordUsage(storage.UsageEvent{
		ToolName:  "tool_b",
		Timestamp: now.Add(-24 * time.Hour),
	})

	tools := []string{"tool_a", "tool_b"}

	result := bandit.SelectTool(tools, mockStore)

	// Should select tool_a (higher recency score)
	if result != "tool_a" {
		t.Errorf("expected 'tool_a' (higher score), got '%s'", result)
	}
}

func TestSelectTool_Exploration(t *testing.T) {
	// Run many times to verify exploration happens
	bandit := NewEpsilonGreedy()
	bandit.Epsilon = 1.0 // Force 100% exploration

	mockStore := newMockStorage()
	tools := []string{"tool_a", "tool_b", "tool_c"}

	selections := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		result := bandit.SelectTool(tools, mockStore)
		selections[result]++
	}

	// With 100% exploration, all tools should be selected at least once
	for _, tool := range tools {
		if selections[tool] == 0 {
			t.Errorf("tool '%s' was never selected (exploration failed)", tool)
		}
	}
}

func TestSelectRankedTools_SingleTool(t *testing.T) {
	bandit := NewEpsilonGreedy()
	mockStore := newMockStorage()
	tools := []string{"tool_a"}

	result := bandit.SelectRankedTools(tools, mockStore)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	if result[0] != "tool_a" {
		t.Errorf("expected 'tool_a', got '%s'", result[0])
	}
}

func TestSelectRankedTools_Exploitation(t *testing.T) {
	bandit := NewEpsilonGreedy()
	bandit.Epsilon = 0.0 // Force exploitation

	mockStore := newMockStorage()
	now := time.Now()

	// Create history with clear ranking
	mockStore.RecordUsage(storage.UsageEvent{
		ToolName:  "tool_a",
		Timestamp: now.Add(-1 * time.Hour),
		Rating:    5,
	})
	mockStore.RecordUsage(storage.UsageEvent{
		ToolName:  "tool_a",
		Timestamp: now.Add(-2 * time.Hour),
		Rating:    5,
	})
	mockStore.RecordUsage(storage.UsageEvent{
		ToolName:  "tool_b",
		Timestamp: now.Add(-24 * time.Hour),
		Rating:    3,
	})

	tools := []string{"tool_a", "tool_b"}

	result := bandit.SelectRankedTools(tools, mockStore)

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	// tool_a should be first (higher score)
	if result[0] != "tool_a" {
		t.Errorf("expected 'tool_a' first, got '%s'", result[0])
	}

	if result[1] != "tool_b" {
		t.Errorf("expected 'tool_b' second, got '%s'", result[1])
	}
}

func TestSelectRankedTools_Exploration(t *testing.T) {
	bandit := NewEpsilonGreedy()
	bandit.Epsilon = 1.0 // Force 100% exploration

	mockStore := newMockStorage()
	tools := []string{"tool_a", "tool_b", "tool_c"}

	result := bandit.SelectRankedTools(tools, mockStore)

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	// All tools should be present
	toolSet := make(map[string]bool)
	for _, tool := range result {
		toolSet[tool] = true
	}

	for _, tool := range tools {
		if !toolSet[tool] {
			t.Errorf("tool '%s' missing from result", tool)
		}
	}
}

func TestIsExploration(t *testing.T) {
	bandit := NewEpsilonGreedy()
	bandit.Epsilon = 1.0 // Force exploration

	explorations := 0
	iterations := 100

	for i := 0; i < iterations; i++ {
		if bandit.IsExploration() {
			explorations++
		}
	}

	// With epsilon=1.0, should always explore
	if explorations != iterations {
		t.Errorf("expected %d explorations, got %d", iterations, explorations)
	}
}

func TestSetEpsilon(t *testing.T) {
	bandit := NewEpsilonGreedy()

	// Valid epsilon
	bandit.SetEpsilon(0.5)
	if bandit.Epsilon != 0.5 {
		t.Errorf("expected Epsilon=0.5, got %f", bandit.Epsilon)
	}

	// Invalid epsilon (should be ignored)
	bandit.SetEpsilon(-0.1)
	if bandit.Epsilon != 0.5 {
		t.Errorf("expected Epsilon to remain 0.5, got %f", bandit.Epsilon)
	}

	bandit.SetEpsilon(1.5)
	if bandit.Epsilon != 0.5 {
		t.Errorf("expected Epsilon to remain 0.5, got %f", bandit.Epsilon)
	}
}

func TestGetEpsilon(t *testing.T) {
	bandit := NewEpsilonGreedy()
	bandit.Epsilon = 0.25

	result := bandit.GetEpsilon()
	if result != 0.25 {
		t.Errorf("expected 0.25, got %f", result)
	}
}

func TestSelectTool_NoHistory(t *testing.T) {
	bandit := NewEpsilonGreedy()
	bandit.Epsilon = 0.0 // Force exploitation

	mockStore := newMockStorage()
	tools := []string{"tool_a", "tool_b"}

	// No history, should fallback to random
	result := bandit.SelectTool(tools, mockStore)

	if result == "" {
		t.Error("expected a tool to be selected, got empty string")
	}

	if result != "tool_a" && result != "tool_b" {
		t.Errorf("expected 'tool_a' or 'tool_b', got '%s'", result)
	}
}
