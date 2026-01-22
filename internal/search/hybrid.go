package search

import (
	"sort"
)

// FusionConfig defines weights for hybrid score fusion.
type FusionConfig struct {
	SemanticWeight float64
	KeywordWeight  float64
}

// DefaultFusionConfig provides balanced fusion (70% semantic, 30% keyword).
var DefaultFusionConfig = FusionConfig{
	SemanticWeight: 0.7,
	KeywordWeight:  0.3,
}

// SearchHybrid performs hybrid search combining BM25 and semantic scores.
func (i *Indexer) SearchHybrid(query string, limit int, config FusionConfig) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get BM25 results (always available)
	bm25Results, err := i.SearchBM25(query, limit*2)
	if err != nil {
		return nil, err
	}

	// Get semantic results (may be nil if not available)
	semanticResults, err := i.SearchSemantic(query, limit*2)
	if err != nil {
		// Fall back to BM25 only
		return bm25Results, nil
	}

	// If semantic search is not available, return BM25 results
	if semanticResults == nil {
		return bm25Results, nil
	}

	// Fuse scores
	fusedResults := fuseScores(bm25Results, semanticResults, config)

	// Sort by combined score (descending)
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	// Return top N results
	if len(fusedResults) > limit {
		fusedResults = fusedResults[:limit]
	}

	return fusedResults, nil
}

// fuseScores combines BM25 and semantic results using weighted fusion.
func fuseScores(bm25Results, semanticResults []SearchResult, config FusionConfig) []SearchResult {
	// Create map for semantic results by tool ID
	semanticMap := make(map[string]SearchResult)
	for _, result := range semanticResults {
		toolID := result.ServerName + "/" + result.ToolName
		semanticMap[toolID] = result
	}

	// Create map for BM25 results
	bm25Map := make(map[string]SearchResult)
	for _, result := range bm25Results {
		toolID := result.ServerName + "/" + result.ToolName
		bm25Map[toolID] = result
	}

	// Collect all unique tool IDs
	allToolIDs := make(map[string]bool)
	for _, result := range bm25Results {
		toolID := result.ServerName + "/" + result.ToolName
		allToolIDs[toolID] = true
	}
	for _, result := range semanticResults {
		toolID := result.ServerName + "/" + result.ToolName
		allToolIDs[toolID] = true
	}

	// Fuse scores for each unique tool
	fusedResults := make([]SearchResult, 0, len(allToolIDs))

	for toolID := range allToolIDs {
		var bm25Result, semanticResult SearchResult
		var hasBM25, hasSemantic bool

		if bm25Res, exists := bm25Map[toolID]; exists {
			bm25Result = bm25Res
			hasBM25 = true
		}

		if semRes, exists := semanticMap[toolID]; exists {
			semanticResult = semRes
			hasSemantic = true
		}

		// Calculate fused score
		var fusedScore float64
		var baseResult SearchResult

		if hasBM25 && hasSemantic {
			// Both available: weighted combination
			fusedScore = config.SemanticWeight*semanticResult.Score +
				config.KeywordWeight*bm25Result.Score
			baseResult = semanticResult // Use semantic result as base
		} else if hasBM25 {
			// Only BM25 available
			fusedScore = bm25Result.Score
			baseResult = bm25Result
		} else if hasSemantic {
			// Only semantic available
			fusedScore = semanticResult.Score
			baseResult = semanticResult
		} else {
			continue
		}

		// Create fused result
		fusedResult := SearchResult{
			ToolName:    baseResult.ToolName,
			Description: baseResult.Description,
			InputSchema: baseResult.InputSchema,
			ServerName:  baseResult.ServerName,
			Score:       fusedScore,
		}

		fusedResults = append(fusedResults, fusedResult)
	}

	return fusedResults
}

// normalizeScores normalizes scores to [0, 1] range.
func normalizeScores(results []SearchResult) []SearchResult {
	if len(results) == 0 {
		return results
	}

	// Find min and max scores
	minScore := results[0].Score
	maxScore := results[0].Score

	for _, result := range results {
		if result.Score < minScore {
			minScore = result.Score
		}
		if result.Score > maxScore {
			maxScore = result.Score
		}
	}

	// Avoid division by zero - when all scores are equal, set all to 1.0
	if maxScore == minScore {
		normalized := make([]SearchResult, len(results))
		for i, result := range results {
			normalized[i] = result
			normalized[i].Score = 1.0
		}
		return normalized
	}

	// Normalize
	normalized := make([]SearchResult, len(results))
	for i, result := range results {
		normalized[i] = result
		normalized[i].Score = (result.Score - minScore) / (maxScore - minScore)
	}

	return normalized
}
