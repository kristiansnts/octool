package arms

import (
	"fmt"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

// RunResumeArm is Arm 7: advises using `copilot resume` when the last session
// was heavy, or reminds the user to be specific on a resumed session.
func RunResumeArm(db *storage.DB, tr *tracker.Tracker, cwd, source string) Result {
	log := logger.New("arm:resume")

	if source == "resume" {
		s := tr.CurrentSession()
		if s != nil && s.PromptCount == 0 {
			log.Info("resumed session — reminding user to be specific")
			return Result{
				SystemMessage: "You resumed a session. Be specific in your first prompt to avoid re-reading files unnecessarily.",
				Action:        "suggested",
				Arm:           "resume",
				Detail:        "resume with no prompts yet",
			}
		}
		return Result{}
	}

	// New session: check if last session was heavy.
	recent, err := db.GetRecentSessionMetrics(cwd, 1)
	if err != nil || len(recent) == 0 {
		return Result{}
	}
	last := recent[0]
	if last.TotalTools > 20 {
		log.Info(fmt.Sprintf("last session had %d tools. suggested resume.", last.TotalTools))
		return Result{
			SystemMessage: fmt.Sprintf(
				"Your last session had %d tool calls. If you're continuing that work, consider using 'copilot resume' instead of starting fresh.",
				last.TotalTools,
			),
			Action: "suggested",
			Arm:    "resume",
			Detail: fmt.Sprintf("last session tools=%d", last.TotalTools),
		}
	}
	return Result{}
}
