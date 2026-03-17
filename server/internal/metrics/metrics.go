// Package metrics computes estimated token-waste from a tracked session.
package metrics

import (
	"github.com/kristiansnts/octool/internal/tracker"
)

const (
	tokensPerRedundantRead = 1500
	tokensPerBuildCycle    = 2000
	tokensPerStillFollowup = 3000
	tokensPerLowPrompt     = 1000
)

// WasteBreakdown holds the per-category token-waste estimates for a session.
type WasteBreakdown struct {
	FileReads      int // tokens wasted on redundant file reads
	BuildCycles    int // tokens wasted on build cycles
	StillFollowups int // tokens wasted on "still not working" loops
	LowPrompts     int // tokens wasted due to vague prompts
	Total          int
}

// ComputeWaste calculates estimated wasted tokens from a session.
func ComputeWaste(s *tracker.Session) WasteBreakdown {
	if s == nil {
		return WasteBreakdown{}
	}

	// Sum up redundant reads: each file read more than once contributes (count-1)*1500.
	fileReadWaste := 0
	for _, count := range s.FileReads {
		if count > 1 {
			fileReadWaste += (count - 1) * tokensPerRedundantRead
		}
	}

	buildCycleWaste := s.BuildCycles * tokensPerBuildCycle
	stillWaste := s.StillCount * tokensPerStillFollowup
	lowPromptWaste := s.PromptLow * tokensPerLowPrompt

	total := fileReadWaste + buildCycleWaste + stillWaste + lowPromptWaste

	return WasteBreakdown{
		FileReads:      fileReadWaste,
		BuildCycles:    buildCycleWaste,
		StillFollowups: stillWaste,
		LowPrompts:     lowPromptWaste,
		Total:          total,
	}
}

// GetRedundantReads returns a map of file path → read count for files that
// were read more than threshold times.
func GetRedundantReads(s *tracker.Session, threshold int) map[string]int {
	result := make(map[string]int)
	if s == nil {
		return result
	}
	for path, count := range s.FileReads {
		if count > threshold {
			result[path] = count
		}
	}
	return result
}
