// Package storage provides SQLite-backed persistence for octool.
// The database lives at ~/.octool/octool.db.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kristiansnts/octool/internal/logger"
	_ "modernc.org/sqlite" // register "sqlite" driver
)

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// ContextEntry represents a single piece of stored project context.
type ContextEntry struct {
	ID            int64
	ProjectPath   string
	Type          string
	Title         string
	Content       string
	Source        string
	StalenessRisk string
	Priority      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SessionMetrics holds per-session token-efficiency statistics.
type SessionMetrics struct {
	ID                   int64
	SessionID            string
	ProjectPath          string
	Source               string
	CreatedAt            time.Time
	DurationSeconds      int
	TotalViews           int
	TotalEdits           int
	TotalBash            int
	TotalGrep            int
	TotalGlob            int
	TotalCreate          int
	TotalTools           int
	ViewEditRatio        float64
	RedundantReads       string // JSON blob
	BuildCycles          int
	StillFollowups       int
	PromptCount          int
	PromptLow            int
	PromptMedium         int
	PromptHigh           int
	EstimatedWasteTokens int
	WasteBreakdown       string // JSON blob
}

// ArmActivity records a single ARM action event.
type ArmActivity struct {
	ID          int64
	SessionID   string
	ProjectPath string
	Arm         string
	Action      string
	Detail      string
	CreatedAt   time.Time
}

// FileAccess tracks cumulative read/edit counts per file across all sessions.
type FileAccess struct {
	ID          int64
	ProjectPath string
	FilePath    string
	ReadCount   int
	EditCount   int
	LastSeen    time.Time
}

// ImportedSession tracks which session files have already been ingested.
type ImportedSession struct {
	ID             int64
	SessionFile    string
	ProjectPath    string
	EntriesCreated int
	ImportedAt     time.Time
}

// ---------------------------------------------------------------------------
// DB
// ---------------------------------------------------------------------------

// DB wraps a *sql.DB and attaches a logger for all operations.
type DB struct {
	db  *sql.DB
	log *logger.Logger
}

// defaultDBPath returns ~/.octool/octool.db.
func defaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, ".octool")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create octool dir: %w", err)
	}
	return filepath.Join(dir, "octool.db"), nil
}

// Open opens the default database at ~/.octool/octool.db and runs migrations.
func Open() (*DB, error) {
	path, err := defaultDBPath()
	if err != nil {
		return nil, err
	}
	return OpenAt(path)
}

// OpenAt opens (or creates) a SQLite database at the given path and runs
// migrations. This function is exported primarily to support tests.
func OpenAt(path string) (*DB, error) {
	log := logger.New("storage")

	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		log.Error(fmt.Sprintf("open db at %s: %v", path, err))
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Enable WAL mode and foreign keys for better concurrency.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
	} {
		if _, err := sqlDB.Exec(pragma); err != nil {
			log.Error(fmt.Sprintf("set pragma %q: %v", pragma, err))
		}
	}

	d := &DB{db: sqlDB, log: log}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}

	log.Info(fmt.Sprintf("database opened: %s", path))
	return d, nil
}

// Close releases the underlying database connection.
func (d *DB) Close() error {
	d.log.Info("closing database")
	return d.db.Close()
}

// ---------------------------------------------------------------------------
// Migrations
// ---------------------------------------------------------------------------

func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS context_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_path TEXT NOT NULL,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			source TEXT DEFAULT 'manual',
			staleness_risk TEXT DEFAULT 'medium',
			priority TEXT DEFAULT 'normal',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS session_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			project_path TEXT NOT NULL,
			source TEXT DEFAULT 'new',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			duration_seconds INTEGER DEFAULT 0,
			total_views INTEGER DEFAULT 0,
			total_edits INTEGER DEFAULT 0,
			total_bash INTEGER DEFAULT 0,
			total_grep INTEGER DEFAULT 0,
			total_glob INTEGER DEFAULT 0,
			total_create INTEGER DEFAULT 0,
			total_tools INTEGER DEFAULT 0,
			view_edit_ratio REAL DEFAULT 0.0,
			redundant_reads TEXT DEFAULT '{}',
			build_cycles INTEGER DEFAULT 0,
			still_followups INTEGER DEFAULT 0,
			prompt_count INTEGER DEFAULT 0,
			prompt_low INTEGER DEFAULT 0,
			prompt_medium INTEGER DEFAULT 0,
			prompt_high INTEGER DEFAULT 0,
			estimated_waste_tokens INTEGER DEFAULT 0,
			waste_breakdown TEXT DEFAULT '{}'
		);`,

		`CREATE TABLE IF NOT EXISTS arm_activity (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT,
			project_path TEXT NOT NULL,
			arm TEXT NOT NULL,
			action TEXT NOT NULL,
			detail TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS imported_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_file TEXT NOT NULL UNIQUE,
			project_path TEXT,
			entries_created INTEGER DEFAULT 0,
			imported_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS file_access (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_path TEXT NOT NULL,
			file_path TEXT NOT NULL,
			read_count INTEGER DEFAULT 0,
			edit_count INTEGER DEFAULT 0,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_path, file_path)
		);`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS context_entries_fts USING fts5(
			title, content, content=context_entries, content_rowid=id
		);`,
	}

	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			d.log.Error(fmt.Sprintf("migration failed: %v", err))
			return fmt.Errorf("migration: %w", err)
		}
	}

	d.log.Info("migrations applied successfully")
	return nil
}

// ---------------------------------------------------------------------------
// Context entries CRUD
// ---------------------------------------------------------------------------

// SaveContextEntry inserts a new ContextEntry and returns its row ID.
func (d *DB) SaveContextEntry(e ContextEntry) (int64, error) {
	query := `INSERT INTO context_entries
		(project_path, type, title, content, source, staleness_risk, priority)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	src := coalesce(e.Source, "manual")
	sr := coalesce(e.StalenessRisk, "medium")
	pri := coalesce(e.Priority, "normal")

	res, err := d.db.Exec(query, e.ProjectPath, e.Type, e.Title, e.Content, src, sr, pri)
	if err != nil {
		d.log.Error(fmt.Sprintf("SaveContextEntry: %v", err))
		return 0, fmt.Errorf("save context entry: %w", err)
	}

	id, _ := res.LastInsertId()

	// Keep FTS index in sync.
	if _, ferr := d.db.Exec(
		`INSERT INTO context_entries_fts(rowid, title, content) VALUES (?, ?, ?)`,
		id, e.Title, e.Content,
	); ferr != nil {
		d.log.Warn(fmt.Sprintf("SaveContextEntry FTS sync: %v", ferr))
	}

	d.log.Info(fmt.Sprintf("SaveContextEntry: id=%d project=%s type=%s", id, e.ProjectPath, e.Type))
	return id, nil
}

// GetContextEntries retrieves context entries, optionally filtered by
// projectPath and/or entryType. Empty string means "no filter".
func (d *DB) GetContextEntries(projectPath, entryType string) ([]ContextEntry, error) {
	query := `SELECT id, project_path, type, title, content, source,
		staleness_risk, priority, created_at, updated_at
		FROM context_entries WHERE 1=1`
	args := []any{}

	if projectPath != "" {
		query += " AND project_path = ?"
		args = append(args, projectPath)
	}
	if entryType != "" {
		query += " AND type = ?"
		args = append(args, entryType)
	}
	query += " ORDER BY created_at DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		d.log.Error(fmt.Sprintf("GetContextEntries: %v", err))
		return nil, fmt.Errorf("get context entries: %w", err)
	}
	defer rows.Close()

	entries, err := scanContextEntries(rows)
	if err != nil {
		d.log.Error(fmt.Sprintf("GetContextEntries scan: %v", err))
		return nil, err
	}

	d.log.Info(fmt.Sprintf("GetContextEntries: returned %d entries", len(entries)))
	return entries, nil
}

// SearchContextEntries performs a full-text search across title and content.
func (d *DB) SearchContextEntries(query string) ([]ContextEntry, error) {
	ftsQuery := `SELECT c.id, c.project_path, c.type, c.title, c.content,
		c.source, c.staleness_risk, c.priority, c.created_at, c.updated_at
		FROM context_entries c
		JOIN context_entries_fts fts ON c.id = fts.rowid
		WHERE context_entries_fts MATCH ?
		ORDER BY rank`

	rows, err := d.db.Query(ftsQuery, query)
	if err != nil {
		d.log.Error(fmt.Sprintf("SearchContextEntries: %v", err))
		return nil, fmt.Errorf("search context entries: %w", err)
	}
	defer rows.Close()

	entries, err := scanContextEntries(rows)
	if err != nil {
		d.log.Error(fmt.Sprintf("SearchContextEntries scan: %v", err))
		return nil, err
	}

	d.log.Info(fmt.Sprintf("SearchContextEntries query=%q: returned %d entries", query, len(entries)))
	return entries, nil
}

// DeleteContextEntry removes a context entry by ID.
func (d *DB) DeleteContextEntry(id int64) error {
	// Remove from FTS first.
	if _, err := d.db.Exec(`DELETE FROM context_entries_fts WHERE rowid = ?`, id); err != nil {
		d.log.Warn(fmt.Sprintf("DeleteContextEntry FTS: %v", err))
	}

	_, err := d.db.Exec(`DELETE FROM context_entries WHERE id = ?`, id)
	if err != nil {
		d.log.Error(fmt.Sprintf("DeleteContextEntry id=%d: %v", id, err))
		return fmt.Errorf("delete context entry: %w", err)
	}

	d.log.Info(fmt.Sprintf("DeleteContextEntry: id=%d", id))
	return nil
}

// UpdateContextEntry updates an existing context entry's mutable fields.
func (d *DB) UpdateContextEntry(e ContextEntry) error {
	query := `UPDATE context_entries SET
		project_path = ?, type = ?, title = ?, content = ?,
		source = ?, staleness_risk = ?, priority = ?,
		updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err := d.db.Exec(query,
		e.ProjectPath, e.Type, e.Title, e.Content,
		e.Source, e.StalenessRisk, e.Priority, e.ID)
	if err != nil {
		d.log.Error(fmt.Sprintf("UpdateContextEntry id=%d: %v", e.ID, err))
		return fmt.Errorf("update context entry: %w", err)
	}

	// Re-sync FTS.
	if _, ferr := d.db.Exec(
		`INSERT INTO context_entries_fts(context_entries_fts, rowid, title, content)
		 VALUES ('delete', ?, ?, ?)`,
		e.ID, e.Title, e.Content,
	); ferr != nil {
		d.log.Warn(fmt.Sprintf("UpdateContextEntry FTS delete: %v", ferr))
	}
	if _, ferr := d.db.Exec(
		`INSERT INTO context_entries_fts(rowid, title, content) VALUES (?, ?, ?)`,
		e.ID, e.Title, e.Content,
	); ferr != nil {
		d.log.Warn(fmt.Sprintf("UpdateContextEntry FTS insert: %v", ferr))
	}

	d.log.Info(fmt.Sprintf("UpdateContextEntry: id=%d", e.ID))
	return nil
}

// scanContextEntries scans all rows into a slice of ContextEntry.
func scanContextEntries(rows *sql.Rows) ([]ContextEntry, error) {
	var entries []ContextEntry
	for rows.Next() {
		var e ContextEntry
		err := rows.Scan(
			&e.ID, &e.ProjectPath, &e.Type, &e.Title, &e.Content,
			&e.Source, &e.StalenessRisk, &e.Priority, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan context entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ---------------------------------------------------------------------------
// Session metrics CRUD
// ---------------------------------------------------------------------------

// SaveSessionMetrics inserts a new SessionMetrics row and returns its ID.
func (d *DB) SaveSessionMetrics(m SessionMetrics) (int64, error) {
	query := `INSERT INTO session_metrics
		(session_id, project_path, source, duration_seconds,
		 total_views, total_edits, total_bash, total_grep, total_glob,
		 total_create, total_tools, view_edit_ratio, redundant_reads,
		 build_cycles, still_followups, prompt_count, prompt_low,
		 prompt_medium, prompt_high, estimated_waste_tokens, waste_breakdown)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	rr := coalesce(m.RedundantReads, "{}")
	wb := coalesce(m.WasteBreakdown, "{}")
	src := coalesce(m.Source, "new")

	res, err := d.db.Exec(query,
		m.SessionID, m.ProjectPath, src, m.DurationSeconds,
		m.TotalViews, m.TotalEdits, m.TotalBash, m.TotalGrep, m.TotalGlob,
		m.TotalCreate, m.TotalTools, m.ViewEditRatio, rr,
		m.BuildCycles, m.StillFollowups, m.PromptCount, m.PromptLow,
		m.PromptMedium, m.PromptHigh, m.EstimatedWasteTokens, wb,
	)
	if err != nil {
		d.log.Error(fmt.Sprintf("SaveSessionMetrics session=%s: %v", m.SessionID, err))
		return 0, fmt.Errorf("save session metrics: %w", err)
	}

	id, _ := res.LastInsertId()
	d.log.Info(fmt.Sprintf("SaveSessionMetrics: id=%d session=%s", id, m.SessionID))
	return id, nil
}

// GetSessionMetrics retrieves the metrics row for a specific session ID.
func (d *DB) GetSessionMetrics(sessionID string) (*SessionMetrics, error) {
	query := `SELECT id, session_id, project_path, source, created_at,
		duration_seconds, total_views, total_edits, total_bash, total_grep,
		total_glob, total_create, total_tools, view_edit_ratio, redundant_reads,
		build_cycles, still_followups, prompt_count, prompt_low, prompt_medium,
		prompt_high, estimated_waste_tokens, waste_breakdown
		FROM session_metrics WHERE session_id = ? LIMIT 1`

	row := d.db.QueryRow(query, sessionID)
	m, err := scanSessionMetrics(row)
	if err == sql.ErrNoRows {
		d.log.Info(fmt.Sprintf("GetSessionMetrics: session=%s not found", sessionID))
		return nil, nil
	}
	if err != nil {
		d.log.Error(fmt.Sprintf("GetSessionMetrics session=%s: %v", sessionID, err))
		return nil, fmt.Errorf("get session metrics: %w", err)
	}

	d.log.Info(fmt.Sprintf("GetSessionMetrics: session=%s id=%d", sessionID, m.ID))
	return m, nil
}

// UpdateSessionMetrics updates an existing session metrics row identified by SessionID.
func (d *DB) UpdateSessionMetrics(m SessionMetrics) error {
	query := `UPDATE session_metrics SET
		project_path = ?, source = ?, duration_seconds = ?,
		total_views = ?, total_edits = ?, total_bash = ?, total_grep = ?,
		total_glob = ?, total_create = ?, total_tools = ?, view_edit_ratio = ?,
		redundant_reads = ?, build_cycles = ?, still_followups = ?,
		prompt_count = ?, prompt_low = ?, prompt_medium = ?, prompt_high = ?,
		estimated_waste_tokens = ?, waste_breakdown = ?
		WHERE session_id = ?`

	rr := coalesce(m.RedundantReads, "{}")
	wb := coalesce(m.WasteBreakdown, "{}")

	_, err := d.db.Exec(query,
		m.ProjectPath, m.Source, m.DurationSeconds,
		m.TotalViews, m.TotalEdits, m.TotalBash, m.TotalGrep,
		m.TotalGlob, m.TotalCreate, m.TotalTools, m.ViewEditRatio,
		rr, m.BuildCycles, m.StillFollowups,
		m.PromptCount, m.PromptLow, m.PromptMedium, m.PromptHigh,
		m.EstimatedWasteTokens, wb, m.SessionID,
	)
	if err != nil {
		d.log.Error(fmt.Sprintf("UpdateSessionMetrics session=%s: %v", m.SessionID, err))
		return fmt.Errorf("update session metrics: %w", err)
	}

	d.log.Info(fmt.Sprintf("UpdateSessionMetrics: session=%s", m.SessionID))
	return nil
}

// GetRecentSessionMetrics returns the most recent `limit` session metrics rows
// for the given project path, ordered newest-first.
func (d *DB) GetRecentSessionMetrics(projectPath string, limit int) ([]SessionMetrics, error) {
	query := `SELECT id, session_id, project_path, source, created_at,
		duration_seconds, total_views, total_edits, total_bash, total_grep,
		total_glob, total_create, total_tools, view_edit_ratio, redundant_reads,
		build_cycles, still_followups, prompt_count, prompt_low, prompt_medium,
		prompt_high, estimated_waste_tokens, waste_breakdown
		FROM session_metrics WHERE project_path = ?
		ORDER BY created_at DESC LIMIT ?`

	rows, err := d.db.Query(query, projectPath, limit)
	if err != nil {
		d.log.Error(fmt.Sprintf("GetRecentSessionMetrics: %v", err))
		return nil, fmt.Errorf("get recent session metrics: %w", err)
	}
	defer rows.Close()

	var results []SessionMetrics
	for rows.Next() {
		m, err := scanSessionMetricsRow(rows)
		if err != nil {
			d.log.Error(fmt.Sprintf("GetRecentSessionMetrics scan: %v", err))
			return nil, err
		}
		results = append(results, *m)
	}

	d.log.Info(fmt.Sprintf("GetRecentSessionMetrics: project=%s returned %d rows", projectPath, len(results)))
	return results, rows.Err()
}

// scanSessionMetrics scans a single *sql.Row into a SessionMetrics value.
func scanSessionMetrics(row *sql.Row) (*SessionMetrics, error) {
	var m SessionMetrics
	err := row.Scan(
		&m.ID, &m.SessionID, &m.ProjectPath, &m.Source, &m.CreatedAt,
		&m.DurationSeconds, &m.TotalViews, &m.TotalEdits, &m.TotalBash, &m.TotalGrep,
		&m.TotalGlob, &m.TotalCreate, &m.TotalTools, &m.ViewEditRatio, &m.RedundantReads,
		&m.BuildCycles, &m.StillFollowups, &m.PromptCount, &m.PromptLow, &m.PromptMedium,
		&m.PromptHigh, &m.EstimatedWasteTokens, &m.WasteBreakdown,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// scanSessionMetricsRow scans a *sql.Rows row into a SessionMetrics value.
func scanSessionMetricsRow(rows *sql.Rows) (*SessionMetrics, error) {
	var m SessionMetrics
	err := rows.Scan(
		&m.ID, &m.SessionID, &m.ProjectPath, &m.Source, &m.CreatedAt,
		&m.DurationSeconds, &m.TotalViews, &m.TotalEdits, &m.TotalBash, &m.TotalGrep,
		&m.TotalGlob, &m.TotalCreate, &m.TotalTools, &m.ViewEditRatio, &m.RedundantReads,
		&m.BuildCycles, &m.StillFollowups, &m.PromptCount, &m.PromptLow, &m.PromptMedium,
		&m.PromptHigh, &m.EstimatedWasteTokens, &m.WasteBreakdown,
	)
	if err != nil {
		return nil, fmt.Errorf("scan session metrics row: %w", err)
	}
	return &m, nil
}

// ---------------------------------------------------------------------------
// Arm activity
// ---------------------------------------------------------------------------

// LogArmActivity inserts an arm activity record.
func (d *DB) LogArmActivity(a ArmActivity) error {
	query := `INSERT INTO arm_activity (session_id, project_path, arm, action, detail)
		VALUES (?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query, a.SessionID, a.ProjectPath, a.Arm, a.Action, a.Detail)
	if err != nil {
		d.log.Error(fmt.Sprintf("LogArmActivity: %v", err))
		return fmt.Errorf("log arm activity: %w", err)
	}

	d.log.Info(fmt.Sprintf("LogArmActivity: arm=%s action=%s project=%s", a.Arm, a.Action, a.ProjectPath))
	return nil
}

// GetArmActivity returns the most recent `limit` arm activity records.
// If projectPath is non-empty, results are filtered to that project.
func (d *DB) GetArmActivity(projectPath string, limit int) ([]ArmActivity, error) {
	var query string
	var rows *sql.Rows
	var err error
	if projectPath == "" {
		query = `SELECT id, session_id, project_path, arm, action, detail, created_at
			FROM arm_activity ORDER BY created_at DESC LIMIT ?`
		rows, err = d.db.Query(query, limit)
	} else {
		query = `SELECT id, session_id, project_path, arm, action, detail, created_at
			FROM arm_activity WHERE project_path = ?
			ORDER BY created_at DESC LIMIT ?`
		rows, err = d.db.Query(query, projectPath, limit)
	}

	if err != nil {
		d.log.Error(fmt.Sprintf("GetArmActivity: %v", err))
		return nil, fmt.Errorf("get arm activity: %w", err)
	}
	defer rows.Close()

	var results []ArmActivity
	for rows.Next() {
		var a ArmActivity
		var sessionID sql.NullString
		if err := rows.Scan(&a.ID, &sessionID, &a.ProjectPath, &a.Arm, &a.Action, &a.Detail, &a.CreatedAt); err != nil {
			d.log.Error(fmt.Sprintf("GetArmActivity scan: %v", err))
			return nil, fmt.Errorf("scan arm activity: %w", err)
		}
		a.SessionID = sessionID.String
		results = append(results, a)
	}

	d.log.Info(fmt.Sprintf("GetArmActivity: project=%s returned %d rows", projectPath, len(results)))
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// File access tracking
// ---------------------------------------------------------------------------

// UpsertFileAccess increments the read or edit count for a file path.
// isEdit=true increments edit_count, false increments read_count.
func (d *DB) UpsertFileAccess(projectPath, filePath string, isEdit bool) error {
	var query string
	if isEdit {
		query = `INSERT INTO file_access (project_path, file_path, edit_count, last_seen)
			VALUES (?, ?, 1, CURRENT_TIMESTAMP)
			ON CONFLICT(project_path, file_path)
			DO UPDATE SET edit_count = edit_count + 1, last_seen = CURRENT_TIMESTAMP`
	} else {
		query = `INSERT INTO file_access (project_path, file_path, read_count, last_seen)
			VALUES (?, ?, 1, CURRENT_TIMESTAMP)
			ON CONFLICT(project_path, file_path)
			DO UPDATE SET read_count = read_count + 1, last_seen = CURRENT_TIMESTAMP`
	}
	_, err := d.db.Exec(query, projectPath, filePath)
	return err
}

// GetHotFiles returns files sorted by total access count (reads + edits) for a project.
// Only files with total count >= minCount are returned.
func (d *DB) GetHotFiles(projectPath string, minCount, limit int) ([]FileAccess, error) {
	var query string
	var rows *sql.Rows
	var err error
	if projectPath == "" {
		query = `SELECT id, project_path, file_path, read_count, edit_count, last_seen
			FROM file_access
			WHERE (read_count + edit_count) >= ?
			ORDER BY (read_count + edit_count) DESC LIMIT ?`
		rows, err = d.db.Query(query, minCount, limit)
	} else {
		query = `SELECT id, project_path, file_path, read_count, edit_count, last_seen
			FROM file_access
			WHERE project_path = ? AND (read_count + edit_count) >= ?
			ORDER BY (read_count + edit_count) DESC LIMIT ?`
		rows, err = d.db.Query(query, projectPath, minCount, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("get hot files: %w", err)
	}
	defer rows.Close()

	var results []FileAccess
	for rows.Next() {
		var f FileAccess
		if err := rows.Scan(&f.ID, &f.ProjectPath, &f.FilePath, &f.ReadCount, &f.EditCount, &f.LastSeen); err != nil {
			return nil, fmt.Errorf("scan file access: %w", err)
		}
		results = append(results, f)
	}
	return results, rows.Err()
}

// GetFileAccessCount returns the read_count for a specific file in the given project.
// Returns 0 if the file has not been accessed yet.
func (d *DB) GetFileAccessCount(projectPath, filePath string) (int, error) {
	var count int
	err := d.db.QueryRow(
		`SELECT read_count FROM file_access WHERE project_path = ? AND file_path = ?`,
		projectPath, filePath,
	).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		d.log.Error(fmt.Sprintf("GetFileAccessCount: %v", err))
		return 0, fmt.Errorf("get file access count: %w", err)
	}
	return count, nil
}

// GetFileAccess returns the access record for a specific file, or nil if not found.
func (d *DB) GetFileAccess(projectPath, filePath string) (*FileAccess, error) {
	row := d.db.QueryRow(`SELECT id, project_path, file_path, read_count, edit_count, last_seen
		FROM file_access WHERE project_path = ? AND file_path = ?`, projectPath, filePath)
	var f FileAccess
	if err := row.Scan(&f.ID, &f.ProjectPath, &f.FilePath, &f.ReadCount, &f.EditCount, &f.LastSeen); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get file access: %w", err)
	}
	return &f, nil
}

// ---------------------------------------------------------------------------
// Imported sessions
// ---------------------------------------------------------------------------

// IsSessionImported reports whether the given file path has already been imported.
func (d *DB) IsSessionImported(filePath string) (bool, error) {
	var count int
	err := d.db.QueryRow(
		`SELECT COUNT(*) FROM imported_sessions WHERE session_file = ?`, filePath,
	).Scan(&count)
	if err != nil {
		d.log.Error(fmt.Sprintf("IsSessionImported: %v", err))
		return false, fmt.Errorf("is session imported: %w", err)
	}

	imported := count > 0
	d.log.Info(fmt.Sprintf("IsSessionImported: file=%s imported=%v", filePath, imported))
	return imported, nil
}

// MarkSessionImported records that the given file has been imported.
func (d *DB) MarkSessionImported(filePath, projectPath string, entriesCreated int) error {
	query := `INSERT OR IGNORE INTO imported_sessions (session_file, project_path, entries_created)
		VALUES (?, ?, ?)`

	_, err := d.db.Exec(query, filePath, projectPath, entriesCreated)
	if err != nil {
		d.log.Error(fmt.Sprintf("MarkSessionImported file=%s: %v", filePath, err))
		return fmt.Errorf("mark session imported: %w", err)
	}

	d.log.Info(fmt.Sprintf("MarkSessionImported: file=%s project=%s entries=%d", filePath, projectPath, entriesCreated))
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// coalesce returns s if non-empty, otherwise returns fallback.
func coalesce(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
