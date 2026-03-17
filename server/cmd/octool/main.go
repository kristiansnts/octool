package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kristiansnts/octool/internal/arms"
	"github.com/kristiansnts/octool/internal/dashboard"
	"github.com/kristiansnts/octool/internal/session"
	"github.com/kristiansnts/octool/internal/storage"
	"github.com/kristiansnts/octool/internal/tracker"
	"github.com/spf13/cobra"
)

func printJSON(v any) {
	b, _ := json.Marshal(v)
	fmt.Println(string(b))
}

func main() {
	root := &cobra.Command{
		Use:   "octool",
		Short: "Octool — automated token efficiency layer for Copilot CLI",
	}

	// inject --cwd PATH --source new|resume
	var injectCwd, injectSource string
	injectCmd := &cobra.Command{
		Use:   "inject",
		Short: "Inject context into session",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			tr := tracker.New()
			tr.StartSession(fmt.Sprintf("session-%d", time.Now().UnixMilli()), injectCwd, injectSource)
			mgr := arms.NewManager(db, tr)
			results := mgr.OnSessionStart(injectCwd, injectSource)
			msg := arms.CombineMessages(results)
			printJSON(map[string]string{"systemMessage": msg})
			return nil
		},
	}
	injectCmd.Flags().StringVar(&injectCwd, "cwd", "", "Working directory path")
	injectCmd.Flags().StringVar(&injectSource, "source", "new", "Session source: new|resume")

	// track --cwd PATH --tool NAME --args JSON --result TYPE
	var trackCwd, trackTool, trackArgs, trackResult string
	trackCmd := &cobra.Command{
		Use:   "track",
		Short: "Track a tool use event",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			tr := tracker.New()
			tr.StartSession(fmt.Sprintf("session-%d", time.Now().UnixMilli()), trackCwd, "new")
			tr.RecordToolCall(tracker.ToolCall{Tool: trackTool, Args: trackArgs, Result: trackResult})

			// Persist file access to DB so counts accumulate across sessions
			tool := strings.ToLower(trackTool)
			if tool == "view" || tool == "edit" || tool == "write" || tool == "create" {
				if filePath := extractFilePath(trackArgs); filePath != "" && trackCwd != "" {
					_ = db.UpsertFileAccess(trackCwd, filePath, tool != "view")
				}
			}

			mgr := arms.NewManager(db, tr)
			results := mgr.OnPostToolUse(trackCwd, trackTool, trackArgs, trackResult)
			msg := arms.CombineMessages(results)
			printJSON(map[string]string{"systemMessage": msg})
			return nil
		},
	}
	trackCmd.Flags().StringVar(&trackCwd, "cwd", "", "Working directory path")
	trackCmd.Flags().StringVar(&trackTool, "tool", "", "Tool name")
	trackCmd.Flags().StringVar(&trackArgs, "args", "{}", "Tool arguments as JSON")
	trackCmd.Flags().StringVar(&trackResult, "result", "", "Result type")

	// pre-check --cwd PATH --tool NAME --args JSON
	var preCheckCwd, preCheckTool, preCheckArgs string
	preCheckCmd := &cobra.Command{
		Use:   "pre-check",
		Short: "Pre-tool-use check",
		RunE: func(cmd *cobra.Command, args []string) error {
			printJSON(map[string]string{"systemMessage": ""})
			return nil
		},
	}
	preCheckCmd.Flags().StringVar(&preCheckCwd, "cwd", "", "Working directory path")
	preCheckCmd.Flags().StringVar(&preCheckTool, "tool", "", "Tool name")
	preCheckCmd.Flags().StringVar(&preCheckArgs, "args", "{}", "Tool arguments as JSON")

	// prompt-check --cwd PATH --text STRING
	var promptCheckCwd, promptCheckText string
	promptCheckCmd := &cobra.Command{
		Use:   "prompt-check",
		Short: "Analyze user prompt for token efficiency",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			tr := tracker.New()
			tr.StartSession(fmt.Sprintf("session-%d", time.Now().UnixMilli()), promptCheckCwd, "new")
			mgr := arms.NewManager(db, tr)
			results := mgr.OnUserPrompt(promptCheckCwd, promptCheckText)
			msg := arms.CombineMessages(results)
			printJSON(map[string]string{"systemMessage": msg})
			return nil
		},
	}
	promptCheckCmd.Flags().StringVar(&promptCheckCwd, "cwd", "", "Working directory path")
	promptCheckCmd.Flags().StringVar(&promptCheckText, "text", "", "Prompt text to analyze")

	// finalize --cwd PATH
	var finalizeCwd string
	finalizeCmd := &cobra.Command{
		Use:   "finalize",
		Short: "Finalize and summarize a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			tr := tracker.New()
			tr.StartSession(fmt.Sprintf("session-%d", time.Now().UnixMilli()), finalizeCwd, "new")
			mgr := arms.NewManager(db, tr)
			mgr.OnSessionEnd(finalizeCwd)
			printJSON(map[string]bool{"ok": true})
			return nil
		},
	}
	finalizeCmd.Flags().StringVar(&finalizeCwd, "cwd", "", "Working directory path")

	// track-error --cwd PATH --name TYPE --message TEXT
	var trackErrorCwd, trackErrorName, trackErrorMessage string
	trackErrorCmd := &cobra.Command{
		Use:   "track-error",
		Short: "Track an error event",
		RunE: func(cmd *cobra.Command, args []string) error {
			printJSON(map[string]bool{"ok": true})
			return nil
		},
	}
	trackErrorCmd.Flags().StringVar(&trackErrorCwd, "cwd", "", "Working directory path")
	trackErrorCmd.Flags().StringVar(&trackErrorName, "name", "", "Error type name")
	trackErrorCmd.Flags().StringVar(&trackErrorMessage, "message", "", "Error message text")

	// status
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show current session token efficiency status",
		RunE: func(cmd *cobra.Command, args []string) error {
			printJSON(map[string]any{"views": 0, "edits": 0, "ratio": 0.0})
			return nil
		},
	}

	// fetch-session --limit N --project PATH --all --dry-run
	var fetchLimit int
	var fetchProject string
	var fetchAll, fetchDryRun bool
	fetchSessionCmd := &cobra.Command{
		Use:   "fetch-session",
		Short: "Fetch session history",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			p := session.New(db)
			scanned, created, skipped, err := p.Run(session.Options{
				Limit:       fetchLimit,
				ProjectPath: fetchProject,
				All:         fetchAll,
				DryRun:      fetchDryRun,
			})
			if err != nil {
				return err
			}
			printJSON(map[string]any{
				"ok":      true,
				"scanned": scanned,
				"created": created,
				"skipped": skipped,
				"dryRun":  fetchDryRun,
			})
			return nil
		},
	}
	fetchSessionCmd.Flags().IntVar(&fetchLimit, "limit", 10, "Number of sessions to fetch")
	fetchSessionCmd.Flags().StringVar(&fetchProject, "project", "", "Project path filter")
	fetchSessionCmd.Flags().BoolVar(&fetchAll, "all", false, "Fetch all sessions")
	fetchSessionCmd.Flags().BoolVar(&fetchDryRun, "dry-run", false, "Dry run without saving")

	// entries --project PATH --type TYPE
	var entriesProject, entriesType string
	entriesCmd := &cobra.Command{
		Use:   "entries",
		Short: "List stored context entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			entries, err := db.GetContextEntries(entriesProject, entriesType)
			if err != nil {
				return err
			}
			printJSON(entries)
			return nil
		},
	}
	entriesCmd.Flags().StringVar(&entriesProject, "project", "", "Project path filter")
	entriesCmd.Flags().StringVar(&entriesType, "type", "", "Entry type filter")

	// save --type TYPE --title TITLE --content CONTENT --project PATH
	var saveType, saveTitle, saveContent, saveProject string
	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Save a context entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			id, err := db.SaveContextEntry(storage.ContextEntry{
				ProjectPath: saveProject,
				Type:        saveType,
				Title:       saveTitle,
				Content:     saveContent,
				Source:      "manual",
			})
			if err != nil {
				return err
			}
			printJSON(map[string]any{"ok": true, "id": id})
			return nil
		},
	}
	saveCmd.Flags().StringVar(&saveType, "type", "", "Entry type")
	saveCmd.Flags().StringVar(&saveTitle, "title", "", "Entry title")
	saveCmd.Flags().StringVar(&saveContent, "content", "", "Entry content")
	saveCmd.Flags().StringVar(&saveProject, "project", "", "Project path")

	// delete --id ID
	var deleteID string
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a context entry by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			id, err := strconv.ParseInt(deleteID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q: %w", deleteID, err)
			}
			if err := db.DeleteContextEntry(id); err != nil {
				return err
			}
			printJSON(map[string]bool{"ok": true})
			return nil
		},
	}
	deleteCmd.Flags().StringVar(&deleteID, "id", "", "Entry ID to delete")

	// serve --port INT --background
	var servePort int
	var serveBackground bool
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the octool HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serveBackground {
				// Re-exec self without --background so the child runs the server
				self, err := os.Executable()
				if err != nil {
					return err
				}
				child := exec.Command(self, "serve", "--port", strconv.Itoa(servePort))
				child.Stdout = nil
				child.Stderr = nil
				// Detach child into its own session so it survives when the hook exits
				child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
				if err := child.Start(); err != nil {
					return err
				}
				printJSON(map[string]any{"ok": true, "pid": child.Process.Pid, "port": servePort})
				return nil
			}
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close()
			srv := dashboard.New(db, servePort)
			fmt.Fprintf(os.Stderr, "OcTool dashboard running at http://localhost:%d\n", servePort)
			return srv.Start()
		},
	}
	serveCmd.Flags().IntVar(&servePort, "port", 37888, "Port to listen on")
	serveCmd.Flags().BoolVar(&serveBackground, "background", false, "Run server in background (daemonize)")

	// setup --cwd PATH  (creates .github/hooks/octool.json with absolute script paths)
	var setupCwd string
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Install octool hooks into the current project's .github/hooks/",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd := setupCwd
			if cwd == "" {
				var err error
				cwd, err = os.Getwd()
				if err != nil {
					return err
				}
			}

			// Resolve the installed plugin scripts directory
			self, err := os.Executable()
			if err != nil {
				return err
			}
			scriptDir := filepath.Join(filepath.Dir(self), "..", "scripts")
			scriptDir, err = filepath.Abs(scriptDir)
			if err != nil {
				return err
			}

			// Walk up to find git root, fall back to cwd
			gitRoot := cwd
			for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
				if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
					gitRoot = dir
					break
				}
			}

			hooksDir := filepath.Join(gitRoot, ".github", "hooks")
			if err := os.MkdirAll(hooksDir, 0o755); err != nil {
				return fmt.Errorf("create hooks dir: %w", err)
			}

			hookJSON := fmt.Sprintf(`{
  "version": 1,
  "hooks": {
    "sessionStart": [{"type":"command","bash":"%s/session-start.sh","cwd":".","timeoutSec":10}],
    "sessionEnd":   [{"type":"command","bash":"%s/session-end.sh","cwd":".","timeoutSec":15}],
    "userPromptSubmitted": [{"type":"command","bash":"%s/user-prompt.sh","cwd":".","timeoutSec":5}],
    "preToolUse":   [{"type":"command","bash":"%s/pre-tool-use.sh","cwd":".","timeoutSec":3}],
    "postToolUse":  [{"type":"command","bash":"%s/post-tool-use.sh","cwd":".","timeoutSec":3}],
    "errorOccurred":[{"type":"command","bash":"%s/error-occurred.sh","cwd":".","timeoutSec":3}]
  }
}`, scriptDir, scriptDir, scriptDir, scriptDir, scriptDir, scriptDir)

			outPath := filepath.Join(hooksDir, "octool.json")
			if err := os.WriteFile(outPath, []byte(hookJSON), 0o644); err != nil {
				return fmt.Errorf("write hooks file: %w", err)
			}

			printJSON(map[string]any{"ok": true, "path": outPath, "scripts": scriptDir})
			return nil
		},
	}
	setupCmd.Flags().StringVar(&setupCwd, "cwd", "", "Project directory (default: current dir)")

	// version
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print octool version",
		RunE: func(cmd *cobra.Command, args []string) error {
			printJSON(map[string]string{"version": "0.1.0"})
			return nil
		},
	}

	root.AddCommand(
		injectCmd,
		trackCmd,
		preCheckCmd,
		promptCheckCmd,
		finalizeCmd,
		trackErrorCmd,
		statusCmd,
		fetchSessionCmd,
		entriesCmd,
		saveCmd,
		deleteCmd,
		serveCmd,
		setupCmd,
		versionCmd,
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// extractFilePath pulls the first file path from a tool args string.
func extractFilePath(args string) string {
	for _, f := range strings.Fields(args) {
		if strings.Contains(f, "/") || (strings.Contains(f, ".") && !strings.HasPrefix(f, "-")) {
			return f
		}
	}
	return ""
}
