package learning

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

func TestScore_EmptyHistory(t *testing.T) {
	history := []storage.UsageEvent{}
	score := Score("tool_a", history)

	if score != 0.0 {
		t.Errorf("expected score 0.0 for empty history, got %f", score)
	}
}

func TestCalculateFrequency_NoHistory(t *testing.T) {
	history := []storage.UsageEvent{}
	freq := calculateFrequency("tool_a", history)

	if freq != 0.0 {
		t.Errorf("expected frequency 0.0 for empty history, got %f", freq)
	}
}

func TestCalculateFrequency_WithinWindow(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now.Add(-1 * time.Hour)},
		{ToolName: "tool_a", Timestamp: now.Add(-2 * time.Hour)},
		{ToolName: "tool_a", Timestamp: now.Add(-24 * time.Hour)},
	}

	freq := calculateFrequency("tool_a", history)

	// 3 uses in 7-day window
	expected := 3.0 / 100.0 // normalized
	if math.Abs(freq-expected) > 0.001 {
		t.Errorf("expected frequency ~%f, got %f", expected, freq)
	}
}

func TestCalculateFrequency_OutsideWindow(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now.Add(-8 * 24 * time.Hour)}, // 8 days ago
	}

	freq := calculateFrequency("tool_a", history)

	if freq != 0.0 {
		t.Errorf("expected frequency 0.0 for events outside window, got %f", freq)
	}
}

func TestCalculateFrequency_DifferentTool(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_b", Timestamp: now.Add(-1 * time.Hour)},
	}

	freq := calculateFrequency("tool_a", history)

	if freq != 0.0 {
		t.Errorf("expected frequency 0.0 for different tool, got %f", freq)
	}
}

func TestCalculateFrequency_HighUsage(t *testing.T) {
	now := time.Now()
	history := make([]storage.UsageEvent, 150)
	for i := 0; i < 150; i++ {
		history[i] = storage.UsageEvent{
			ToolName:  "tool_a",
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
		}
	}

	freq := calculateFrequency("tool_a", history)

	// Should be capped at 1.0
	if freq != 1.0 {
		t.Errorf("expected frequency 1.0 (capped) for high usage, got %f", freq)
	}
}

func TestCalculateRecency_NoHistory(t *testing.T) {
	history := []storage.UsageEvent{}
	recency := calculateRecency(history)

	if recency != 0.0 {
		t.Errorf("expected recency 0.0 for empty history, got %f", recency)
	}
}

func TestCalculateRecency_RecentUsage(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now.Add(-1 * time.Hour)},
	}

	recency := calculateRecency(history)

	// Recent usage should have high recency (>0.5)
	if recency < 0.5 {
		t.Errorf("expected recency >0.5 for recent usage, got %f", recency)
	}
}

func TestCalculateRecency_OldUsage(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now.Add(-72 * time.Hour)}, // 3 days ago
	}

	recency := calculateRecency(history)

	// Old usage should have low recency (<0.5)
	if recency > 0.5 {
		t.Errorf("expected recency <0.5 for old usage, got %f", recency)
	}
}

func TestCalculateRecency_ExponentialDecay(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now.Add(-24 * time.Hour)}, // 1 day ago
	}

	recency := calculateRecency(history)

	// After 24 hours (one half-life), decay should be ~0.5
	if math.Abs(recency-0.5) > 0.1 {
		t.Errorf("expected recency ~0.5 after one half-life, got %f", recency)
	}
}

func TestCalculateRating_NoHistory(t *testing.T) {
	history := []storage.UsageEvent{}
	rating := calculateRating(history)

	if rating != 0.0 {
		t.Errorf("expected rating 0.0 for empty history, got %f", rating)
	}
}

func TestCalculateRating_NoRatings(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now, Rating: 0},
		{ToolName: "tool_a", Timestamp: now, Rating: 0},
	}

	rating := calculateRating(history)

	// No ratings should return neutral (0.5)
	if rating != 0.5 {
		t.Errorf("expected neutral rating 0.5 when no ratings, got %f", rating)
	}
}

func TestCalculateRating_Perfect(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now, Rating: 5},
		{ToolName: "tool_a", Timestamp: now, Rating: 5},
	}

	rating := calculateRating(history)

	// Average 5/5 = 1.0
	if rating != 1.0 {
		t.Errorf("expected rating 1.0 for perfect scores, got %f", rating)
	}
}

func TestCalculateRating_Mixed(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now, Rating: 5},
		{ToolName: "tool_a", Timestamp: now, Rating: 3},
	}

	rating := calculateRating(history)

	// Average (5+3)/2 = 4, normalized to 4/5 = 0.8
	expected := 4.0 / 5.0
	if math.Abs(rating-expected) > 0.001 {
		t.Errorf("expected rating %f, got %f", expected, rating)
	}
}

func TestCalculateRating_IgnoresUnrated(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now, Rating: 5},
		{ToolName: "tool_a", Timestamp: now, Rating: 0}, // Unrated
		{ToolName: "tool_a", Timestamp: now, Rating: 0}, // Unrated
	}

	rating := calculateRating(history)

	// Should only count the rated event (5/5 = 1.0)
	if rating != 1.0 {
		t.Errorf("expected rating 1.0 (ignoring unrated), got %f", rating)
	}
}

func TestScore_Components(t *testing.T) {
	now := time.Now()
	history := []storage.UsageEvent{
		{ToolName: "tool_a", Timestamp: now.Add(-1 * time.Hour), Rating: 5},
	}

	score := Score("tool_a", history)

	// Score = 0.6*freq + 0.3*recency + 0.1*rating
	// freq: 1 use / 100 = 0.01
	// recency: recent usage (>0.5)
	// rating: 5/5 = 1.0
	// Expected: 0.6*0.01 + 0.3*0.7 + 0.1*1.0 ≈ 0.316

	if score <= 0.0 {
		t.Errorf("expected positive score, got %f", score)
	}

	if score > 1.0 {
		t.Errorf("expected score <=1.0, got %f", score)
	}
}

func TestRankTools_EmptyList(t *testing.T) {
	mockStore := newMockStorage()
	tools := []string{}

	result := RankTools(tools, mockStore)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

func TestRankTools_SingleTool(t *testing.T) {
	mockStore := newMockStorage()
	now := time.Now()
	mockStore.RecordUsage(storage.UsageEvent{
		ToolName:  "tool_a",
		Timestamp: now,
	})

	tools := []string{"tool_a"}

	result := RankTools(tools, mockStore)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	if result[0].ToolName != "tool_a" {
		t.Errorf("expected 'tool_a', got '%s'", result[0].ToolName)
	}

	if result[0].Score <= 0.0 {
		t.Errorf("expected positive score, got %f", result[0].Score)
	}
}

func TestRankTools_Sorting(t *testing.T) {
	mockStore := newMockStorage()
	now := time.Now()

	// tool_a: more recent, higher rating
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

	// tool_b: older, lower rating
	mockStore.RecordUsage(storage.UsageEvent{
		ToolName:  "tool_b",
		Timestamp: now.Add(-24 * time.Hour),
		Rating:    3,
	})

	tools := []string{"tool_b", "tool_a"} // Intentionally reversed

	result := RankTools(tools, mockStore)

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	// tool_a should be first (higher score)
	if result[0].ToolName != "tool_a" {
		t.Errorf("expected 'tool_a' first, got '%s'", result[0].ToolName)
	}

	// Scores should be descending
	if result[0].Score <= result[1].Score {
		t.Errorf("expected descending scores: %f <= %f", result[0].Score, result[1].Score)
	}
}

func TestRankTools_StorageError(t *testing.T) {
	// Use a mock that returns errors
	errorStore := &errorMockStorage{}
	tools := []string{"tool_a", "tool_b"}

	result := RankTools(tools, errorStore)

	// Should return empty list on error
	if len(result) != 0 {
		t.Errorf("expected empty result on error, got %d items", len(result))
	}
}

// errorMockStorage always returns errors
type errorMockStorage struct{}

func (e *errorMockStorage) Init() error {
	return nil
}

func (e *errorMockStorage) Close() error {
	return nil
}

func (e *errorMockStorage) RecordUsage(event storage.UsageEvent) error {
	return nil
}

func (e *errorMockStorage) GetUsageHistory(toolName string, since time.Time) ([]storage.UsageEvent, error) {
	return nil, fmt.Errorf("storage not initialized")
}

func (e *errorMockStorage) SaveEmbedding(toolName string, vector []float32, version string) error {
	return nil
}

func (e *errorMockStorage) GetEmbedding(toolName string) ([]float32, string, error) {
	return nil, "", nil
}

func (e *errorMockStorage) ClearHistory(toolName string) error {
	return nil
}

func (e *errorMockStorage) RecordSearch(search storage.SearchRecord) error {
	return nil
}

func (e *errorMockStorage) Cleanup(retention time.Duration) error {
	return nil
}
