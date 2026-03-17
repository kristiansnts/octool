// Package logger provides a file-based structured logger for octool.
// Logs are written to ~/.octool/logs/ with daily rotation.
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger writes structured log entries to ~/.octool/logs/.
type Logger struct {
	source string
	mu     sync.Mutex
	logDir string
}

// New creates a Logger tagged with the given source name.
// It ensures the log directory (~/.octool/logs/) exists.
func New(source string) *Logger {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	logDir := filepath.Join(home, ".octool", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		// best effort — if we cannot create the dir, we'll fail silently on writes
		_ = err
	}
	return &Logger{source: source, logDir: logDir}
}

// dateSuffix returns the current UTC date in YYYY-MM-DD format.
func dateSuffix() string {
	return time.Now().UTC().Format("2006-01-02")
}

// timestamp returns the current UTC time in RFC3339 format.
func timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// appendLine opens the named file (appending) and writes a single line.
func (l *Logger) appendLine(filename, line string) {
	path := filepath.Join(l.logDir, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintln(f, line)
}

// Info writes an INFO-level message to the daily main log.
func (l *Logger) Info(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("[%s] [INFO] [%s] %s", timestamp(), l.source, msg)
	l.appendLine("octool-"+dateSuffix()+".log", line)
}

// Warn writes a WARN-level message to the daily main log.
func (l *Logger) Warn(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("[%s] [WARN] [%s] %s", timestamp(), l.source, msg)
	l.appendLine("octool-"+dateSuffix()+".log", line)
}

// Error writes an ERROR-level message to both the daily main log and the
// daily error log.
func (l *Logger) Error(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("[%s] [ERROR] [%s] %s", timestamp(), l.source, msg)
	l.appendLine("octool-"+dateSuffix()+".log", line)
	l.appendLine("errors-"+dateSuffix()+".log", line)
}

// LogArm writes an ARM activity entry to the daily arm-activity log.
func (l *Logger) LogArm(arm, action, detail string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("[%s] [ARM] [%s] arm=%s action=%s detail=%s",
		timestamp(), l.source, arm, action, detail)
	l.appendLine("arm-activity-"+dateSuffix()+".log", line)
}
