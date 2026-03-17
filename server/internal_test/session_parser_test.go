package internal_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kristiansnts/octool/internal/session"
	"github.com/kristiansnts/octool/internal/storage"
)

// openSessionTestDB creates an isolated SQLite database for session tests.
func openSessionTestDB(t *testing.T) *storage.DB {
	t.Helper()
	path := fmt.Sprintf("/tmp/octool_session_test_%d_%s.db", os.Getpid(), t.Name())
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

// jsonlEvent is one line in a .jsonl session file.
type jsonlEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// writeJSONLFixture writes a slice of events as a .jsonl file and returns the dir.
func writeJSONLFixture(t *testing.T, events []jsonlEvent) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session-001.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			t.Fatalf("encode event: %v", err)
		}
	}
	return dir
}

// standardJSONLFixture builds the canonical test session in real Copilot CLI format.
func standardJSONLFixture() []jsonlEvent {
	return []jsonlEvent{
		{
			Type: "session.start",
			Data: map[string]any{"sessionId": "test-session-001", "startTime": "2026-03-17T00:00:00Z"},
		},
		// 4 view calls on /src/auth.go → should produce a file-map entry
		{Type: "tool.execution_start", Data: map[string]any{"toolName": "view", "arguments": map[string]any{"path": "/src/auth.go"}}},
		{Type: "tool.execution_complete", Data: map[string]any{"toolName": "view", "output": "..."}},
		{Type: "tool.execution_start", Data: map[string]any{"toolName": "view", "arguments": map[string]any{"path": "/src/auth.go"}}},
		{Type: "tool.execution_complete", Data: map[string]any{"toolName": "view", "output": "..."}},
		{Type: "tool.execution_start", Data: map[string]any{"toolName": "view", "arguments": map[string]any{"path": "/src/auth.go"}}},
		{Type: "tool.execution_complete", Data: map[string]any{"toolName": "view", "output": "..."}},
		{Type: "tool.execution_start", Data: map[string]any{"toolName": "view", "arguments": map[string]any{"path": "/src/auth.go"}}},
		{Type: "tool.execution_complete", Data: map[string]any{"toolName": "view", "output": "..."}},
		// 1 view on /src/utils.go — only once, should NOT produce a file-map
		{Type: "tool.execution_start", Data: map[string]any{"toolName": "view", "arguments": map[string]any{"path": "/src/utils.go"}}},
		{Type: "tool.execution_complete", Data: map[string]any{"toolName": "view", "output": "..."}},
		// User error → assistant fix → should produce a gotcha entry
		{Type: "user.message", Data: map[string]any{"content": "there's an error in auth.go: undefined variable"}},
		{Type: "assistant.message", Data: map[string]any{"content": "Fixed: added the missing variable declaration. The solution is to declare the variable before use."}},
	}
}

// TestParserExtractsFilemap verifies that 4 reads of the same file produce a file-map entry.
func TestParserExtractsFilemap(t *testing.T) {
	db := openSessionTestDB(t)
	dir := writeJSONLFixture(t, standardJSONLFixture())

	p := session.NewWithStateDir(db, dir)
	_, created, _, err := p.Run(session.Options{All: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if created == 0 {
		t.Fatal("expected at least one entry to be created")
	}

	entries, err := db.GetContextEntries("", "file-map")
	if err != nil {
		t.Fatalf("GetContextEntries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected a file-map entry for /src/auth.go (read 4 times)")
	}

	found := false
	for _, e := range entries {
		if e.Title == "File map: /src/auth.go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected file-map entry titled 'File map: /src/auth.go', got %+v", entries)
	}
}

// TestParserGotcha verifies that an error→fix message sequence produces a gotcha entry.
func TestParserGotcha(t *testing.T) {
	db := openSessionTestDB(t)
	dir := writeJSONLFixture(t, standardJSONLFixture())

	p := session.NewWithStateDir(db, dir)
	_, created, _, err := p.Run(session.Options{All: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if created == 0 {
		t.Fatal("expected entries to be created")
	}

	entries, err := db.GetContextEntries("", "gotcha")
	if err != nil {
		t.Fatalf("GetContextEntries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one gotcha entry from error→fix sequence")
	}
}

// TestParserDryRun verifies that dry-run mode counts entries but does not persist anything.
func TestParserDryRun(t *testing.T) {
	db := openSessionTestDB(t)
	dir := writeJSONLFixture(t, standardJSONLFixture())

	p := session.NewWithStateDir(db, dir)
	_, created, _, err := p.Run(session.Options{All: true, DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if created == 0 {
		t.Fatal("expected non-zero created count in dry-run mode")
	}

	entries, err := db.GetContextEntries("", "")
	if err != nil {
		t.Fatalf("GetContextEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected zero DB entries in dry-run mode, got %d", len(entries))
	}
}

// TestParserSkipsAlreadyImported verifies that a second run skips already-imported files.
func TestParserSkipsAlreadyImported(t *testing.T) {
	db := openSessionTestDB(t)
	dir := writeJSONLFixture(t, standardJSONLFixture())

	p := session.NewWithStateDir(db, dir)

	scanned1, created1, skipped1, err := p.Run(session.Options{All: true})
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if scanned1 == 0 {
		t.Fatal("first run: expected scanned > 0")
	}
	if created1 == 0 {
		t.Fatal("first run: expected created > 0")
	}
	if skipped1 != 0 {
		t.Fatalf("first run: expected skipped=0, got %d", skipped1)
	}

	scanned2, created2, skipped2, err := p.Run(session.Options{All: true})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if created2 != 0 {
		t.Fatalf("second run: expected created=0, got %d", created2)
	}
	if skipped2 != 1 {
		t.Fatalf("second run: expected skipped=1, got %d (scanned=%d)", skipped2, scanned2)
	}
}
