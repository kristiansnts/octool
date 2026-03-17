package storage_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/kristiansnts/octool/internal/storage"
)

// tempDB opens a fresh test database and returns it along with a cleanup func.
func tempDB(t *testing.T) (*storage.DB, func()) {
	t.Helper()
	path := fmt.Sprintf("/tmp/octool_test_%s.db", strconv.Itoa(os.Getpid()))
	db, err := storage.OpenAt(path)
	if err != nil {
		t.Fatalf("OpenAt(%q): %v", path, err)
	}
	cleanup := func() {
		db.Close()
		os.Remove(path)
	}
	return db, cleanup
}

// ---------------------------------------------------------------------------
// Context entries
// ---------------------------------------------------------------------------

func TestSaveAndGetContextEntry(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	entry := storage.ContextEntry{
		ProjectPath:   "/my/project",
		Type:          "decision",
		Title:         "Use SQLite",
		Content:       "We chose SQLite for simplicity.",
		Source:        "manual",
		StalenessRisk: "low",
		Priority:      "high",
	}

	id, err := db.SaveContextEntry(entry)
	if err != nil {
		t.Fatalf("SaveContextEntry: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	entries, err := db.GetContextEntries("/my/project", "")
	if err != nil {
		t.Fatalf("GetContextEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Title != entry.Title {
		t.Errorf("Title: want %q, got %q", entry.Title, got.Title)
	}
	if got.Content != entry.Content {
		t.Errorf("Content: want %q, got %q", entry.Content, got.Content)
	}
	if got.Type != entry.Type {
		t.Errorf("Type: want %q, got %q", entry.Type, got.Type)
	}
	if got.StalenessRisk != entry.StalenessRisk {
		t.Errorf("StalenessRisk: want %q, got %q", entry.StalenessRisk, got.StalenessRisk)
	}
	if got.Priority != entry.Priority {
		t.Errorf("Priority: want %q, got %q", entry.Priority, got.Priority)
	}

	// Filter by type — should still return the one entry.
	filtered, err := db.GetContextEntries("/my/project", "decision")
	if err != nil {
		t.Fatalf("GetContextEntries (filtered): %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered entry, got %d", len(filtered))
	}

	// Filter by wrong type — should return nothing.
	none, err := db.GetContextEntries("/my/project", "architecture")
	if err != nil {
		t.Fatalf("GetContextEntries (no match): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected 0 entries, got %d", len(none))
	}
}

func TestSearchContextEntries(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	entries := []storage.ContextEntry{
		{ProjectPath: "/p", Type: "note", Title: "Goroutine patterns", Content: "Use channels for communication."},
		{ProjectPath: "/p", Type: "note", Title: "Error handling", Content: "Always wrap errors with fmt.Errorf."},
		{ProjectPath: "/p", Type: "note", Title: "Database choice", Content: "We use SQLite for local storage."},
	}
	for _, e := range entries {
		if _, err := db.SaveContextEntry(e); err != nil {
			t.Fatalf("SaveContextEntry: %v", err)
		}
	}

	results, err := db.SearchContextEntries("SQLite")
	if err != nil {
		t.Fatalf("SearchContextEntries: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 FTS result, got %d", len(results))
	}
	if results[0].Title != "Database choice" {
		t.Errorf("unexpected result title: %q", results[0].Title)
	}

	// Search by title word.
	results2, err := db.SearchContextEntries("Goroutine")
	if err != nil {
		t.Fatalf("SearchContextEntries (goroutine): %v", err)
	}
	if len(results2) != 1 {
		t.Errorf("expected 1 result for 'Goroutine', got %d", len(results2))
	}
}

func TestDeleteContextEntry(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	id, err := db.SaveContextEntry(storage.ContextEntry{
		ProjectPath: "/p", Type: "note", Title: "To delete", Content: "temporary",
	})
	if err != nil {
		t.Fatalf("SaveContextEntry: %v", err)
	}

	if err := db.DeleteContextEntry(id); err != nil {
		t.Fatalf("DeleteContextEntry: %v", err)
	}

	entries, err := db.GetContextEntries("/p", "")
	if err != nil {
		t.Fatalf("GetContextEntries after delete: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(entries))
	}
}

func TestUpdateContextEntry(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	id, err := db.SaveContextEntry(storage.ContextEntry{
		ProjectPath: "/p", Type: "note", Title: "Original", Content: "original content",
	})
	if err != nil {
		t.Fatalf("SaveContextEntry: %v", err)
	}

	updated := storage.ContextEntry{
		ID:          id,
		ProjectPath: "/p",
		Type:        "decision",
		Title:       "Updated",
		Content:     "updated content",
		Source:      "import",
		Priority:    "high",
	}
	if err := db.UpdateContextEntry(updated); err != nil {
		t.Fatalf("UpdateContextEntry: %v", err)
	}

	entries, err := db.GetContextEntries("/p", "")
	if err != nil {
		t.Fatalf("GetContextEntries after update: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Title != "Updated" {
		t.Errorf("Title: want %q, got %q", "Updated", entries[0].Title)
	}
	if entries[0].Type != "decision" {
		t.Errorf("Type: want %q, got %q", "decision", entries[0].Type)
	}
}

// ---------------------------------------------------------------------------
// Session metrics
// ---------------------------------------------------------------------------

func TestSaveAndGetSessionMetrics(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := storage.SessionMetrics{
		SessionID:            "sess-abc-123",
		ProjectPath:          "/my/project",
		Source:               "new",
		DurationSeconds:      3600,
		TotalViews:           50,
		TotalEdits:           10,
		TotalBash:            5,
		TotalGrep:            8,
		TotalGlob:            3,
		TotalCreate:          2,
		TotalTools:           78,
		ViewEditRatio:        5.0,
		RedundantReads:       `{"file.go": 3}`,
		BuildCycles:          2,
		StillFollowups:       1,
		PromptCount:          20,
		PromptLow:            10,
		PromptMedium:         8,
		PromptHigh:           2,
		EstimatedWasteTokens: 1500,
		WasteBreakdown:       `{"redundant": 1000, "build": 500}`,
	}

	id, err := db.SaveSessionMetrics(m)
	if err != nil {
		t.Fatalf("SaveSessionMetrics: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	got, err := db.GetSessionMetrics("sess-abc-123")
	if err != nil {
		t.Fatalf("GetSessionMetrics: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil SessionMetrics")
	}
	if got.SessionID != m.SessionID {
		t.Errorf("SessionID: want %q, got %q", m.SessionID, got.SessionID)
	}
	if got.DurationSeconds != m.DurationSeconds {
		t.Errorf("DurationSeconds: want %d, got %d", m.DurationSeconds, got.DurationSeconds)
	}
	if got.TotalViews != m.TotalViews {
		t.Errorf("TotalViews: want %d, got %d", m.TotalViews, got.TotalViews)
	}
	if got.ViewEditRatio != m.ViewEditRatio {
		t.Errorf("ViewEditRatio: want %f, got %f", m.ViewEditRatio, got.ViewEditRatio)
	}
	if got.WasteBreakdown != m.WasteBreakdown {
		t.Errorf("WasteBreakdown: want %q, got %q", m.WasteBreakdown, got.WasteBreakdown)
	}

	// GetSessionMetrics for unknown ID should return nil.
	missing, err := db.GetSessionMetrics("nonexistent")
	if err != nil {
		t.Fatalf("GetSessionMetrics (missing): %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for missing session, got %+v", missing)
	}

	// UpdateSessionMetrics.
	got.TotalViews = 99
	got.DurationSeconds = 7200
	if err := db.UpdateSessionMetrics(*got); err != nil {
		t.Fatalf("UpdateSessionMetrics: %v", err)
	}
	updated, err := db.GetSessionMetrics("sess-abc-123")
	if err != nil {
		t.Fatalf("GetSessionMetrics after update: %v", err)
	}
	if updated.TotalViews != 99 {
		t.Errorf("TotalViews after update: want 99, got %d", updated.TotalViews)
	}

	// GetRecentSessionMetrics.
	recent, err := db.GetRecentSessionMetrics("/my/project", 10)
	if err != nil {
		t.Fatalf("GetRecentSessionMetrics: %v", err)
	}
	if len(recent) != 1 {
		t.Errorf("expected 1 recent metric, got %d", len(recent))
	}
}

// ---------------------------------------------------------------------------
// Arm activity
// ---------------------------------------------------------------------------

func TestLogArmActivity(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	activities := []storage.ArmActivity{
		{SessionID: "s1", ProjectPath: "/proj", Arm: "tracker", Action: "track_tool", Detail: "Read file.go"},
		{SessionID: "s1", ProjectPath: "/proj", Arm: "scorer", Action: "score_session", Detail: "score=0.85"},
		{SessionID: "s2", ProjectPath: "/other", Arm: "dashboard", Action: "render", Detail: "ok"},
	}

	for _, a := range activities {
		if err := db.LogArmActivity(a); err != nil {
			t.Fatalf("LogArmActivity: %v", err)
		}
	}

	results, err := db.GetArmActivity("/proj", 10)
	if err != nil {
		t.Fatalf("GetArmActivity: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 activities for /proj, got %d", len(results))
	}

	// Respect limit.
	limited, err := db.GetArmActivity("/proj", 1)
	if err != nil {
		t.Fatalf("GetArmActivity (limited): %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("expected 1 activity with limit=1, got %d", len(limited))
	}
}

// ---------------------------------------------------------------------------
// Imported sessions
// ---------------------------------------------------------------------------

func TestImportedSessions(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	file := "/tmp/session-2024-01-01.jsonl"

	imported, err := db.IsSessionImported(file)
	if err != nil {
		t.Fatalf("IsSessionImported (before): %v", err)
	}
	if imported {
		t.Error("expected false before marking imported")
	}

	if err := db.MarkSessionImported(file, "/my/project", 42); err != nil {
		t.Fatalf("MarkSessionImported: %v", err)
	}

	imported, err = db.IsSessionImported(file)
	if err != nil {
		t.Fatalf("IsSessionImported (after): %v", err)
	}
	if !imported {
		t.Error("expected true after marking imported")
	}

	// Idempotent: marking the same file twice should not error (INSERT OR IGNORE).
	if err := db.MarkSessionImported(file, "/my/project", 10); err != nil {
		t.Fatalf("MarkSessionImported (duplicate): %v", err)
	}
}
