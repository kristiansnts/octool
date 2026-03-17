package arms

import (
	"fmt"
	"strings"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

// RunFilemapArm is Arm 1: auto-creates file-map context entries for frequently
// accessed files (≥3 total reads across all sessions). Runs at session end.
func RunFilemapArm(db *storage.DB, tr *tracker.Tracker, cwd string) Result {
	log := logger.New("arm:filemap")

	// Use DB counts (cumulative across sessions) — threshold: 3+ reads
	hotFiles, err := db.GetHotFiles(cwd, 3, 20)
	if err != nil {
		log.Error(fmt.Sprintf("GetHotFiles: %v", err))
		return Result{}
	}

	// Also include in-session files not yet in DB (fallback for current session)
	s := tr.CurrentSession()
	if s != nil {
		for path, count := range s.FileReads {
			if count < 3 {
				continue
			}
			found := false
			for _, h := range hotFiles {
				if h.FilePath == path {
					found = true
					break
				}
			}
			if !found {
				hotFiles = append(hotFiles, storage.FileAccess{
					ProjectPath: cwd,
					FilePath:    path,
					ReadCount:   count,
				})
			}
		}
	}

	created := 0
	for _, f := range hotFiles {
		// Skip if file-map entry already exists for this path
		entries, err := db.SearchContextEntries(f.FilePath)
		if err == nil {
			found := false
			for _, e := range entries {
				if e.Type == "file-map" && strings.Contains(e.Title, f.FilePath) {
					found = true
					break
				}
			}
			if found {
				continue
			}
		}

		total := f.ReadCount + f.EditCount
		entry := storage.ContextEntry{
			ProjectPath:   cwd,
			Type:          "file-map",
			Title:         fmt.Sprintf("File map: %s", f.FilePath),
			Content:       fmt.Sprintf("Frequently accessed file: %s (read %d×, edited %d×, total %d accesses)", f.FilePath, f.ReadCount, f.EditCount, total),
			Source:        "octool-auto",
			StalenessRisk: "medium",
			Priority:      "normal",
		}
		if _, saveErr := db.SaveContextEntry(entry); saveErr != nil {
			log.Error(fmt.Sprintf("failed to save file-map for %s: %v", f.FilePath, saveErr))
			continue
		}
		log.Info(fmt.Sprintf("auto_saved file-map for %s (%d accesses)", f.FilePath, total))
		created++
	}

	if created == 0 {
		return Result{}
	}
	return Result{
		Action: "auto_saved",
		Arm:    "filemap",
		Detail: fmt.Sprintf("created %d file-map entries", created),
	}
}
