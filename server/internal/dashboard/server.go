// Package dashboard provides an HTTP dashboard server for octool.
package dashboard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
)

//go:embed templates/index.html
var indexHTML []byte

// Server is the HTTP dashboard server.
type Server struct {
	db   *storage.DB
	log  *logger.Logger
	port int
}

// New creates a new dashboard Server backed by the given DB.
func New(db *storage.DB, port int) *Server {
	return &Server{
		db:   db,
		log:  logger.New("dashboard"),
		port: port,
	}
}

// Start registers routes and starts the HTTP server on s.port (blocks).
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/entries", s.handleEntries)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/arm-activity", s.handleArmActivity)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/hot-files", s.handleHotFiles)
	mux.HandleFunc("/api/entries/", s.handleDeleteEntry)

	addr := fmt.Sprintf(":%d", s.port)
	s.log.Info(fmt.Sprintf("dashboard listening on %s", addr))
	return http.ListenAndServe(addr, mux)
}

// writeJSON writes v as JSON to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// dbPath returns the default DB path string for display.
func dbPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.octool/octool.db"
	}
	return filepath.Join(home, ".octool", "octool.db")
}

// handleIndex serves the embedded dashboard HTML.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.log.Info(fmt.Sprintf("GET / from %s", r.RemoteAddr))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

// handleStatus returns a JSON summary of the database state.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.log.Info(fmt.Sprintf("GET /api/status from %s", r.RemoteAddr))

	entries, err := s.db.GetContextEntries("", "")
	if err != nil {
		s.log.Error(fmt.Sprintf("handleStatus GetContextEntries: %v", err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	sessions, err := s.db.GetRecentSessionMetrics("", 100)
	if err != nil {
		s.log.Error(fmt.Sprintf("handleStatus GetRecentSessionMetrics: %v", err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"db":       dbPath(),
		"entries":  len(entries),
		"sessions": len(sessions),
		"arms":     8,
	})
}

// handleEntries returns a JSON array of context entries filtered by optional
// project and type query params.
func (s *Server) handleEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")
	entryType := r.URL.Query().Get("type")
	s.log.Info(fmt.Sprintf("GET /api/entries project=%q type=%q from %s", project, entryType, r.RemoteAddr))

	entries, err := s.db.GetContextEntries(project, entryType)
	if err != nil {
		s.log.Error(fmt.Sprintf("handleEntries: %v", err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if entries == nil {
		entries = []storage.ContextEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// handleSessions returns a JSON array of recent session metrics.
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	s.log.Info(fmt.Sprintf("GET /api/sessions project=%q limit=%d from %s", project, limit, r.RemoteAddr))

	sessions, err := s.db.GetRecentSessionMetrics(project, limit)
	if err != nil {
		s.log.Error(fmt.Sprintf("handleSessions: %v", err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if sessions == nil {
		sessions = []storage.SessionMetrics{}
	}
	writeJSON(w, http.StatusOK, sessions)
}

// handleArmActivity returns a JSON array of recent arm activity records.
func (s *Server) handleArmActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	s.log.Info(fmt.Sprintf("GET /api/arm-activity project=%q limit=%d from %s", project, limit, r.RemoteAddr))

	activity, err := s.db.GetArmActivity(project, limit)
	if err != nil {
		s.log.Error(fmt.Sprintf("handleArmActivity: %v", err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if activity == nil {
		activity = []storage.ArmActivity{}
	}
	writeJSON(w, http.StatusOK, activity)
}

// handleProjects returns a sorted JSON array of unique non-empty project paths.
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.GetContextEntries("", "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	seen := map[string]struct{}{}
	var projects []string
	for _, e := range entries {
		if e.ProjectPath != "" {
			if _, ok := seen[e.ProjectPath]; !ok {
				seen[e.ProjectPath] = struct{}{}
				projects = append(projects, e.ProjectPath)
			}
		}
	}
	if projects == nil {
		projects = []string{}
	}
	writeJSON(w, http.StatusOK, projects)
}

// logLine represents one parsed line from a log file.
type logLine struct {
	Raw  string `json:"raw"`
	File string `json:"file"`
}

// handleHotFiles returns the most-accessed files for a project.
func (s *Server) handleHotFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	project := r.URL.Query().Get("project")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
		limit = n
	}
	files, err := s.db.GetHotFiles(project, 1, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if files == nil {
		files = []storage.FileAccess{}
	}
	writeJSON(w, http.StatusOK, files)
}

// handleLogs returns recent lines from today's hook and arm-activity log files.
// Query params: file=hooks|arm|octool (default hooks), lines=N (default 200).
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".octool", "logs")
	today := time.Now().UTC().Format("2006-01-02")

	fileParam := r.URL.Query().Get("file")
	linesParam := r.URL.Query().Get("lines")
	maxLines := 200
	if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
		maxLines = n
	}

	fileMap := map[string]string{
		"hooks":  filepath.Join(logDir, "hooks-"+today+".log"),
		"arm":    filepath.Join(logDir, "arm-activity-"+today+".log"),
		"octool": filepath.Join(logDir, "octool-"+today+".log"),
	}

	// Determine which files to read
	var files []string
	if name, ok := fileMap[fileParam]; ok {
		files = []string{name}
	} else {
		// Default: hooks + arm
		files = []string{fileMap["hooks"], fileMap["arm"]}
	}

	var lines []logLine
	for _, path := range files {
		fname := filepath.Base(path)
		f, err := os.Open(path)
		if err != nil {
			continue // file may not exist yet — that's fine
		}
		var fileLines []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fileLines = append(fileLines, sc.Text())
		}
		f.Close()
		// Take last maxLines from this file
		start := len(fileLines) - maxLines
		if start < 0 {
			start = 0
		}
		for _, l := range fileLines[start:] {
			if strings.TrimSpace(l) != "" {
				lines = append(lines, logLine{Raw: l, File: fname})
			}
		}
	}

	writeJSON(w, http.StatusOK, lines)
}

// handleDeleteEntry handles DELETE /api/entries/{id}.
func (s *Server) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path: /api/entries/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/entries/")
	idStr = strings.TrimSuffix(idStr, "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.log.Warn(fmt.Sprintf("handleDeleteEntry: invalid id %q: %v", idStr, err))
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	s.log.Info(fmt.Sprintf("DELETE /api/entries/%d from %s", id, r.RemoteAddr))

	if err := s.db.DeleteContextEntry(id); err != nil {
		s.log.Error(fmt.Sprintf("handleDeleteEntry id=%d: %v", id, err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
