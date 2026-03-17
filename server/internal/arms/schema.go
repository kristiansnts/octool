package arms

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

// isTypeFile returns true if the path looks like a type-definition or schema file.
func isTypeFile(path string) bool {
	lower := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(base, "_service.go"),
		strings.HasSuffix(base, "types.go"),
		strings.HasSuffix(base, "types.ts"),
		strings.HasSuffix(base, ".d.ts"),
		strings.HasSuffix(base, "_model.go"),
		strings.Contains(base, "schema"):
		return true
	case strings.Contains(lower, "/models/"):
		return true
	}
	return false
}

// RunSchemaArm is Arm 6: when a type-definition file is read >3 times and no
// schema entry exists, suggests saving it as a schema entry.
func RunSchemaArm(db *storage.DB, tr *tracker.Tracker, cwd, toolName, toolArgs string) Result {
	if !strings.EqualFold(toolName, "view") {
		return Result{}
	}

	path := extractFilePath(toolArgs)
	if path == "" || !isTypeFile(path) {
		return Result{}
	}

	s := tr.CurrentSession()
	if s == nil {
		return Result{}
	}
	count := s.FileReads[path]
	if count <= 3 {
		return Result{}
	}

	// Check if a schema entry already exists.
	entries, err := db.SearchContextEntries(path)
	if err == nil {
		for _, e := range entries {
			if e.Type == "schema" && strings.Contains(e.Title, path) {
				return Result{}
			}
		}
	}

	log := logger.New("arm:schema")
	log.Info(fmt.Sprintf("%s read %d times. suggested schema entry.", path, count))
	return Result{
		SystemMessage: fmt.Sprintf("'%s' has been read %d times. Consider saving it as a schema entry for auto-injection.", path, count),
		Action:        "suggested",
		Arm:           "schema",
		Detail:        fmt.Sprintf("path=%s reads=%d", path, count),
	}
}

// extractFilePath pulls the first path-like token from a toolArgs string.
func extractFilePath(args string) string {
	for f := range strings.FieldsSeq(args) {
		if strings.Contains(f, "/") || (strings.Contains(f, ".") && !strings.HasPrefix(f, "-")) {
			return f
		}
	}
	return ""
}
