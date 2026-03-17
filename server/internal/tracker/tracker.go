// Package tracker provides in-memory session state tracking for octool.
package tracker

import (
	"strings"
	"sync"
	"time"
)

// ToolCall represents a single tool invocation recorded in a session.
type ToolCall struct {
	Tool      string
	Args      string
	Result    string
	Timestamp int64
}

// Session holds the in-memory state for an active Claude session.
type Session struct {
	ID          string
	ProjectPath string
	Source      string // "new" | "resume"
	StartedAt   int64
	ToolCalls   []ToolCall

	// Counters
	Views      int
	Edits      int
	Bash       int
	Grep       int
	Glob       int
	Creates    int
	TotalTools int

	// File read counts: map[filePath]readCount
	FileReads map[string]int

	// Build cycle detection
	BuildCycles int

	// Prompt tracking
	PromptCount int
	PromptLow   int
	PromptMid   int
	PromptHigh  int

	// "still" followups counter
	StillCount int

	// CoachSent prevents buildwatch from firing more than once per session
	CoachSent bool
}

// Tracker manages in-memory sessions.
type Tracker struct {
	mu       sync.Mutex
	sessions map[string]*Session
	current  *Session
}

// New creates a new Tracker with an empty session map.
func New() *Tracker {
	return &Tracker{
		sessions: make(map[string]*Session),
	}
}

// StartSession creates and registers a new session, setting it as current.
func (t *Tracker) StartSession(id, projectPath, source string) *Session {
	s := &Session{
		ID:          id,
		ProjectPath: projectPath,
		Source:      source,
		StartedAt:   time.Now().UnixMilli(),
		FileReads:   make(map[string]int),
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.sessions[id] = s
	t.current = s
	return s
}

// GetSession returns the session with the given ID, or nil if not found.
func (t *Tracker) GetSession(id string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.sessions[id]
}

// CurrentSession returns the most recently started session, or nil if none.
func (t *Tracker) CurrentSession() *Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current
}

// RecordToolCall appends a tool call to the current session and updates counters.
// If there is no current session, the call is a no-op.
func (t *Tracker) RecordToolCall(tc ToolCall) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.current
	if s == nil {
		return
	}

	if tc.Timestamp == 0 {
		tc.Timestamp = time.Now().UnixMilli()
	}

	s.ToolCalls = append(s.ToolCalls, tc)
	s.TotalTools++

	tool := strings.ToLower(tc.Tool)
	switch tool {
	case "view":
		s.Views++
		// Track file reads: extract path from args.
		path := extractPath(tc.Args)
		if path != "" {
			s.FileReads[path]++
		}
	case "edit":
		s.Edits++
	case "bash", "run_bash":
		s.Bash++
	case "grep":
		s.Grep++
	case "glob":
		s.Glob++
	case "create", "write":
		s.Creates++
	}

	// Build cycle detection: edit → bash → bash(result contains "fail" or "error")
	n := len(s.ToolCalls)
	if n >= 3 {
		prev3 := s.ToolCalls[n-3]
		prev2 := s.ToolCalls[n-2]
		prev1 := s.ToolCalls[n-1]

		t1 := strings.ToLower(prev3.Tool)
		t2 := strings.ToLower(prev2.Tool)
		t3 := strings.ToLower(prev1.Tool)

		isBash2 := t2 == "bash" || t2 == "run_bash"
		isBash3 := t3 == "bash" || t3 == "run_bash"

		if t1 == "edit" && isBash2 && isBash3 {
			res := strings.ToLower(prev1.Result)
			if strings.Contains(res, "fail") || strings.Contains(res, "error") {
				s.BuildCycles++
			}
		}
	}
}

// RecordPrompt records a prompt quality label into the current session.
// quality must be "LOW", "MEDIUM", or "HIGH".
func (t *Tracker) RecordPrompt(quality string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.current
	if s == nil {
		return
	}

	s.PromptCount++
	switch strings.ToUpper(quality) {
	case "LOW":
		s.PromptLow++
	case "MEDIUM":
		s.PromptMid++
	case "HIGH":
		s.PromptHigh++
	}
}

// IncrStill increments the "still" followup counter on the current session.
func (t *Tracker) IncrStill() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.current != nil {
		t.current.StillCount++
	}
}

// ViewEditRatio returns Views / (Edits + 1) for the current session to avoid
// division by zero. Returns 0 if there is no current session.
func (t *Tracker) ViewEditRatio() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.current
	if s == nil {
		return 0
	}
	return float64(s.Views) / float64(s.Edits+1)
}

// extractPath attempts to pull a file path from an args string.
// It returns the first token that looks like a path (contains "/" or ".").
func extractPath(args string) string {
	// Try to find a token that looks like a filesystem path.
	for f := range strings.FieldsSeq(args) {
		if strings.Contains(f, "/") || (strings.Contains(f, ".") && !strings.HasPrefix(f, "-")) {
			return f
		}
	}
	return ""
}
