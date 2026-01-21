package learning

import (
	"math/rand"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

const (
	// epsilon is the exploration rate (0.1 = 10% explore, 90% exploit).
	epsilon = 0.1
)

// EpsilonGreedy implements ε-greedy multi-armed bandit for tool selection.
type EpsilonGreedy struct {
	// Epsilon is the exploration rate (default: 0.1).
	Epsilon float64

	// Seed for reproducible randomness (optional).
	Seed int64
}

// NewEpsilonGreedy creates a new ε-greedy bandit with default parameters.
func NewEpsilonGreedy() *EpsilonGreedy {
	return &EpsilonGreedy{
		Epsilon: epsilon,
		Seed:    time.Now().UnixNano(),
	}
}

// SelectTool selects a tool using ε-greedy strategy.
// With probability ε, explore (random selection).
// With probability 1-ε, exploit (highest score).
func (e *EpsilonGreedy) SelectTool(toolNames []string, storage storage.Storage) string {
	if len(toolNames) == 0 {
		return ""
	}

	if len(toolNames) == 1 {
		return toolNames[0]
	}

	// Initialize random seed if provided
	if e.Seed != 0 {
		rand.Seed(e.Seed)
	}

	// Explore: random selection
	if rand.Float64() < e.Epsilon {
		idx := rand.Intn(len(toolNames))
		return toolNames[idx]
	}

	// Exploit: select tool with highest score
	scores := RankTools(toolNames, storage)
	if len(scores) == 0 {
		// Fallback to random if no scores available
		idx := rand.Intn(len(toolNames))
		return toolNames[idx]
	}

	return scores[0].ToolName
}

// SelectRankedTools selects tools using ε-greedy, returning ranked list.
// First element is the selected tool, rest are ranked by score.
func (e *EpsilonGreedy) SelectRankedTools(toolNames []string, storage storage.Storage) []string {
	if len(toolNames) == 0 {
		return []string{}
	}

	if len(toolNames) == 1 {
		return toolNames
	}

	// Initialize random seed if provided
	if e.Seed != 0 {
		rand.Seed(e.Seed)
	}

	// Rank all tools by score
	scores := RankTools(toolNames, storage)

	// Explore: return shuffled list
	if rand.Float64() < e.Epsilon {
		shuffled := make([]string, len(toolNames))
		copy(shuffled, toolNames)

		// Fisher-Yates shuffle
		for i := len(shuffled) - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		}

		return shuffled
	}

	// Exploit: return ranked list
	result := make([]string, len(scores))
	for i, score := range scores {
		result[i] = score.ToolName
	}

	return result
}

// IsExploration returns whether the last selection was exploration (true) or exploitation (false).
func (e *EpsilonGreedy) IsExploration() bool {
	return rand.Float64() < e.Epsilon
}

// SetEpsilon updates the exploration rate.
func (e *EpsilonGreedy) SetEpsilon(eps float64) {
	if eps < 0 || eps > 1 {
		return
	}
	e.Epsilon = eps
}

// GetEpsilon returns the current exploration rate.
func (e *EpsilonGreedy) GetEpsilon() float64 {
	return e.Epsilon
}
