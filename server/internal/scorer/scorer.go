// Package scorer scores the quality of a Claude prompt and returns actionable feedback.
package scorer

import (
	"regexp"
	"strings"
)

// Quality represents a prompt quality tier.
type Quality int

const (
	QualityLow    Quality = 1
	QualityMedium Quality = 2
	QualityHigh   Quality = 3
)

// Score holds the result of scoring a single prompt.
type Score struct {
	Quality     Quality
	Label       string // "LOW" | "MEDIUM" | "HIGH"
	Suggestion  string // actionable tip; empty when quality is HIGH
	CharCount   int
	HasFilePath bool
	HasError    bool
}

// filePathRe matches a dot-separated word that looks like a filename (e.g. "file.go").
var filePathRe = regexp.MustCompile(`\b\w+\.\w+\b`)

// errorPhrases are the lower-cased substrings that indicate an error context.
var errorPhrases = []string{"error", "fail", "not work", "broken", "still"}

// ScorePrompt analyses text and returns a Score.
func ScorePrompt(text string) Score {
	charCount := len(text)
	lower := strings.ToLower(text)

	hasFilePath := strings.Contains(text, "/") || filePathRe.MatchString(text)

	hasError := false
	for _, phrase := range errorPhrases {
		if strings.Contains(lower, phrase) {
			hasError = true
			break
		}
	}

	// Determine quality tier.
	var quality Quality
	var label string
	switch {
	case charCount >= 120 && hasFilePath:
		quality = QualityHigh
		label = "HIGH"
	case charCount >= 80:
		quality = QualityMedium
		label = "MEDIUM"
	default:
		quality = QualityLow
		label = "LOW"
	}

	// Build suggestion.
	suggestion := buildSuggestion(quality, charCount, hasFilePath, hasError)

	return Score{
		Quality:     quality,
		Label:       label,
		Suggestion:  suggestion,
		CharCount:   charCount,
		HasFilePath: hasFilePath,
		HasError:    hasError,
	}
}

// buildSuggestion returns the most relevant actionable tip for the score.
func buildSuggestion(quality Quality, charCount int, hasFilePath, hasError bool) string {
	if quality == QualityHigh {
		return ""
	}
	if !hasFilePath && hasError {
		return "Include the file path where the error occurs"
	}
	if !hasFilePath {
		return "Include the file path to save ~1500 tokens of re-reading"
	}
	if charCount < 80 {
		return "Add more detail: what you tried, what failed, what you expect"
	}
	return ""
}
