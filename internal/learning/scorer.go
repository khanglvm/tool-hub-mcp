package learning

import (
	"math"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

const (
	// frequencyWeight is the weight for frequency in the score (0.6 = 60%).
	frequencyWeight = 0.6

	// recencyWeight is the weight for recency in the score (0.3 = 30%).
	recencyWeight = 0.3

	// ratingWeight is the weight for rating in the score (0.1 = 10%).
	ratingWeight = 0.1

	// frequencyWindow is the time window to consider for frequency (7 days).
	frequencyWindow = 7 * 24 * time.Hour

	// recencyHalfLife is the half-life for exponential decay (24 hours).
	recencyHalfLife = 24 * time.Hour

	// maxRating is the maximum possible rating (for normalization).
	maxRating = 5.0
)

// Score calculates a tool's score based on usage history.
// Formula: 0.6*frequency + 0.3*recency + 0.1*rating
func Score(toolName string, history []storage.UsageEvent) float64 {
	if len(history) == 0 {
		return 0.0
	}

	// Calculate components
	freq := calculateFrequency(toolName, history)
	recency := calculateRecency(history)
	rating := calculateRating(history)

	// Combined score
	score := frequencyWeight*freq + recencyWeight*recency + ratingWeight*rating

	return score
}

// calculateFrequency measures how often a tool is used (normalized 0-1).
// Counts usage in the last 7 days, normalized by a reasonable max (e.g., 100 uses).
func calculateFrequency(toolName string, history []storage.UsageEvent) float64 {
	if len(history) == 0 {
		return 0.0
	}

	count := 0
	now := time.Now()
	windowStart := now.Add(-frequencyWindow)

	for _, event := range history {
		if event.ToolName == toolName && event.Timestamp.After(windowStart) {
			count++
		}
	}

	// Normalize to 0-1 range (assuming 100 uses is "high frequency")
	normalized := float64(count) / 100.0
	return math.Min(normalized, 1.0)
}

// calculateRecency measures how recent the usage is (normalized 0-1).
// Uses exponential decay: recent usage weighted higher.
func calculateRecency(history []storage.UsageEvent) float64 {
	if len(history) == 0 {
		return 0.0
	}

	now := time.Now()
	totalWeight := 0.0
	weightedSum := 0.0

	for _, event := range history {
		// Calculate time difference in hours
		hoursSince := now.Sub(event.Timestamp).Hours()

		// Exponential decay: weight = e^(-ln(2) * t / half_life)
		// After 24 hours: weight = 0.5
		// After 48 hours: weight = 0.25
		decay := math.Exp(-math.Ln2 * hoursSince / recencyHalfLife.Hours())

		weightedSum += decay
		totalWeight += 1.0
	}

	if totalWeight == 0 {
		return 0.0
	}

	// Normalize to 0-1 range
	averageRecency := weightedSum / totalWeight
	return math.Min(averageRecency, 1.0)
}

// calculateRating averages user ratings (normalized 0-1).
// Only considers rated events (rating > 0).
func calculateRating(history []storage.UsageEvent) float64 {
	if len(history) == 0 {
		return 0.0
	}

	sum := 0
	count := 0

	for _, event := range history {
		if event.Rating > 0 {
			sum += event.Rating
			count++
		}
	}

	if count == 0 {
		// No ratings, return neutral score (0.5)
		return 0.5
	}

	average := float64(sum) / float64(count)

	// Normalize to 0-1 range
	return average / maxRating
}

// ToolScore represents a tool with its score for ranking.
type ToolScore struct {
	ToolName string
	Score    float64
}

// RankTools sorts tools by score (descending).
func RankTools(toolNames []string, storage storage.Storage) []ToolScore {
	scores := make([]ToolScore, 0, len(toolNames))

	for _, toolName := range toolNames {
		// Get usage history for the last 7 days
		history, err := storage.GetUsageHistory(toolName, time.Now().Add(-frequencyWindow))
		if err != nil {
			continue
		}

		score := Score(toolName, history)
		scores = append(scores, ToolScore{
			ToolName: toolName,
			Score:    score,
		})
	}

	// Sort by score descending
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	return scores
}
