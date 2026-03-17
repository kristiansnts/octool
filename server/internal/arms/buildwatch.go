package arms

import (
	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/tracker"
)

// RunBuildWatchArm is Arm 2: detects repeated build cycles and coaches batching.
// Fires exactly once per session when BuildCycles reaches 3.
func RunBuildWatchArm(tr *tracker.Tracker, cwd string) Result {
	log := logger.New("arm:buildwatch")
	s := tr.CurrentSession()
	if s == nil {
		return Result{}
	}
	if s.BuildCycles >= 3 && !s.CoachSent {
		s.CoachSent = true
		log.Warn("3 build cycles detected. coaching message injected.")
		return Result{
			SystemMessage: "Tip: You have had 3+ build cycles (edit → build fail → repeat). " +
				"Consider batching all edits before running build/test to reduce redundant compilation cycles.",
			Action: "suggested",
			Arm:    "buildwatch",
			Detail: "3 build cycles detected",
		}
	}
	return Result{}
}
