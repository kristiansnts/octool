package internal_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/kristiansnts/octool/internal/arms"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

func openTestDB(t *testing.T) *storage.DB {
	t.Helper()
	path := fmt.Sprintf("/tmp/octool_arms_test_%d.db", os.Getpid())
	db, err := storage.OpenAt(path)
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}

func TestFilemapArmCreatesEntry(t *testing.T) {
	db := openTestDB(t)
	tr := tracker.New()
	tr.StartSession("s1", "/proj", "new")

	// Read the same file 4 times (threshold is >3).
	for range 4 {
		tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})
	}

	result := arms.RunFilemapArm(db, tr, "/proj")
	if result.Action != "auto_saved" {
		t.Fatalf("expected auto_saved, got %q", result.Action)
	}

	entries, err := db.GetContextEntries("/proj", "file-map")
	if err != nil {
		t.Fatalf("GetContextEntries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one file-map entry to be created")
	}
}

func TestFilemapArmNoEntryBelow3(t *testing.T) {
	db := openTestDB(t)
	tr := tracker.New()
	tr.StartSession("s2", "/proj", "new")

	for range 3 {
		tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/utils.go"})
	}

	result := arms.RunFilemapArm(db, tr, "/proj")
	if result.Action != "" {
		t.Fatalf("expected no action for 3 reads, got %q", result.Action)
	}
}

func TestBuildWatchArmCoaches(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("s3", "/proj", "new")

	// Trigger 3 build cycles.
	for range 3 {
		tr.RecordToolCall(tracker.ToolCall{Tool: "edit", Args: "main.go", Result: "ok"})
		tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go build", Result: "ok"})
		tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go test", Result: "FAIL: test failed"})
	}

	result := arms.RunBuildWatchArm(tr, "/proj")
	if result.SystemMessage == "" {
		t.Fatal("expected coaching message after 3 build cycles")
	}
	if result.Arm != "buildwatch" {
		t.Fatalf("expected arm=buildwatch, got %q", result.Arm)
	}
}

func TestBuildWatchArmFiresOnce(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("s4", "/proj", "new")

	for range 3 {
		tr.RecordToolCall(tracker.ToolCall{Tool: "edit", Args: "main.go", Result: "ok"})
		tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go build", Result: "ok"})
		tr.RecordToolCall(tracker.ToolCall{Tool: "bash", Args: "go test", Result: "FAIL"})
	}

	arms.RunBuildWatchArm(tr, "/proj") // first call sets CoachSent
	result := arms.RunBuildWatchArm(tr, "/proj")
	if result.SystemMessage != "" {
		t.Fatal("buildwatch should not fire twice in the same session")
	}
}

func TestRecoveryArmOnResume(t *testing.T) {
	db := openTestDB(t)
	db.SaveContextEntry(storage.ContextEntry{
		ProjectPath: "/proj", Type: "file-map",
		Title: "File map: main.go", Content: "Frequently accessed: main.go",
		Source: "octool-auto",
	})

	tr := tracker.New()
	tr.StartSession("s5", "/proj", "resume")

	result := arms.RunRecoveryArm(db, tr, "/proj", "resume")
	if result.Action != "injected" {
		t.Fatalf("expected injected, got %q", result.Action)
	}
	if result.SystemMessage == "" {
		t.Fatal("expected non-empty system message")
	}
}

func TestRecoveryArmOnNew(t *testing.T) {
	db := openTestDB(t)
	tr := tracker.New()
	tr.StartSession("s6", "/proj", "new")

	result := arms.RunRecoveryArm(db, tr, "/proj", "new")
	if result.Action != "" {
		t.Fatalf("recovery arm should not fire on new session, got action=%q", result.Action)
	}
}

func TestConventionArmOnStill(t *testing.T) {
	db := openTestDB(t)
	tr := tracker.New()
	tr.StartSession("s7", "/proj", "new")

	var result arms.Result
	for range 3 {
		result = arms.RunConventionArm(db, tr, "/proj", "it's still not working")
	}
	if result.Action != "suggested" {
		t.Fatalf("expected suggested after 3 still prompts, got %q", result.Action)
	}
}

func TestConventionArmNoTrigger(t *testing.T) {
	db := openTestDB(t)
	tr := tracker.New()
	tr.StartSession("s8", "/proj", "new")

	result := arms.RunConventionArm(db, tr, "/proj", "add a button to the header")
	if result.Action != "" {
		t.Fatalf("expected no action for normal prompt, got %q", result.Action)
	}
}

func TestPromptCoachLow(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("s9", "/proj", "new")

	result := arms.RunPromptCoachArm(tr, "/proj", "fix it")
	if result.Action != "suggested" {
		t.Fatalf("expected suggested for low-quality prompt, got %q", result.Action)
	}
	if result.SystemMessage == "" {
		t.Fatal("expected non-empty tip")
	}
}

func TestPromptCoachHigh(t *testing.T) {
	tr := tracker.New()
	tr.StartSession("s10", "/proj", "new")

	result := arms.RunPromptCoachArm(tr, "/proj",
		"The function in /src/auth/login.go is returning a 401 error when the token is valid. "+
			"Please check the token validation logic and fix the issue.")
	if result.Action != "" {
		t.Fatalf("expected no action for high-quality prompt, got %q", result.Action)
	}
}

func TestResumeArmNewSession(t *testing.T) {
	db := openTestDB(t)
	db.SaveSessionMetrics(storage.SessionMetrics{
		SessionID:   "prev",
		ProjectPath: "/proj",
		TotalTools:  25,
	})

	tr := tracker.New()
	tr.StartSession("s11", "/proj", "new")

	result := arms.RunResumeArm(db, tr, "/proj", "new")
	if result.Action != "suggested" {
		t.Fatalf("expected suggested for heavy last session, got %q", result.Action)
	}
}

func TestResumeArmLightSession(t *testing.T) {
	db := openTestDB(t)
	db.SaveSessionMetrics(storage.SessionMetrics{
		SessionID:   "prev2",
		ProjectPath: "/proj2",
		TotalTools:  5,
	})

	tr := tracker.New()
	tr.StartSession("s12", "/proj2", "new")

	result := arms.RunResumeArm(db, tr, "/proj2", "new")
	if result.Action != "" {
		t.Fatalf("expected no action for light last session, got %q", result.Action)
	}
}

func TestViewEditArmHighRatio(t *testing.T) {
	db := openTestDB(t)
	db.SaveContextEntry(storage.ContextEntry{
		ProjectPath: "/proj", Type: "file-map",
		Title: "File map: main.go", Content: "Frequently accessed: main.go",
		Source: "octool-auto",
	})

	tr := tracker.New()
	tr.StartSession("s13", "/proj", "new")
	// 8 views, 1 edit → ratio = 8/2 = 4.0 (well above 0.7)
	for range 8 {
		tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})
	}
	tr.RecordToolCall(tracker.ToolCall{Tool: "edit", Args: "main.go"})

	result := arms.RunViewEditArm(db, tr, "/proj")
	if result.Action != "injected" {
		t.Fatalf("expected injected for high view:edit ratio, got %q", result.Action)
	}
}

func TestViewEditArmLowRatio(t *testing.T) {
	db := openTestDB(t)
	tr := tracker.New()
	tr.StartSession("s14", "/proj", "new")
	// 1 view, 5 edits → ratio = 1/6 ≈ 0.17
	tr.RecordToolCall(tracker.ToolCall{Tool: "view", Args: "/src/main.go"})
	for range 5 {
		tr.RecordToolCall(tracker.ToolCall{Tool: "edit", Args: "main.go"})
	}

	result := arms.RunViewEditArm(db, tr, "/proj")
	if result.Action != "" {
		t.Fatalf("expected no action for low view:edit ratio, got %q", result.Action)
	}
}
