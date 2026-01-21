package search

import (
	"math"
	"testing"
)

func TestNormalizeScores_Empty(t *testing.T) {
	results := []SearchResult{}
	normalized := normalizeScores(results)

	if len(normalized) != 0 {
		t.Errorf("expected empty result, got %d items", len(normalized))
	}
}

func TestNormalizeScores_Single(t *testing.T) {
	results := []SearchResult{
		{ToolName: "tool_a", Score: 0.5},
	}
	normalized := normalizeScores(results)

	if len(normalized) != 1 {
		t.Fatalf("expected 1 result, got %d", len(normalized))
	}

	// Single result should have score 1.0 (all scores are min=max)
	if normalized[0].Score != 1.0 {
		t.Errorf("expected score 1.0 for single result, got %f", normalized[0].Score)
	}
}

func TestNormalizeScores_Multiple(t *testing.T) {
	results := []SearchResult{
		{ToolName: "tool_a", Score: 0.0},
		{ToolName: "tool_b", Score: 0.5},
		{ToolName: "tool_c", Score: 1.0},
	}
	normalized := normalizeScores(results)

	if len(normalized) != 3 {
		t.Fatalf("expected 3 results, got %d", len(normalized))
	}

	// Check normalization
	// min=0.0, max=1.0, range=1.0
	// 0.0 -> (0.0-0.0)/1.0 = 0.0
	// 0.5 -> (0.5-0.0)/1.0 = 0.5
	// 1.0 -> (1.0-0.0)/1.0 = 1.0

	if math.Abs(normalized[0].Score-0.0) > 0.001 {
		t.Errorf("expected score 0.0, got %f", normalized[0].Score)
	}

	if math.Abs(normalized[1].Score-0.5) > 0.001 {
		t.Errorf("expected score 0.5, got %f", normalized[1].Score)
	}

	if math.Abs(normalized[2].Score-1.0) > 0.001 {
		t.Errorf("expected score 1.0, got %f", normalized[2].Score)
	}
}

func TestNormalizeScores_AlreadyNormalized(t *testing.T) {
	results := []SearchResult{
		{ToolName: "tool_a", Score: 0.0},
		{ToolName: "tool_b", Score: 0.5},
		{ToolName: "tool_c", Score: 1.0},
	}
	normalized := normalizeScores(results)

	// Should not change already normalized scores
	if normalized[0].Score != results[0].Score {
		t.Errorf("score changed: %f -> %f", results[0].Score, normalized[0].Score)
	}
}

func TestNormalizeScores_Negative(t *testing.T) {
	results := []SearchResult{
		{ToolName: "tool_a", Score: -1.0},
		{ToolName: "tool_b", Score: 0.0},
		{ToolName: "tool_c", Score: 1.0},
	}
	normalized := normalizeScores(results)

	// min=-1.0, max=1.0, range=2.0
	// -1.0 -> (-1.0-(-1.0))/2.0 = 0.0
	// 0.0 -> (0.0-(-1.0))/2.0 = 0.5
	// 1.0 -> (1.0-(-1.0))/2.0 = 1.0

	if math.Abs(normalized[0].Score-0.0) > 0.001 {
		t.Errorf("expected score 0.0, got %f", normalized[0].Score)
	}

	if math.Abs(normalized[1].Score-0.5) > 0.001 {
		t.Errorf("expected score 0.5, got %f", normalized[1].Score)
	}

	if math.Abs(normalized[2].Score-1.0) > 0.001 {
		t.Errorf("expected score 1.0, got %f", normalized[2].Score)
	}
}

func TestFuseScores_NoResults(t *testing.T) {
	bm25Results := []SearchResult{}
	semanticResults := []SearchResult{}
	config := DefaultFusionConfig

	fused := fuseScores(bm25Results, semanticResults, config)

	if len(fused) != 0 {
		t.Errorf("expected 0 fused results, got %d", len(fused))
	}
}

func TestFuseScores_OnlyBM25(t *testing.T) {
	bm25Results := []SearchResult{
		{ToolName: "tool_a", ServerName: "server1", Score: 0.8},
		{ToolName: "tool_b", ServerName: "server1", Score: 0.6},
	}
	semanticResults := []SearchResult{}
	config := DefaultFusionConfig

	fused := fuseScores(bm25Results, semanticResults, config)

	if len(fused) != 2 {
		t.Fatalf("expected 2 fused results, got %d", len(fused))
	}

	// Should use BM25 scores directly
	if fused[0].Score != bm25Results[0].Score {
		t.Errorf("expected BM25 score, got %f", fused[0].Score)
	}
}

func TestFuseScores_OnlySemantic(t *testing.T) {
	bm25Results := []SearchResult{}
	semanticResults := []SearchResult{
		{ToolName: "tool_a", ServerName: "server1", Score: 0.9},
		{ToolName: "tool_b", ServerName: "server1", Score: 0.7},
	}
	config := DefaultFusionConfig

	fused := fuseScores(bm25Results, semanticResults, config)

	if len(fused) != 2 {
		t.Fatalf("expected 2 fused results, got %d", len(fused))
	}

	// Should use semantic scores directly
	if fused[0].Score != semanticResults[0].Score {
		t.Errorf("expected semantic score, got %f", fused[0].Score)
	}
}

func TestFuseScores_BothAvailable(t *testing.T) {
	bm25Results := []SearchResult{
		{ToolName: "tool_a", ServerName: "server1", Score: 0.8},
	}
	semanticResults := []SearchResult{
		{ToolName: "tool_a", ServerName: "server1", Score: 0.9},
	}
	config := FusionConfig{
		SemanticWeight: 0.7,
		KeywordWeight:  0.3,
	}

	fused := fuseScores(bm25Results, semanticResults, config)

	if len(fused) != 1 {
		t.Fatalf("expected 1 fused result, got %d", len(fused))
	}

	// Expected: 0.7*0.9 + 0.3*0.8 = 0.63 + 0.24 = 0.87
	expectedScore := 0.7*0.9 + 0.3*0.8
	if math.Abs(fused[0].Score-expectedScore) > 0.001 {
		t.Errorf("expected fused score %f, got %f", expectedScore, fused[0].Score)
	}
}

func TestFuseScores_DifferentTools(t *testing.T) {
	bm25Results := []SearchResult{
		{ToolName: "tool_a", ServerName: "server1", Score: 0.8},
	}
	semanticResults := []SearchResult{
		{ToolName: "tool_b", ServerName: "server1", Score: 0.9},
	}
	config := DefaultFusionConfig

	fused := fuseScores(bm25Results, semanticResults, config)

	if len(fused) != 2 {
		t.Fatalf("expected 2 fused results (one from each), got %d", len(fused))
	}

	// Both should be present with their respective scores
	foundA, foundB := false, false
	for _, result := range fused {
		if result.ToolName == "tool_a" {
			foundA = true
			if result.Score != 0.8 {
				t.Errorf("expected tool_a score 0.8, got %f", result.Score)
			}
		}
		if result.ToolName == "tool_b" {
			foundB = true
			if result.Score != 0.9 {
				t.Errorf("expected tool_b score 0.9, got %f", result.Score)
			}
		}
	}

	if !foundA || !foundB {
		t.Error("not all tools were fused")
	}
}

func TestFuseScores_Overlapping(t *testing.T) {
	bm25Results := []SearchResult{
		{ToolName: "tool_a", ServerName: "server1", Score: 0.8},
		{ToolName: "tool_b", ServerName: "server1", Score: 0.6},
	}
	semanticResults := []SearchResult{
		{ToolName: "tool_a", ServerName: "server1", Score: 0.9},
		{ToolName: "tool_c", ServerName: "server1", Score: 0.7},
	}
	config := FusionConfig{
		SemanticWeight: 0.7,
		KeywordWeight:  0.3,
	}

	fused := fuseScores(bm25Results, semanticResults, config)

	if len(fused) != 3 {
		t.Fatalf("expected 3 fused results, got %d", len(fused))
	}

	// tool_a should have fused score
	// tool_b should have BM25 score only
	// tool_c should have semantic score only

	for _, result := range fused {
		switch result.ToolName {
		case "tool_a":
			expected := 0.7*0.9 + 0.3*0.8
			if math.Abs(result.Score-expected) > 0.001 {
				t.Errorf("tool_a: expected %f, got %f", expected, result.Score)
			}
		case "tool_b":
			if result.Score != 0.6 {
				t.Errorf("tool_b: expected 0.6, got %f", result.Score)
			}
		case "tool_c":
			if result.Score != 0.7 {
				t.Errorf("tool_c: expected 0.7, got %f", result.Score)
			}
		}
	}
}

func TestDefaultFusionConfig(t *testing.T) {
	if DefaultFusionConfig.SemanticWeight != 0.7 {
		t.Errorf("expected semantic weight 0.7, got %f", DefaultFusionConfig.SemanticWeight)
	}

	if DefaultFusionConfig.KeywordWeight != 0.3 {
		t.Errorf("expected keyword weight 0.3, got %f", DefaultFusionConfig.KeywordWeight)
	}

	// Weights should sum to 1.0
	sum := DefaultFusionConfig.SemanticWeight + DefaultFusionConfig.KeywordWeight
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("weights should sum to 1.0, got %f", sum)
	}
}

func TestFusionConfig_Custom(t *testing.T) {
	config := FusionConfig{
		SemanticWeight: 0.5,
		KeywordWeight:  0.5,
	}

	if config.SemanticWeight != 0.5 {
		t.Errorf("expected semantic weight 0.5, got %f", config.SemanticWeight)
	}

	if config.KeywordWeight != 0.5 {
		t.Errorf("expected keyword weight 0.5, got %f", config.KeywordWeight)
	}
}
