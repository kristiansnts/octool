package arms

import (
	"strings"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

// RunConventionArm is Arm 4: detects repeated "still not working" patterns and
// suggests saving a convention entry after 3 occurrences.
func RunConventionArm(db *storage.DB, tr *tracker.Tracker, cwd, text string) Result {
	lower := strings.ToLower(text)
	isStill := strings.Contains(lower, "still") ||
		strings.Contains(lower, "not working") ||
		strings.Contains(lower, "didn't work") ||
		strings.Contains(lower, "does not work")

	if !isStill {
		return Result{}
	}

	tr.IncrStill()
	log := logger.New("arm:convention")
	s := tr.CurrentSession()
	if s == nil || s.StillCount < 3 {
		return Result{}
	}

	// Check if a convention entry already exists for this project.
	entries, err := db.GetContextEntries(cwd, "convention")
	if err == nil && len(entries) > 0 {
		return Result{}
	}

	log.Info("suggested convention after 3 still-followups")
	return Result{
		SystemMessage: "You've hit the same issue 3+ times. Consider saving a convention entry " +
			"to prevent this in future sessions. Use: octool save --type convention --title '...' --content '...'",
		Action: "suggested",
		Arm:    "convention",
		Detail: "3 still-followups detected",
	}
}
