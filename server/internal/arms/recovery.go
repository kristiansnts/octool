package arms

import (
	"fmt"
	"strings"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

// RunRecoveryArm is Arm 3: on resume, pre-loads top file-map entries as context.
func RunRecoveryArm(db *storage.DB, tr *tracker.Tracker, cwd, source string) Result {
	if source != "resume" {
		return Result{}
	}
	log := logger.New("arm:recovery")

	entries, err := db.GetContextEntries(cwd, "file-map")
	if err != nil || len(entries) == 0 {
		return Result{}
	}

	// Cap at 5.
	if len(entries) > 5 {
		entries = entries[:5]
	}

	var lines []string
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("- %s", e.Content))
	}
	msg := "Resuming session. Relevant file maps:\n" + strings.Join(lines, "\n")

	log.Info(fmt.Sprintf("resume detected. injecting %d file-maps from previous session.", len(entries)))
	return Result{
		SystemMessage: msg,
		Action:        "injected",
		Arm:           "recovery",
		Detail:        fmt.Sprintf("injected %d file-maps", len(entries)),
	}
}
