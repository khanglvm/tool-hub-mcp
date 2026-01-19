package config

import (
	"strings"
	"unicode"
)

// ToCamelCase converts a string from various formats to camelCase.
//
// Supported input formats:
//   - dash-case: "jira-mcp" → "jiraMcp"
//   - snake_case: "jira_mcp" → "jiraMcp"
//   - PascalCase: "JiraMcp" → "jiraMcp"
//   - Already camelCase: "jiraMcp" → "jiraMcp"
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	// Split by common separators
	words := splitWords(s)
	if len(words) == 0 {
		return s
	}

	// First word lowercase, rest title case
	result := strings.Builder{}
	for i, word := range words {
		if word == "" {
			continue
		}
		if i == 0 {
			result.WriteString(strings.ToLower(word))
		} else {
			result.WriteString(strings.Title(strings.ToLower(word)))
		}
	}

	return result.String()
}

// splitWords splits a string into words based on separators and case changes.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		switch {
		case r == '-' || r == '_' || r == ' ':
			// Separator - flush current word
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		case unicode.IsUpper(r):
			// Check if this is a case transition (lowercase → uppercase)
			if i > 0 && current.Len() > 0 {
				prev := []rune(s)[i-1]
				if unicode.IsLower(prev) {
					words = append(words, current.String())
					current.Reset()
				}
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}

	// Flush remaining
	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// ToEnvVarCase converts a key to SCREAMING_SNAKE_CASE for environment variables.
//
// Examples:
//   - "jiraBaseUrl" → "JIRA_BASE_URL"
//   - "JIRA_BASE_URL" → "JIRA_BASE_URL"
//   - "jira-base-url" → "JIRA_BASE_URL"
func ToEnvVarCase(s string) string {
	words := splitWords(s)
	for i, word := range words {
		words[i] = strings.ToUpper(word)
	}
	return strings.Join(words, "_")
}

// NormalizeEnvVars converts all environment variable keys to SCREAMING_SNAKE_CASE.
// This handles configs that use camelCase or dash-case for env var names.
func NormalizeEnvVars(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}

	normalized := make(map[string]string, len(env))
	for key, value := range env {
		normalizedKey := ToEnvVarCase(key)
		normalized[normalizedKey] = value
	}
	return normalized
}
