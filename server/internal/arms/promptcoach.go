package arms

import (
	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/scorer"
	"github.com/kristiansnts/octool/internal/tracker"
)

// RunPromptCoachArm is Arm 5: scores the user's prompt and injects a tip when quality is LOW.
func RunPromptCoachArm(tr *tracker.Tracker, cwd, text string) Result {
	log := logger.New("arm:promptcoach")
	sc := scorer.ScorePrompt(text)
	tr.RecordPrompt(sc.Label)

	if sc.Quality == scorer.QualityLow && sc.Suggestion != "" {
		log.Info("quality=LOW suggestion=\"" + sc.Suggestion + "\"")
		return Result{
			SystemMessage: "Prompt tip: " + sc.Suggestion,
			Action:        "suggested",
			Arm:           "promptcoach",
			Detail:        "quality=LOW",
		}
	}
	return Result{}
}
