package internal_test

import (
	"testing"

	"github.com/kristiansnts/octool/internal/metrics"
	"github.com/kristiansnts/octool/internal/tracker"
)

func TestComputeWasteNilSession(t *testing.T) {
	wb := metrics.ComputeWaste(nil)
	if wb.Total != 0 {
		t.Errorf("ComputeWaste(nil).Total: got %d, want 0", wb.Total)
	}
}

func TestComputeWasteFileReads(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("m-1", "/proj", "new")

	// Read the same file 3 times → (3-1)*1500 = 3000 waste.
	tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})

	s := tr.CurrentSession()
	wb := metrics.ComputeWaste(s)

	if wb.FileReads != 3000 {
		t.Errorf("FileReads waste: got %d, want 3000", wb.FileReads)
	}
	if wb.Total != 3000 {
		t.Errorf("Total waste: got %d, want 3000", wb.Total)
	}
}

func TestComputeWasteBuildCycles(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("m-2", "/proj", "new")

	// Trigger one build cycle.
	tr.RecordToolCall(tracker.ToolCall{Tool: "edit", Args: "main.go", Result: "ok"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go build", Result: "ok"})
	tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go test", Result: "FAIL: test failed"})

	s := tr.CurrentSession()
	wb := metrics.ComputeWaste(s)

	if wb.BuildCycles != 2000 {
		t.Errorf("BuildCycles waste: got %d, want 2000", wb.BuildCycles)
	}
}

func TestComputeWasteStillFollowups(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("m-3", "/proj", "new")
	tr.IncrStill()
	tr.IncrStill()

	s := tr.CurrentSession()
	wb := metrics.ComputeWaste(s)

	if wb.StillFollowups != 6000 {
		t.Errorf("StillFollowups waste: got %d, want 6000", wb.StillFollowups)
	}
}

func TestComputeWasteLowPrompts(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("m-4", "/proj", "new")
	tr.RecordPrompt("LOW")
	tr.RecordPrompt("LOW")
	tr.RecordPrompt("LOW")

	s := tr.CurrentSession()
	wb := metrics.ComputeWaste(s)

	if wb.LowPrompts != 3000 {
		t.Errorf("LowPrompts waste: got %d, want 3000", wb.LowPrompts)
	}
}

func TestComputeWasteTotal(t *testing.T) {
	// Manually construct a session to set exact values.
	s := &tracker.Session{
		ID:          "manual",
		ProjectPath: "/proj",
		FileReads: map[string]int{
			"/a.go": 3, // (3-1)*1500 = 3000
			"/b.go": 1, // no waste
		},
		BuildCycles: 2,  // 2*2000 = 4000
		StillCount:  1,  // 1*3000 = 3000
		PromptLow:   4,  // 4*1000 = 4000
	}

	wb := metrics.ComputeWaste(s)

	wantFileReads := 3000
	wantBuild := 4000
	wantStill := 3000
	wantLow := 4000
	wantTotal := wantFileReads + wantBuild + wantStill + wantLow

	if wb.FileReads != wantFileReads {
		t.Errorf("FileReads: got %d, want %d", wb.FileReads, wantFileReads)
	}
	if wb.BuildCycles != wantBuild {
		t.Errorf("BuildCycles: got %d, want %d", wb.BuildCycles, wantBuild)
	}
	if wb.StillFollowups != wantStill {
		t.Errorf("StillFollowups: got %d, want %d", wb.StillFollowups, wantStill)
	}
	if wb.LowPrompts != wantLow {
		t.Errorf("LowPrompts: got %d, want %d", wb.LowPrompts, wantLow)
	}
	if wb.Total != wantTotal {
		t.Errorf("Total: got %d, want %d", wb.Total, wantTotal)
	}
}

func TestGetRedundantReads(t *testing.T) {
	s := &tracker.Session{
		ID:        "rr-test",
		FileReads: map[string]int{
			"/a.go": 1,
			"/b.go": 2,
			"/c.go": 5,
		},
	}

	// threshold=1 → files read more than 1 time.
	result := metrics.GetRedundantReads(s, 1)
	if _, ok := result["/a.go"]; ok {
		t.Error("/a.go should not be in redundant reads (count=1, threshold=1)")
	}
	if result["/b.go"] != 2 {
		t.Errorf("/b.go: got %d, want 2", result["/b.go"])
	}
	if result["/c.go"] != 5 {
		t.Errorf("/c.go: got %d, want 5", result["/c.go"])
	}
}

func TestGetRedundantReadsNilSession(t *testing.T) {
	result := metrics.GetRedundantReads(nil, 1)
	if result == nil {
		t.Error("GetRedundantReads(nil) should return empty map, not nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestGetRedundantReadsHighThreshold(t *testing.T) {
	s := &tracker.Session{
		ID:        "rr-high",
		FileReads: map[string]int{"/a.go": 3},
	}

	// threshold=3 → need count > 3, so nothing should appear.
	result := metrics.GetRedundantReads(s, 3)
	if len(result) != 0 {
		t.Errorf("expected empty result with threshold=3 and max count=3, got %v", result)
	}

	// threshold=2 → count 3 > 2, so /a.go should appear.
	result = metrics.GetRedundantReads(s, 2)
	if result["/a.go"] != 3 {
		t.Errorf("/a.go with threshold=2: got %d, want 3", result["/a.go"])
	}
}
