package internal_test

import (
	"testing"

	"github.com/kristiansnts/octool/internal/tracker"
)

func TestStartSession(t *testing.T) {
	tr := tracker.New()
	s := tr.StartSession("sess-1", "/my/project", "new")

	if s == nil {
		t.Fatal("StartSession returned nil")
	}
	if s.ID != "sess-1" {
		t.Errorf("ID: got %q, want %q", s.ID, "sess-1")
	}
	if s.ProjectPath != "/my/project" {
		t.Errorf("ProjectPath: got %q, want %q", s.ProjectPath, "/my/project")
	}
	if s.Source != "new" {
		t.Errorf("Source: got %q, want %q", s.Source, "new")
	}
	if s.FileReads == nil {
		t.Error("FileReads map should be initialised")
	}
}

func TestGetSession(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-2", "/proj", "resume")

	got := tr.GetSession("sess-2")
	if got == nil {
		t.Fatal("GetSession returned nil for known session")
	}
	if got.ID != "sess-2" {
		t.Errorf("ID: got %q, want %q", got.ID, "sess-2")
	}

	missing := tr.GetSession("does-not-exist")
	if missing != nil {
		t.Error("GetSession should return nil for unknown ID")
	}
}

func TestCurrentSession(t *testing.T) {
	tr := tracker.New()

	if tr.CurrentSession() != nil {
		t.Error("CurrentSession should be nil before any session is started")
	}

	tr.StartSession("sess-3", "/proj", "new")
	tr.StartSession("sess-4", "/proj", "new")

	cur := tr.CurrentSession()
	if cur == nil {
		t.Fatal("CurrentSession returned nil after sessions were started")
	}
	if cur.ID != "sess-4" {
		t.Errorf("CurrentSession.ID: got %q, want %q", cur.ID, "sess-4")
	}
}

func TestRecordToolCallCounters(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-5", "/proj", "new")

	calls := []tracker.ToolCall{
		{Tool: "view", Args: "/some/file.go"},
		{Tool: "view", Args: "/other/file.go"},
		{Tool: "edit", Args: ""},
		{Tool: "bash", Args: "go build"},
		{Tool: "run_bash", Args: "go test"},
		{Tool: "grep", Args: "pattern"},
		{Tool: "glob", Args: "*.go"},
		{Tool: "create", Args: "new.go"},
		{Tool: "write", Args: "other.go"},
	}
	for _, tc := range calls {
		tr.RecordToolCall(tc)
	}

	s := tr.CurrentSession()
	if s.Views != 2 {
		t.Errorf("Views: got %d, want 2", s.Views)
	}
	if s.Edits != 1 {
		t.Errorf("Edits: got %d, want 1", s.Edits)
	}
	if s.Bash != 2 {
		t.Errorf("Bash: got %d, want 2", s.Bash)
	}
	if s.Grep != 1 {
		t.Errorf("Grep: got %d, want 1", s.Grep)
	}
	if s.Glob != 1 {
		t.Errorf("Glob: got %d, want 1", s.Glob)
	}
	if s.Creates != 2 {
		t.Errorf("Creates: got %d, want 2", s.Creates)
	}
	if s.TotalTools != len(calls) {
		t.Errorf("TotalTools: got %d, want %d", s.TotalTools, len(calls))
	}
}

func TestRecordToolCallFileReads(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-6", "/proj", "new")

	tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/util.go"})

	s := tr.CurrentSession()
	if s.FileReads["/src/main.go"] != 2 {
		t.Errorf("FileReads[main.go]: got %d, want 2", s.FileReads["/src/main.go"])
	}
	if s.FileReads["/src/util.go"] != 1 {
		t.Errorf("FileReads[util.go]: got %d, want 1", s.FileReads["/src/util.go"])
	}
}

func TestBuildCycleDetection(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-7", "/proj", "new")

	// Pattern: edit → bash(ok) → bash(fail) should trigger a build cycle.
	tr.RecordToolCall(tracker.ToolCall{Tool: "edit", Args: "main.go", Result: "ok"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go build", Result: "ok"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go test", Result: "FAIL: test failed"})

	s := tr.CurrentSession()
	if s.BuildCycles != 1 {
		t.Errorf("BuildCycles: got %d, want 1", s.BuildCycles)
	}
}

func TestBuildCycleNotTriggeredWithoutError(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-8", "/proj", "new")

	// edit → bash → bash but no fail/error in last result.
	tr.RecordToolCall(tracker.ToolCall{Tool: "edit", Args: "main.go", Result: "ok"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go build", Result: "ok"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go test", Result: "PASS"})

	s := tr.CurrentSession()
	if s.BuildCycles != 0 {
		t.Errorf("BuildCycles: got %d, want 0", s.BuildCycles)
	}
}

func TestRecordPrompt(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-9", "/proj", "new")

	tr.RecordPrompt("LOW")
	tr.RecordPrompt("LOW")
	tr.RecordPrompt("MEDIUM")
	tr.RecordPrompt("HIGH")

	s := tr.CurrentSession()
	if s.PromptCount != 4 {
		t.Errorf("PromptCount: got %d, want 4", s.PromptCount)
	}
	if s.PromptLow != 2 {
		t.Errorf("PromptLow: got %d, want 2", s.PromptLow)
	}
	if s.PromptMid != 1 {
		t.Errorf("PromptMid: got %d, want 1", s.PromptMid)
	}
	if s.PromptHigh != 1 {
		t.Errorf("PromptHigh: got %d, want 1", s.PromptHigh)
	}
}

func TestViewEditRatio(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-10", "/proj", "new")

	// 0 views, 0 edits → 0/(0+1) = 0.0
	if r := tr.ViewEditRatio(); r != 0.0 {
		t.Errorf("ratio with no calls: got %f, want 0.0", r)
	}

	tr.RecordToolCall(tracker.ToolCall{Tool: "view"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "view"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "edit"})

	// 2 views, 1 edit → 2/(1+1) = 1.0
	if r := tr.ViewEditRatio(); r != 1.0 {
		t.Errorf("ratio: got %f, want 1.0", r)
	}
}

func TestIncrStill(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("sess-11", "/proj", "new")

	tr.IncrStill()
	tr.IncrStill()

	s := tr.CurrentSession()
	if s.StillCount != 2 {
		t.Errorf("StillCount: got %d, want 2", s.StillCount)
	}
}

func TestNoOpWithoutSession(t *testing.T) {
	tr := tracker.New()

	// All of these should be safe no-ops.
	tr.RecordToolCall(tracker.ToolCall{Tool: "view"})
	tr.RecordPrompt("HIGH")
	tr.IncrStill()

	if r := tr.ViewEditRatio(); r != 0 {
		t.Errorf("ViewEditRatio without session: got %f, want 0", r)
	}
}
