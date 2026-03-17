// Package session parses ~/.copilot/session-state/ files and extracts
// context entries (file-map, convention, gotcha) into the octool database.
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// ParsedEntry is a context entry extracted from a session file.
type ParsedEntry struct {
	Type    string // "file-map" | "convention" | "gotcha"
	Title   string
	Content string
}

// ParseResult is the result of parsing one session file.
type ParseResult struct {
	SessionFile  string
	ProjectPath  string
	EntriesFound []ParsedEntry
}

// Options controls what fetch-session imports.
type Options struct {
	Limit       int    // 0 = use default (5)
	ProjectPath string // "" = all projects
	All         bool   // true = no limit
	DryRun      bool
}

// ---------------------------------------------------------------------------
// JSONL event schema (Copilot CLI real format)
// ---------------------------------------------------------------------------

type jsonlEvent struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp any             `json:"timestamp"`
}

type sessionStartData struct {
	SessionID string `json:"sessionId"`
	StartTime string `json:"startTime"`
}

type toolStartData struct {
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments"`
}

type toolCompleteData struct {
	ToolName string `json:"toolName"`
	Output   string `json:"output"`
}

type messageData struct {
	Role    string `json:"role"`    // for legacy format
	Content any    `json:"content"` // string or array
	Text    string `json:"text"`    // some events use "text"
}

// parsedSession is the in-memory representation of a parsed .jsonl file.
type parsedSession struct {
	SessionID   string
	ProjectPath string // may be empty — Copilot CLI doesn't always store cwd
	ToolCalls   []toolCall
	UserMsgs    []string
	AsstMsgs    []string
}

type toolCall struct {
	Name      string
	Arguments map[string]any
	Output    string
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// Parser scans ~/.copilot/session-state/ and extracts context entries.
type Parser struct {
	db       *storage.DB
	log      *logger.Logger
	stateDir string
}

// defaultStateDir returns ~/.copilot/session-state/.
func defaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".copilot", "session-state")
}

// New creates a Parser using the default ~/.copilot/session-state/ directory.
func New(db *storage.DB) *Parser {
	return NewWithStateDir(db, defaultStateDir())
}

// NewWithStateDir creates a Parser using a custom state directory (for testing).
func NewWithStateDir(db *storage.DB, stateDir string) *Parser {
	return &Parser{
		db:       db,
		log:      logger.New("session-parser"),
		stateDir: stateDir,
	}
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

// Run executes the fetch-session import.
// Returns: sessions scanned, entries created, duplicates skipped, error.
func (p *Parser) Run(opts Options) (scanned, created, skipped int, err error) {
	dirEntries, err := os.ReadDir(p.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			p.log.Warn(fmt.Sprintf("state dir not found: %s", p.stateDir))
			return 0, 0, 0, nil
		}
		return 0, 0, 0, fmt.Errorf("read state dir: %w", err)
	}

	// Collect only .jsonl files.
	var files []string
	for _, e := range dirEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(p.stateDir, e.Name()))
		}
	}

	// Apply limit.
	limit := opts.Limit
	if limit <= 0 {
		limit = 5
	}
	if !opts.All && len(files) > limit {
		// Take the last N (most recent by filesystem order).
		files = files[len(files)-limit:]
	}

	for _, path := range files {
		imported, ierr := p.db.IsSessionImported(path)
		if ierr != nil {
			p.log.Warn(fmt.Sprintf("IsSessionImported %s: %v", path, ierr))
		}
		if imported {
			skipped++
			continue
		}

		result, perr := parseJSONLFile(path)
		if perr != nil {
			p.log.Warn(fmt.Sprintf("parse %s: %v", path, perr))
			continue
		}
		scanned++

		if opts.ProjectPath != "" && result.ProjectPath != opts.ProjectPath {
			if merr := p.db.MarkSessionImported(path, result.ProjectPath, 0); merr != nil {
				p.log.Warn(fmt.Sprintf("MarkSessionImported: %v", merr))
			}
			continue
		}

		if opts.DryRun {
			created += len(result.EntriesFound)
			continue
		}

		for _, entry := range result.EntriesFound {
			_, serr := p.db.SaveContextEntry(storage.ContextEntry{
				ProjectPath: result.ProjectPath,
				Type:        entry.Type,
				Title:       entry.Title,
				Content:     entry.Content,
				Source:      "fetch-session",
			})
			if serr != nil {
				p.log.Error(fmt.Sprintf("SaveContextEntry: %v", serr))
				continue
			}
			created++
		}

		if merr := p.db.MarkSessionImported(path, result.ProjectPath, len(result.EntriesFound)); merr != nil {
			p.log.Warn(fmt.Sprintf("MarkSessionImported %s: %v", path, merr))
		}
	}

	p.log.Info(fmt.Sprintf("Run: scanned=%d created=%d skipped=%d dryRun=%v", scanned, created, skipped, opts.DryRun))
	return scanned, created, skipped, nil
}

// ---------------------------------------------------------------------------
// Parse a single .jsonl file
// ---------------------------------------------------------------------------

func parseJSONLFile(path string) (ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return ParseResult{}, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	var ps parsedSession
	var lastUserMsg string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1 MB per line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev jsonlEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue // skip malformed lines
		}

		switch ev.Type {
		case "session.start":
			var d sessionStartData
			if err := json.Unmarshal(ev.Data, &d); err == nil {
				ps.SessionID = d.SessionID
			}

		case "tool.execution_start":
			var d toolStartData
			if err := json.Unmarshal(ev.Data, &d); err == nil {
				ps.ToolCalls = append(ps.ToolCalls, toolCall{
					Name:      d.ToolName,
					Arguments: d.Arguments,
				})
			}

		case "tool.execution_complete":
			var d toolCompleteData
			if err := json.Unmarshal(ev.Data, &d); err == nil && len(ps.ToolCalls) > 0 {
				// Attach output to last matching tool call.
				for i := len(ps.ToolCalls) - 1; i >= 0; i-- {
					if ps.ToolCalls[i].Name == d.ToolName && ps.ToolCalls[i].Output == "" {
						ps.ToolCalls[i].Output = d.Output
						break
					}
				}
			}

		case "user.message":
			text := extractMessageText(ev.Data)
			if text != "" {
				ps.UserMsgs = append(ps.UserMsgs, text)
				lastUserMsg = text
			}

		case "assistant.message":
			text := extractMessageText(ev.Data)
			if text != "" {
				ps.AsstMsgs = append(ps.AsstMsgs, text)
				_ = lastUserMsg // used by gotcha extraction below
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ParseResult{}, fmt.Errorf("scan: %w", err)
	}

	// Infer project path from file paths if not set by the session.
	if ps.ProjectPath == "" {
		ps.ProjectPath = inferProjectPath(ps.ToolCalls)
	}

	result := ParseResult{
		SessionFile: path,
		ProjectPath: ps.ProjectPath,
	}
	result.EntriesFound = append(result.EntriesFound, extractFileMapsFromSession(ps)...)
	result.EntriesFound = append(result.EntriesFound, extractConventionsFromSession(ps)...)
	result.EntriesFound = append(result.EntriesFound, extractGotchasFromSession(ps)...)
	// Also mine rich structured findings from assistant message text (research sessions).
	result.EntriesFound = append(result.EntriesFound, extractFromAssistantMessages(ps)...)

	return result, nil
}

// extractMessageText pulls the text content from a user/assistant message event.
func extractMessageText(raw json.RawMessage) string {
	var d messageData
	if err := json.Unmarshal(raw, &d); err != nil {
		return ""
	}
	if d.Text != "" {
		return d.Text
	}
	switch v := d.Content.(type) {
	case string:
		return v
	case []any:
		// Content blocks: [{type:"text", text:"..."}]
		var parts []string
		for _, block := range v {
			if m, ok := block.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

// ---------------------------------------------------------------------------
// Extraction rule 0: mine structured findings from assistant messages
// ---------------------------------------------------------------------------
// This handles research sessions where the user asked Claude to analyze a
// project (e.g. via `copilot resume --session-id`) and Claude wrote back
// structured findings. We look for common patterns Claude uses when writing
// file maps, conventions, and gotchas.

func extractFromAssistantMessages(ps parsedSession) []ParsedEntry {
	var entries []ParsedEntry
	seen := map[string]bool{}

	for _, msg := range ps.AsstMsgs {
		// Skip short messages — not worth mining.
		if len(msg) < 200 {
			continue
		}
		// Cap message size to avoid OOM on huge responses.
		if len(msg) > 8000 {
			msg = msg[:8000]
		}
		entries = append(entries, mineFileMapsFromText(msg, ps.ProjectPath, seen)...)
		entries = append(entries, mineConventionsFromText(msg, seen)...)
		entries = append(entries, mineGotchasFromText(msg, seen)...)
		// Cap total entries per session to avoid DB bloat.
		if len(entries) >= 30 {
			break
		}
	}
	return entries
}

// mineFileMapsFromText extracts file-map entries from assistant prose.
// Patterns it recognises:
//   - Markdown bold/header followed by a file path:  **`path/to/file`** — description
//   - Bullet points with paths:  - `path/to/file` — description
//   - Lines starting with a path followed by colon:  path/to/file: description
func mineFileMapsFromText(text, projectPath string, seen map[string]bool) []ParsedEntry {
	var entries []ParsedEntry
	lines := strings.SplitSeq(text, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		// Extract anything that looks like a file path (abs or relative with ext)
		path, desc := extractPathAndDesc(line)
		if path == "" {
			continue
		}
		key := "file-map:" + path
		if seen[key] {
			continue
		}
		seen[key] = true
		content := "Key file: " + path
		if desc != "" {
			content += " — " + desc
		}
		if projectPath != "" {
			content += " (project: " + projectPath + ")"
		}
		entries = append(entries, ParsedEntry{
			Type:    "file-map",
			Title:   "File map: " + path,
			Content: content,
		})
	}
	return entries
}

// mineConventionsFromText extracts convention entries from assistant prose.
// Patterns:
//   - Lines starting with "Always", "Never", "Convention:", "Rule:", "Note:"
//   - Markdown bold phrases like **Always use X**
func mineConventionsFromText(text string, seen map[string]bool) []ParsedEntry {
	var entries []ParsedEntry
	prefixes := []string{
		"always ", "never ", "convention:", "rule:", "important:",
		"must ", "should always", "**always", "**never",
	}
	lines := strings.SplitSeq(text, "\n")
	for line := range lines {
		clean := strings.TrimSpace(line)
		clean = strings.TrimLeft(clean, "-*# ")
		lower := strings.ToLower(clean)
		matched := false
		for _, p := range prefixes {
			if strings.HasPrefix(lower, p) {
				matched = true
				break
			}
		}
		if !matched || len(clean) < 20 {
			continue
		}
		snippet := clean
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		key := "convention:" + snippet[:min(60, len(snippet))]
		if seen[key] {
			continue
		}
		seen[key] = true
		entries = append(entries, ParsedEntry{
			Type:    "convention",
			Title:   "Convention: " + snippet[:min(60, len(snippet))],
			Content: snippet,
		})
	}
	return entries
}

// mineGotchasFromText extracts gotcha/warning entries from assistant prose.
// Patterns:
//   - Lines starting with "Gotcha:", "Warning:", "Watch out", "Be careful", "Note:"
//   - Markdown ⚠️ or 🚨 emoji lines
func mineGotchasFromText(text string, seen map[string]bool) []ParsedEntry {
	var entries []ParsedEntry
	prefixes := []string{
		"gotcha:", "warning:", "watch out", "be careful", "⚠️", "🚨",
		"pitfall:", "known issue:", "caveat:", "important gotcha",
	}
	lines := strings.SplitSeq(text, "\n")
	for line := range lines {
		clean := strings.TrimSpace(line)
		clean = strings.TrimLeft(clean, "-*# ")
		lower := strings.ToLower(clean)
		matched := false
		for _, p := range prefixes {
			if strings.HasPrefix(lower, p) || strings.Contains(lower[:min(30, len(lower))], p) {
				matched = true
				break
			}
		}
		if !matched || len(clean) < 20 {
			continue
		}
		snippet := clean
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		key := "gotcha:" + snippet[:min(60, len(snippet))]
		if seen[key] {
			continue
		}
		seen[key] = true
		entries = append(entries, ParsedEntry{
			Type:    "gotcha",
			Title:   "Gotcha: " + snippet[:min(60, len(snippet))],
			Content: snippet,
		})
	}
	return entries
}

// extractPathAndDesc tries to pull a file path and optional description from a
// line of markdown text. Returns ("", "") if no path-like token is found.
func extractPathAndDesc(line string) (path, desc string) {
	// Strip markdown syntax: **, *, `, #, -, numbers
	clean := strings.TrimLeft(line, " \t-*#`1234567890.")
	clean = strings.ReplaceAll(clean, "`", "")
	clean = strings.ReplaceAll(clean, "**", "")
	clean = strings.ReplaceAll(clean, "__", "")
	clean = strings.TrimSpace(clean)

	// Look for a path-like token (must have / or .ext, length > 4)
	for f := range strings.FieldsSeq(clean) {
		f = strings.Trim(f, "(),;:")
		if len(f) < 5 {
			continue
		}
		isPath := (strings.Contains(f, "/") && !strings.HasPrefix(f, "http")) ||
			(strings.Contains(f, ".") && !strings.HasPrefix(f, "-") &&
				strings.Count(f, ".") <= 3 && !strings.Contains(f, "@"))
		if !isPath {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f))
		knownExts := map[string]bool{
			".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
			".php": true, ".py": true, ".rb": true, ".java": true, ".kt": true,
			".swift": true, ".rs": true, ".cs": true, ".cpp": true, ".c": true,
			".h": true, ".sh": true, ".yaml": true, ".yml": true, ".json": true,
			".toml": true, ".env": true, ".sql": true, ".blade": true, ".vue": true,
		}
		hasAbsPath := strings.HasPrefix(f, "/")
		if !hasAbsPath && !knownExts[ext] {
			continue
		}
		path = f
		// Rest of the line after the path token is the description.
		idx := strings.Index(clean, f)
		if idx >= 0 {
			rest := strings.TrimSpace(clean[idx+len(f):])
			rest = strings.TrimLeft(rest, "-—:. ")
			if len(rest) > 10 {
				desc = rest
			}
		}
		return
	}
	return "", ""
}

// ---------------------------------------------------------------------------
// Extraction rule 1: file-map entries
// ---------------------------------------------------------------------------

func extractFileMapsFromSession(ps parsedSession) []ParsedEntry {
	counts := map[string]int{}
	for _, tc := range ps.ToolCalls {
		name := strings.ToLower(tc.Name)
		if name != "view" && name != "read" && name != "read_file" {
			continue
		}
		// Extract path from arguments map.
		path := argPath(tc.Arguments)
		if path != "" {
			counts[path]++
		}
	}

	var entries []ParsedEntry
	for path, count := range counts {
		if count > 3 {
			entries = append(entries, ParsedEntry{
				Type:    "file-map",
				Title:   "File map: " + path,
				Content: fmt.Sprintf("Frequently read file: %s (read %d times in this session)", path, count),
			})
		}
	}
	return entries
}

// inferProjectPath guesses the project root from the most-common directory
// prefix across all viewed file paths. It walks up the path until it finds a
// level shared by the majority of files.
func inferProjectPath(calls []toolCall) string {
	freq := map[string]int{}
	for _, tc := range calls {
		name := strings.ToLower(tc.Name)
		if name != "view" && name != "read" && name != "read_file" && name != "edit" && name != "bash" {
			continue
		}
		p := argPath(tc.Arguments)
		if p == "" || !filepath.IsAbs(p) {
			continue
		}
		// Walk up and count each ancestor directory.
		dir := filepath.Dir(p)
		for dir != "/" && dir != "." {
			freq[dir]++
			dir = filepath.Dir(dir)
		}
	}
	if len(freq) == 0 {
		return ""
	}
	// Find the deepest directory that appears for at least 2 files.
	best := ""
	bestDepth := 0
	for dir, count := range freq {
		if count < 2 {
			continue
		}
		depth := strings.Count(dir, string(filepath.Separator))
		if depth > bestDepth {
			bestDepth = depth
			best = dir
		}
	}
	return best
}

// argPath extracts the "path", "file_path", or "filename" value from tool arguments.
func argPath(args map[string]any) string {
	if args == nil {
		return ""
	}
	for _, key := range []string{"path", "file_path", "filename", "file"} {
		if v, ok := args[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Extraction rule 2: convention hints
// ---------------------------------------------------------------------------

var conventionKeywords = []string{"always", "never", "must", "should", "convention"}

func containsConventionKeyword(s string) bool {
	lower := strings.ToLower(s)
	for _, kw := range conventionKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func extractConventionsFromSession(ps parsedSession) []ParsedEntry {
	var matches []string
	for _, msg := range ps.UserMsgs {
		if containsConventionKeyword(msg) {
			matches = append(matches, msg)
		}
	}
	if len(matches) < 2 {
		return nil
	}

	var entries []ParsedEntry
	for _, m := range matches {
		snippet := m
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		entries = append(entries, ParsedEntry{
			Type:    "convention",
			Title:   "Convention from session",
			Content: snippet,
		})
	}
	return entries
}

// ---------------------------------------------------------------------------
// Extraction rule 3: gotcha entries (error → fix sequences)
// ---------------------------------------------------------------------------

func extractGotchasFromSession(ps parsedSession) []ParsedEntry {
	var entries []ParsedEntry
	userMsgs := ps.UserMsgs
	asstMsgs := ps.AsstMsgs

	// Pair each user message with the next assistant message.
	maxPairs := min(len(userMsgs), len(asstMsgs))

	for i := range maxPairs {
		userMsg := userMsgs[i]
		asstMsg := asstMsgs[i]

		userLower := strings.ToLower(userMsg)
		if !strings.Contains(userLower, "error") && !strings.Contains(userLower, "failed") &&
			!strings.Contains(userLower, "not working") {
			continue
		}

		asstLower := strings.ToLower(asstMsg)
		if !strings.Contains(asstLower, "fixed") && !strings.Contains(asstLower, "solution") &&
			!strings.Contains(asstLower, "resolved") {
			continue
		}

		title := userMsg
		if len(title) > 60 {
			title = title[:60]
		}
		fixSummary := asstMsg
		if len(fixSummary) > 200 {
			fixSummary = fixSummary[:200]
		}

		entries = append(entries, ParsedEntry{
			Type:    "gotcha",
			Title:   "Gotcha: " + title,
			Content: userMsg + " → " + fixSummary,
		})
	}
	return entries
}
