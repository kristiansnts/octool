package internal_test

import (
	"strings"
	"testing"

	"github.com/kristiansnts/octool/internal/scorer"
)

func TestScorePromptLow(t *testing.T) {
	// Very short prompt with no file path and no error markers.
	s := scorer.ScorePrompt("fix it")

	if s.Quality != scorer.QualityLow {
		t.Errorf("Quality: got %d, want QualityLow (%d)", s.Quality, scorer.QualityLow)
	}
	if s.Label != "LOW" {
		t.Errorf("Label: got %q, want %q", s.Label, "LOW")
	}
	if s.CharCount != len("fix it") {
		t.Errorf("CharCount: got %d, want %d", s.CharCount, len("fix it"))
	}
}

func TestScorePromptMedium(t *testing.T) {
	// 80+ characters but no file path → MEDIUM (not HIGH).
	prompt := strings.Repeat("a", 85)
	s := scorer.ScorePrompt(prompt)

	if s.Quality != scorer.QualityMedium {
		t.Errorf("Quality: got %d, want QualityMedium (%d)", s.Quality, scorer.QualityMedium)
	}
	if s.Label != "MEDIUM" {
		t.Errorf("Label: got %q, want %q", s.Label, "MEDIUM")
	}
}

func TestScorePromptHigh(t *testing.T) {
	// 120+ chars AND contains a file path.
	base := strings.Repeat("x", 110)
	prompt := base + " /src/main.go"
	s := scorer.ScorePrompt(prompt)

	if s.Quality != scorer.QualityHigh {
		t.Errorf("Quality: got %d, want QualityHigh (%d)", s.Quality, scorer.QualityHigh)
	}
	if s.Label != "HIGH" {
		t.Errorf("Label: got %q, want %q", s.Label, "HIGH")
	}
	if s.Suggestion != "" {
		t.Errorf("HIGH prompt should have empty suggestion, got %q", s.Suggestion)
	}
}

func TestHasFilePath(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"/some/path/file.go", true},
		{"look at main.go for details", true},
		{"nothing here", false},
		{"check the config.yaml please", true},
	}
	for _, tt := range tests {
		s := scorer.ScorePrompt(tt.text)
		if s.HasFilePath != tt.want {
			t.Errorf("HasFilePath(%q): got %v, want %v", tt.text, s.HasFilePath, tt.want)
		}
	}
}

func TestHasError(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"there is an error in the output", true},
		{"it keeps failing on startup", true},
		{"the feature does not work", true},
		{"it is broken", true},
		{"still not working after the fix", true},
		{"everything is great", false},
	}
	for _, tt := range tests {
		s := scorer.ScorePrompt(tt.text)
		if s.HasError != tt.want {
			t.Errorf("HasError(%q): got %v, want %v", tt.text, s.HasError, tt.want)
		}
	}
}

func TestSuggestionNoFilePath(t *testing.T) {
	// Short prompt without file path and without error.
	s := scorer.ScorePrompt("nothing much")
	if s.Suggestion == "" {
		t.Error("expected a suggestion for a prompt with no file path")
	}
	if !strings.Contains(s.Suggestion, "file path") {
		t.Errorf("suggestion should mention file path, got: %q", s.Suggestion)
	}
}

func TestSuggestionErrorWithoutFilePath(t *testing.T) {
	// Contains error signal but no file path.
	s := scorer.ScorePrompt("error here")
	if !strings.Contains(s.Suggestion, "file path") {
		t.Errorf("suggestion should mention file path for error-without-path prompt, got: %q", s.Suggestion)
	}
}

func TestSuggestionShortPromptWithFilePath(t *testing.T) {
	// Short prompt that has a file path token but is < 80 chars.
	s := scorer.ScorePrompt("fix /a/b.go")
	// charCount < 80 and hasFilePath=true → suggestion about more detail.
	if !strings.Contains(s.Suggestion, "detail") {
		t.Errorf("suggestion should ask for more detail, got: %q", s.Suggestion)
	}
}

func TestCharCount(t *testing.T) {
	text := "hello world"
	s := scorer.ScorePrompt(text)
	if s.CharCount != len(text) {
		t.Errorf("CharCount: got %d, want %d", s.CharCount, len(text))
	}
}

func TestQualityConstants(t *testing.T) {
	if scorer.QualityLow != 1 {
		t.Errorf("QualityLow should be 1, got %d", scorer.QualityLow)
	}
	if scorer.QualityMedium != 2 {
		t.Errorf("QualityMedium should be 2, got %d", scorer.QualityMedium)
	}
	if scorer.QualityHigh != 3 {
		t.Errorf("QualityHigh should be 3, got %d", scorer.QualityHigh)
	}
}
