// Package arms coordinates all 8 autonomous OcTool arms.
package arms

import (
	"strings"

	"github.com/kristiansnts/octool/internal/logger"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
)

// Result is what an arm returns — a systemMessage to inject and metadata.
type Result struct {
	SystemMessage string
	Action        string // "injected" | "suggested" | "warned" | "auto_saved" | ""
	Arm           string
	Detail        string
}

// Manager coordinates all 8 arms.
type Manager struct {
	db      *storage.DB
	tracker *tracker.Tracker
	log     *logger.Logger
}

// NewManager creates a Manager wired to the given DB and Tracker.
func NewManager(db *storage.DB, tr *tracker.Tracker) *Manager {
	return &Manager{db: db, tracker: tr, log: logger.New("arms")}
}

// logResults writes an arm-activity entry for each fired result to both the
// file log and the DB so the dashboard can display it.
func (m *Manager) logResults(hook, cwd string, results []Result) {
	for _, r := range results {
		if r.Arm == "" {
			continue
		}
		detail := r.Detail
		if detail == "" {
			detail = hook
		}
		m.log.LogArm(r.Arm, r.Action, detail)
		_ = m.db.LogArmActivity(storage.ArmActivity{
			ProjectPath: cwd,
			Arm:         r.Arm,
			Action:      r.Action,
			Detail:      detail,
		})
	}
}

// OnSessionStart fires Arms 3 (recovery) and 7 (resume advisor).
func (m *Manager) OnSessionStart(cwd, source string) []Result {
	results := nonEmpty([]Result{
		RunRecoveryArm(m.db, m.tracker, cwd, source),
		RunResumeArm(m.db, m.tracker, cwd, source),
	})
	m.logResults("sessionStart", cwd, results)
	return results
}

// OnPostToolUse fires Arms 2 (buildwatch), 6 (schema), 8 (view:edit).
// Arm 1 (filemap) runs at session end, not post-tool-use.
func (m *Manager) OnPostToolUse(cwd, toolName, toolArgs, resultType string) []Result {
	results := nonEmpty([]Result{
		RunBuildWatchArm(m.tracker, cwd),
		RunSchemaArm(m.db, m.tracker, cwd, toolName, toolArgs),
		RunViewEditArm(m.db, m.tracker, cwd),
	})
	m.logResults("postToolUse", cwd, results)
	return results
}

// OnUserPrompt fires Arms 4 (convention) and 5 (prompt coach).
func (m *Manager) OnUserPrompt(cwd, text string) []Result {
	results := nonEmpty([]Result{
		RunConventionArm(m.db, m.tracker, cwd, text),
		RunPromptCoachArm(m.tracker, cwd, text),
	})
	m.logResults("userPromptSubmitted", cwd, results)
	return results
}

// OnSessionEnd fires Arm 1 (filemap auto-generator).
func (m *Manager) OnSessionEnd(cwd string) []Result {
	results := nonEmpty([]Result{
		RunFilemapArm(m.db, m.tracker, cwd),
	})
	m.logResults("sessionEnd", cwd, results)
	return results
}

// CombineMessages merges multiple Result.SystemMessage into one string.
func CombineMessages(results []Result) string {
	var parts []string
	for _, r := range results {
		if r.SystemMessage != "" {
			parts = append(parts, r.SystemMessage)
		}
	}
	return strings.Join(parts, "\n\n")
}

func nonEmpty(results []Result) []Result {
	var out []Result
	for _, r := range results {
		if r.Action != "" || r.SystemMessage != "" {
			out = append(out, r)
		}
	}
	return out
}
