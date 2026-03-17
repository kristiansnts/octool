package arms

import (
	"fmt"
	"strings"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

// RunViewEditArm is Arm 8: monitors the view:edit ratio and injects file-map
// entries when it exceeds 0.7. Uses cumulative DB counts across all sessions,
// with a minimum of 5 total accesses to avoid noise.
func RunViewEditArm(db *storage.DB, tr *tracker.Tracker, cwd string) Result {
	// Get cumulative file access counts from DB
	hotFiles, err := db.GetHotFiles(cwd, 1, 200)
	if err != nil || len(hotFiles) == 0 {
		// Fall back to in-session tracker
		s := tr.CurrentSession()
		if s == nil || s.TotalTools < 5 {
			return Result{}
		}
		ratio := tr.ViewEditRatio()
		if ratio <= 0.7 {
			return Result{}
		}
		return injectFileMaps(db, cwd, ratio)
	}

	// Calculate cumulative ratio from DB
	var totalReads, totalEdits int
	for _, f := range hotFiles {
		totalReads += f.ReadCount
		totalEdits += f.EditCount
	}
	total := totalReads + totalEdits
	if total < 5 {
		return Result{}
	}

	ratio := float64(totalReads) / float64(totalEdits+1)
	if ratio <= 0.7 {
		return Result{}
	}

	return injectFileMaps(db, cwd, ratio)
}

func injectFileMaps(db *storage.DB, cwd string, ratio float64) Result {
	log := logger.New("arm:viewedit")
	entries, err := db.GetContextEntries(cwd, "file-map")
	if err != nil || len(entries) == 0 {
		log.Warn(fmt.Sprintf("ratio=%.2f but no file-map entries to inject", ratio))
		return Result{}
	}

	if len(entries) > 3 {
		entries = entries[:3]
	}

	var lines []string
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("- %s", e.Content))
	}

	log.Warn(fmt.Sprintf("ratio=%.2f injecting %d file-maps", ratio, len(entries)))
	return Result{
		SystemMessage: fmt.Sprintf(
			"High view:edit ratio (%.2f). Injecting file maps:\n%s",
			ratio, strings.Join(lines, "\n"),
		),
		Action: "injected",
		Arm:    "viewedit",
		Detail: fmt.Sprintf("ratio=%.2f", ratio),
	}
}
