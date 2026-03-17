# OcTool — Build Plan (GitHub Copilot CLI Plugin)

## Target Platform

**GitHub Copilot CLI ONLY.** NOT for Claude Code (different hook system entirely).

Copilot CLI hooks: `.github/hooks/*.json` with `sessionStart`, `sessionEnd`, `userPromptSubmitted`, `preToolUse`, `postToolUse`, `errorOccurred` — receive JSON via stdin, return JSON via stdout.

---

## 6 Non-Negotiable Requirements

1. **GLOBAL context** — All data stored in `~/.octool/` (not per-project). One SQLite DB at `~/.octool/octool.db`. Context entries (file maps, conventions, schemas) are global and available across ALL projects. The `project_path` column exists for filtering but the DB is always global.

2. **Always autonomously active** — All 8 arms run automatically via hooks. No skill invocation needed. The `sessionStart` hook auto-injects relevant context. The `postToolUse` hook auto-tracks every tool call. The `userPromptSubmitted` hook auto-coaches prompts. The `sessionEnd` hook auto-saves metrics. User never needs to type `/octool` or any command for the system to work.

3. **`/fetch-session` skill** — A skill that parses `~/.copilot/session-state/` folder to extract decisions, patterns, file maps, and conventions from past Copilot CLI sessions. This is an ON-DEMAND skill (user invokes it), unlike the autonomous hooks.

4. **Comprehensive logging** — Every hook execution, every arm decision, every error is logged to `~/.octool/logs/`. Log files are date-rotated. If a hook script fails, the error is captured to the log (never silently swallowed). A `scripts/log.sh` utility is sourced by all hook scripts for consistent logging.

5. **Install via marketplace** — The repo structure supports `copilot plugin marketplace add kristiansnts/octool` then `copilot plugin install octool`. This requires a `marketplace.json` at `.github/plugin/marketplace.json` AND a plugin directory with `plugin.json`.

6. **Auto-run on every `copilot` session** — Once installed, plugin hooks are automatically loaded by Copilot CLI. Installed plugins live at `~/.copilot/state/installed-plugins/`. The hooks fire on every session start, every prompt, every tool call — zero manual activation.

---

## Research Foundation

15-day session: 210 messages, 2,832 tool calls, 13,063 events.

| Rank | Waste Source | % Budget | Fix |
|------|-------------|----------|-----|
| 1 | Repeated file reads | 25% | Auto file-map entries |
| 2 | Build/check cycles | 10% | Batch edit coaching |
| 3 | Context recovery | 8% | Resume pre-loading |
| 4 | Convention violations | 7% | Auto convention guard |
| 5 | Vague prompts | 5% | Prompt coaching |

Total preventable: ~77,000 tokens (33% of budget). Key metric: view:edit ratio > 0.7 = AI lacks context.

---

## Repository Structure

```
octool/                                  # GitHub repo: kristiansnts/octool
├── .github/
│   └── plugin/
│       └── marketplace.json             # Marketplace manifest (for discovery)
├── plugins/
│   └── octool/                          # The actual plugin directory
│       ├── plugin.json                  # Plugin manifest
│       ├── hooks.json                   # 6 lifecycle hooks
│       ├── scripts/                     # Bash hook handlers
│       │   ├── _lib.sh                  # Shared logging + binary resolver
│       │   ├── session-start.sh
│       │   ├── session-end.sh
│       │   ├── user-prompt.sh
│       │   ├── post-tool-use.sh
│       │   ├── pre-tool-use.sh
│       │   └── error-occurred.sh
│       ├── skills/
│       │   ├── octool-status/
│       │   │   └── SKILL.md             # /octool-status — show efficiency metrics
│       │   └── fetch-session/
│       │       └── SKILL.md             # /fetch-session — parse ~/.copilot/session-state
│       ├── agents/
│       │   └── octool.agent.md          # Optional agent personality
│       └── bin/                         # Pre-compiled Go binaries
│           ├── octool-darwin-arm64
│           ├── octool-darwin-amd64
│           ├── octool-linux-amd64
│           ├── octool-linux-arm64
│           └── octool-windows-amd64.exe
├── server/                              # Go source code
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   └── octool/
│   │       └── main.go                  # CLI entry point (cobra subcommands)
│   ├── internal/
│   │   ├── tracker/
│   │   │   └── tracker.go              # In-memory session state tracker
│   │   ├── scorer/
│   │   │   └── scorer.go               # Prompt quality scorer
│   │   ├── metrics/
│   │   │   └── metrics.go              # Waste computation + DB write
│   │   ├── storage/
│   │   │   ├── storage.go              # SQLite init, migrations, CRUD
│   │   │   └── storage_test.go
│   │   ├── session/
│   │   │   └── parser.go               # Parse ~/.copilot/session-state/ files
│   │   ├── logger/
│   │   │   └── logger.go               # File logger to ~/.octool/logs/
│   │   ├── arms/
│   │   │   ├── manager.go              # Coordinates all 8 arms
│   │   │   ├── filemap.go              # Arm 1: auto file-map generator
│   │   │   ├── buildwatch.go           # Arm 2: build cycle detector
│   │   │   ├── recovery.go             # Arm 3: context recovery optimizer
│   │   │   ├── convention.go           # Arm 4: convention guard
│   │   │   ├── promptcoach.go          # Arm 5: prompt coach
│   │   │   ├── schema.go              # Arm 6: schema hot file detector
│   │   │   ├── resume.go              # Arm 7: resume advisor
│   │   │   └── viewedit.go            # Arm 8: view:edit ratio monitor
│   │   └── dashboard/
│   │       ├── server.go               # HTTP dashboard handlers
│   │       └── templates/
│   │           └── index.html
│   └── internal_test/                   # Tests
│       ├── tracker_test.go
│       ├── scorer_test.go
│       ├── arms_test.go
│       └── session_parser_test.go
└── README.md
```

---

## Installation Flow

```bash
# Step 1: Add the marketplace
copilot plugin marketplace add kristiansnts/octool

# Step 2: Install the plugin
copilot plugin install octool

# Step 3: Restart Copilot CLI — hooks auto-activate
copilot

# Verify
copilot plugin list
# → octool (kristiansnts/octool) — active
```

After install, the plugin lives at:
`~/.copilot/state/installed-plugins/kristiansnts/octool/`

Hooks fire automatically on every `copilot` session. No manual activation.

---

## marketplace.json

Location: `.github/plugin/marketplace.json`

```json
{
  "name": "kristiansnts-octool",
  "owner": {
    "name": "kristiansnts",
    "email": "kristiansnts@users.noreply.github.com"
  },
  "metadata": {
    "description": "Automated token efficiency layer for Copilot CLI — the octopus brain that reduces wasted tool calls",
    "version": "1.0.0"
  },
  "plugins": [
    {
      "name": "octool",
      "description": "Tracks token waste patterns and auto-injects the right context to reduce redundant file reads, build cycles, and rework",
      "version": "0.1.0",
      "source": "./plugins/octool"
    }
  ]
}
```

---

## plugin.json

Location: `plugins/octool/plugin.json`

```json
{
  "name": "octool",
  "description": "Automated token efficiency layer — tracks waste patterns, auto-injects context, coaches prompts, and reduces redundant tool calls by 33%",
  "version": "0.1.0",
  "author": {
    "name": "kristiansnts"
  },
  "license": "MIT",
  "keywords": ["token-efficiency", "context", "memory", "octopus", "optimization"],
  "hooks": "hooks.json",
  "skills": ["skills/"],
  "agents": "agents/"
}
```

---

## hooks.json

Location: `plugins/octool/hooks.json`

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [
      {
        "type": "command",
        "bash": "./scripts/session-start.sh",
        "cwd": ".",
        "timeoutSec": 10
      }
    ],
    "sessionEnd": [
      {
        "type": "command",
        "bash": "./scripts/session-end.sh",
        "cwd": ".",
        "timeoutSec": 15
      }
    ],
    "userPromptSubmitted": [
      {
        "type": "command",
        "bash": "./scripts/user-prompt.sh",
        "cwd": ".",
        "timeoutSec": 5
      }
    ],
    "preToolUse": [
      {
        "type": "command",
        "bash": "./scripts/pre-tool-use.sh",
        "cwd": ".",
        "timeoutSec": 3
      }
    ],
    "postToolUse": [
      {
        "type": "command",
        "bash": "./scripts/post-tool-use.sh",
        "cwd": ".",
        "timeoutSec": 3
      }
    ],
    "errorOccurred": [
      {
        "type": "command",
        "bash": "./scripts/error-occurred.sh",
        "cwd": ".",
        "timeoutSec": 3
      }
    ]
  }
}
```

---

## Copilot CLI Hook I/O Reference

### Input (received via stdin as JSON)

**sessionStart**:
```json
{"timestamp":1704614400000,"cwd":"/path/to/project","source":"new|resume","initialPrompt":"..."}
```

**userPromptSubmitted**:
```json
{"timestamp":1704614400000,"cwd":"/path/to/project","prompt":"user message text"}
```

**preToolUse**:
```json
{"timestamp":1704614400000,"cwd":"/path/to/project","toolName":"view","toolArgs":"{\"path\":\"src/song/[id].tsx\"}"}
```

**postToolUse**:
```json
{"timestamp":1704614400000,"cwd":"/path/to/project","toolName":"view","toolArgs":"{\"path\":\"src/song/[id].tsx\"}","toolResult":{"resultType":"success"}}
```

**sessionEnd**:
```json
{"timestamp":1704614400000,"cwd":"/path/to/project"}
```

**errorOccurred**:
```json
{"timestamp":1704614400000,"cwd":"/path/to/project","error":{"name":"ErrorType","message":"description"}}
```

### Output (returned via stdout as JSON)

All hooks can return:
```json
{"systemMessage": "text injected into agent context"}
```

`preToolUse` can additionally return:
```json
{"permissionDecision": "allow|deny", "permissionDecisionReason": "why denied"}
```

**Critical**: Non-zero exit codes are logged and skipped — hooks never block the agent. The `systemMessage` is the key mechanism for OcTool to inject context.

---

## Logging System (Requirement #4)

All logs go to `~/.octool/logs/`. Date-rotated daily.

### Log files

```
~/.octool/logs/
├── octool-2026-03-16.log        # Main operational log
├── octool-2026-03-15.log        # Previous day
├── errors-2026-03-16.log        # Errors only (separate file for easy grep)
└── arm-activity-2026-03-16.log  # Arm decisions (what was auto-saved/injected/suggested)
```

### Log format

```
[2026-03-16T14:30:00Z] [INFO] [session-start] project=/Users/krist/weworship source=new injected=3_filemaps,2_conventions
[2026-03-16T14:30:01Z] [INFO] [post-tool-use] tool=view file=src/song/[id].tsx reads_this_session=4 total_reads=41
[2026-03-16T14:30:01Z] [WARN] [arm:viewedit] view_edit_ratio=0.83 threshold=0.70 action=injecting_3_filemaps
[2026-03-16T14:30:02Z] [INFO] [arm:promptcoach] quality=LOW suggestion="mention file path"
[2026-03-16T14:30:05Z] [ERROR] [post-tool-use] failed to parse toolArgs: invalid JSON
[2026-03-16T14:35:00Z] [INFO] [session-end] duration=5m tools=47 views=18 edits=12 ratio=0.67 waste=3200tokens
```

### Shared logging library: `scripts/_lib.sh`

Every hook script sources this file first. It provides:
- `OCTOOL_BIN` — resolves the correct binary for the current platform
- `OCTOOL_HOME` — always `~/.octool`
- `OCTOOL_DB` — always `~/.octool/octool.db`
- `OCTOOL_LOG` — today's log file path
- `octool_log()` — writes a log line with timestamp, level, source
- Error trap — if the script exits non-zero, the error is logged before exit

```bash
#!/bin/bash
# scripts/_lib.sh — sourced by all hook scripts

export OCTOOL_HOME="${HOME}/.octool"
export OCTOOL_DB="${OCTOOL_HOME}/octool.db"
export OCTOOL_LOG_DIR="${OCTOOL_HOME}/logs"
export OCTOOL_LOG="${OCTOOL_LOG_DIR}/octool-$(date +%Y-%m-%d).log"
export OCTOOL_ERROR_LOG="${OCTOOL_LOG_DIR}/errors-$(date +%Y-%m-%d).log"

mkdir -p "$OCTOOL_LOG_DIR"

# Resolve binary
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac
export OCTOOL_BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/bin/octool-${OS}-${ARCH}"

# If binary doesn't exist, try fallback
if [ ! -x "$OCTOOL_BIN" ]; then
  OCTOOL_BIN="$(which octool 2>/dev/null || echo "")"
fi

octool_log() {
  local level="$1" source="$2" msg="$3"
  local ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  echo "[$ts] [$level] [$source] $msg" >> "$OCTOOL_LOG"
  if [ "$level" = "ERROR" ]; then
    echo "[$ts] [$level] [$source] $msg" >> "$OCTOOL_ERROR_LOG"
  fi
}

# Trap errors so they always get logged
trap 'octool_log ERROR "$(basename $0)" "script failed with exit code $?"' ERR
```

---

## Global SQLite Schema

Location: `~/.octool/octool.db` — ONE database for ALL projects.

```sql
-- Context entries (file maps, schemas, conventions) — GLOBAL
CREATE TABLE context_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_path TEXT NOT NULL,         -- which project created this (for filtering)
    type TEXT NOT NULL,                 -- 'file-map' | 'schema' | 'convention' | 'api-catalog'
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    source TEXT DEFAULT 'manual',       -- 'manual' | 'octool-auto' | 'fetch-session'
    staleness_risk TEXT DEFAULT 'medium',
    priority TEXT DEFAULT 'normal',     -- 'normal' | 'always_inject'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Session metrics — GLOBAL (one row per session per project)
CREATE TABLE session_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    project_path TEXT NOT NULL,
    source TEXT DEFAULT 'new',          -- 'new' | 'resume'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    duration_seconds INTEGER DEFAULT 0,

    -- Tool counts
    total_views INTEGER DEFAULT 0,
    total_edits INTEGER DEFAULT 0,
    total_bash INTEGER DEFAULT 0,
    total_grep INTEGER DEFAULT 0,
    total_glob INTEGER DEFAULT 0,
    total_create INTEGER DEFAULT 0,
    total_tools INTEGER DEFAULT 0,
    view_edit_ratio REAL DEFAULT 0.0,

    -- Waste signals
    redundant_reads TEXT DEFAULT '{}',   -- JSON: {"file_path": read_count}
    build_cycles INTEGER DEFAULT 0,
    still_followups INTEGER DEFAULT 0,

    -- Prompt quality
    prompt_count INTEGER DEFAULT 0,
    prompt_low INTEGER DEFAULT 0,
    prompt_medium INTEGER DEFAULT 0,
    prompt_high INTEGER DEFAULT 0,

    -- Computed waste
    estimated_waste_tokens INTEGER DEFAULT 0,
    waste_breakdown TEXT DEFAULT '{}'    -- JSON: {"file_reads":N,"build_cycles":N,...}
);

-- Arm activity log — GLOBAL
CREATE TABLE arm_activity (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT,
    project_path TEXT NOT NULL,
    arm TEXT NOT NULL,
    action TEXT NOT NULL,                -- 'auto_saved' | 'injected' | 'suggested' | 'promoted' | 'warned'
    detail TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Imported sessions (tracks which session-state files have been parsed by /fetch-session)
CREATE TABLE imported_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_file TEXT NOT NULL UNIQUE,   -- path to the session-state file
    project_path TEXT,
    entries_created INTEGER DEFAULT 0,
    imported_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Full-text search on context entries
CREATE VIRTUAL TABLE IF NOT EXISTS context_entries_fts USING fts5(
    title, content, content=context_entries, content_rowid=id
);
```

---

## /fetch-session Skill (Requirement #3)

Location: `plugins/octool/skills/fetch-session/SKILL.md`

```markdown
---
name: fetch-session
description: Parse past Copilot CLI sessions from ~/.copilot/session-state/ and import discovered patterns, conventions, file maps, and decisions into OcTool's global database. Use when the user asks to import session history, learn from past sessions, or bootstrap OcTool with existing knowledge.
---

# Fetch Session

This skill reads Copilot CLI session state files and extracts useful context entries.

## How to use

Run the octool binary to scan and import session history:

```bash
# Import last 5 sessions from all projects
~/.octool/bin/octool fetch-session

# Import last N sessions
~/.octool/bin/octool fetch-session --limit 10

# Import sessions for a specific project
~/.octool/bin/octool fetch-session --project /path/to/project

# Import ALL sessions (full scan)
~/.octool/bin/octool fetch-session --all

# Show what would be imported without saving (dry run)
~/.octool/bin/octool fetch-session --dry-run
```

## What it extracts

The fetch-session command scans `~/.copilot/session-state/` and for each session:

1. **File access patterns** — Files read/edited most often → auto-creates `file-map` entries
2. **Convention hints** — Repeated patterns in edits (e.g., always using a specific import) → suggests `convention` entries
3. **Error patterns** — Recurring errors and their fixes → creates `gotcha` entries
4. **Type definitions** — Files that are type definition files read repeatedly → suggests `schema` entries

## Output

After import, it reports:
- Sessions scanned: N
- New entries created: N (by type)
- Duplicates skipped: N
- Run `/octool-status` to see the updated context

## Notes

- Already-imported sessions are tracked in the DB and won't be re-imported
- Entries created by fetch-session are marked `source: "fetch-session"` for easy identification
- All entries are saved to the GLOBAL database at `~/.octool/octool.db`
```

### How the parser works (`server/internal/session/parser.go`)

The `~/.copilot/session-state/` directory contains session state files (JSON). The parser:

1. Lists all files in `~/.copilot/session-state/`
2. For each file not already in `imported_sessions` table:
   - Parse the JSON to extract: tool calls, file paths, user messages, errors
   - Count file read frequency → if >3 reads → create `file-map` entry
   - Detect edit patterns → if same import/pattern used repeatedly → suggest `convention`
   - Detect error → fix sequences → create `gotcha` entry
3. Deduplicate against existing `context_entries` (search by title similarity)
4. Save new entries with `source: "fetch-session"`
5. Record the file in `imported_sessions`
6. Log everything to `~/.octool/logs/`

---

## /octool-status Skill

Location: `plugins/octool/skills/octool-status/SKILL.md`

```markdown
---
name: octool-status
description: Show current session token efficiency metrics including view:edit ratio, waste breakdown, hot files, and optimization suggestions. Use when the user asks about token usage, efficiency, waste, or when the session seems inefficient.
---

Run the octool binary to get current session metrics:

```bash
~/.octool/bin/octool status
```

Report the results:
- Current view:edit ratio (RED if > 0.7, YELLOW if > 0.5, GREEN if < 0.5)
- Hot files: files read 3+ times this session
- Build cycles: edit→fail→re-edit count
- Prompt quality distribution (LOW/MEDIUM/HIGH)
- Estimated tokens wasted vs useful work
- Context entries currently loaded for this project

Suggest specific fixes based on findings.
```

---

## The Eight Arms — All Autonomous (Requirement #2)

Every arm runs automatically via hooks. The user never invokes them manually.

### Arm 1: File Map Auto-Generator (`arms/filemap.go`)
- **Hook**: `sessionEnd` → `octool finalize`
- **Trigger**: File read >3 times across last 5 sessions (queried from `session_metrics.redundant_reads`)
- **Action**: Auto-creates `context_entries` with type=`file-map`, source=`octool-auto`
- **Log**: `[INFO] [arm:filemap] auto_saved file-map for src/song/[id].tsx (read 41 times across 16 sessions)`

### Arm 2: Build Watcher (`arms/buildwatch.go`)
- **Hook**: `postToolUse` → `octool track` (every tool call)
- **Trigger**: Detects sequence: `edit(file)` → `bash(tsc/go build, fail)` — 3rd occurrence
- **Action**: Returns `systemMessage` coaching to batch edits. Fires once per session.
- **Log**: `[WARN] [arm:buildwatch] 3 build cycles detected. coaching message injected.`

### Arm 3: Context Recovery Optimizer (`arms/recovery.go`)
- **Hook**: `sessionStart` → `octool inject` when `source=="resume"`
- **Trigger**: Session is a resume
- **Action**: Pre-loads top 5 most-read files from previous session as file-map context
- **Log**: `[INFO] [arm:recovery] resume detected. injecting 5 file-maps from previous session.`

### Arm 4: Convention Guard (`arms/convention.go`)
- **Hook**: `userPromptSubmitted` → `octool prompt-check`
- **Trigger**: User message contains "still", "not working", "didn't work"
- **Action**: Increments counter. If 3+ "still" for same issue type and no convention exists → suggests saving one. If convention exists but was violated → promotes to `always_inject`.
- **Log**: `[INFO] [arm:convention] promoted "useAppTheme() mandatory" to always_inject after 3 violations`

### Arm 5: Prompt Coach (`arms/promptcoach.go`)
- **Hook**: `userPromptSubmitted` → `octool prompt-check`
- **Trigger**: Prompt quality scores LOW (no file path, no error, <80 chars)
- **Action**: Returns `systemMessage` with targeted suggestion
- **Log**: `[INFO] [arm:promptcoach] quality=LOW suggestion="mention file path to save ~1500 tokens"`

### Arm 6: Schema Detector (`arms/schema.go`)
- **Hook**: `postToolUse` → `octool track` when `toolName=="view"`
- **Trigger**: Type definition file (matching `*_service.go`, `types.ts`, `*.d.ts`, `models/*`) read >3 times across sessions and no `schema` entry exists
- **Action**: Returns `systemMessage` suggesting to save types as schema entry
- **Log**: `[INFO] [arm:schema] song_service.go read 19 times. suggested schema entry.`

### Arm 7: Resume Advisor (`arms/resume.go`)
- **Hook**: `sessionStart` → `octool inject`
- **Trigger**: New session where last session for this project ended <2 hours ago with >20 tools
- **Action**: Suggests using `copilot resume`. If IS a resume with vague first prompt → suggests being specific.
- **Log**: `[INFO] [arm:resume] last session had 47 tools. suggested resume.`

### Arm 8: View:Edit Monitor (`arms/viewedit.go`)
- **Hook**: `postToolUse` → `octool track` (every tool call)
- **Trigger**: Rolling view:edit ratio exceeds 0.7 over last 10 tool calls
- **Action**: Queries DB for file-map entries for recently viewed files → injects them via `systemMessage`. Flags missing entries for Arm 1.
- **Log**: `[WARN] [arm:viewedit] ratio=0.83 injecting 3 file-maps for recently viewed files`

---

## Go Binary CLI Subcommands

```
octool inject    --cwd PATH --source new|resume     # sessionStart: returns JSON with systemMessage
octool track     --cwd PATH --tool NAME --args JSON --result TYPE  # postToolUse: record + return systemMessage
octool pre-check --cwd PATH --tool NAME --args JSON  # preToolUse: check file-map cache
octool prompt-check --cwd PATH --text "..."          # userPromptSubmitted: score + coach
octool finalize  --cwd PATH                          # sessionEnd: compute metrics, auto-save entries
octool track-error --cwd PATH --name TYPE --message TEXT  # errorOccurred: log error
octool status                                         # Return current session efficiency JSON
octool fetch-session [--limit N] [--project PATH] [--all] [--dry-run]  # Parse session-state files
octool entries   [--project PATH] [--type TYPE]       # List context entries
octool save      --type TYPE --title TITLE --content CONTENT [--project PATH]  # Manual save
octool delete    --id ID                              # Delete entry
octool serve     --port 37888 --background            # Start dashboard HTTP server
octool version                                        # Print version
```

Every subcommand:
- Reads/writes `~/.octool/octool.db` (GLOBAL)
- Logs to `~/.octool/logs/` (ALWAYS)
- Returns JSON to stdout when called by hook scripts
- Exits with code 0 on success, non-zero on error (errors are caught by `_lib.sh` trap)

---

## Dashboard

URL: `http://localhost:37888` — started automatically by `sessionStart` hook.

Shows data from the GLOBAL database across ALL projects:

1. **Token waste breakdown** — Donut chart, aggregate across sessions
2. **Efficiency timeline** — View:edit ratio per session, all projects
3. **Hot files leaderboard** — Most-read files globally, with/without file-map
4. **Arm activity feed** — Chronological log of all autonomous actions
5. **Context entries manager** — CRUD, filterable by project and type
6. **Imported sessions** — List of parsed session-state files from `/fetch-session`

---

## Build Order

### Phase 1: Repo + Scaffold (Steps 1-6)
1. Create repo structure with `.github/plugin/marketplace.json`
2. Create `plugins/octool/plugin.json`
3. Create `plugins/octool/hooks.json`
4. Create `plugins/octool/scripts/_lib.sh` (shared logging + binary resolver)
5. Create `server/go.mod` with module path
6. Create `server/cmd/octool/main.go` with cobra CLI and all subcommands (stubs)

### Phase 2: Storage + Logger (Steps 7-10)
7. Create `server/internal/logger/logger.go` — file logging to `~/.octool/logs/`
8. Create `server/internal/storage/storage.go` — SQLite init, migrations, all CRUD
9. Write tests for storage (create, read, update, delete, FTS search)
10. Wire logger into all storage operations

### Phase 3: Tracker + Scorer (Steps 11-14)
11. Create `server/internal/tracker/tracker.go` — in-memory session state
12. Create `server/internal/scorer/scorer.go` — prompt quality scoring
13. Create `server/internal/metrics/metrics.go` — waste computation
14. Write tests for tracker, scorer, metrics

### Phase 4: Arms (Steps 15-24)
15. Create `server/internal/arms/manager.go`
16. Create `server/internal/arms/filemap.go`
17. Create `server/internal/arms/buildwatch.go`
18. Create `server/internal/arms/recovery.go`
19. Create `server/internal/arms/convention.go`
20. Create `server/internal/arms/promptcoach.go`
21. Create `server/internal/arms/schema.go`
22. Create `server/internal/arms/resume.go`
23. Create `server/internal/arms/viewedit.go`
24. Write tests for each arm

### Phase 5: Hook Scripts (Steps 25-31)
25. Create `plugins/octool/scripts/session-start.sh`
26. Create `plugins/octool/scripts/session-end.sh`
27. Create `plugins/octool/scripts/user-prompt.sh`
28. Create `plugins/octool/scripts/post-tool-use.sh`
29. Create `plugins/octool/scripts/pre-tool-use.sh`
30. Create `plugins/octool/scripts/error-occurred.sh`
31. `chmod +x` all scripts. Test each with piped JSON stdin.

### Phase 6: Session Parser (Steps 32-34)
32. Create `server/internal/session/parser.go` — parse `~/.copilot/session-state/` files
33. Wire into `octool fetch-session` subcommand
34. Write tests with sample session-state JSON fixtures

### Phase 7: Skills (Steps 35-37)
35. Create `plugins/octool/skills/fetch-session/SKILL.md`
36. Create `plugins/octool/skills/octool-status/SKILL.md`
37. Create `plugins/octool/agents/octool.agent.md` (optional)

### Phase 8: Dashboard (Steps 38-41)
38. Create `server/internal/dashboard/server.go` — HTTP + SSE
39. Create `server/internal/dashboard/templates/index.html`
40. Wire into `octool serve` subcommand
41. Test dashboard with mock + real data

### Phase 9: Build + Package + Test (Steps 42-50)
42. Build Go binary for current platform
43. Copy binary to `plugins/octool/bin/`
44. Install locally: `copilot plugin install ./plugins/octool`
45. Verify: `copilot plugin list` shows octool
46. Test: start `copilot` session → verify `sessionStart` hook fires → check logs
47. Test: send 10 messages → verify `postToolUse` tracking → check `~/.octool/octool.db`
48. Test: run `/fetch-session` → verify session-state parsing
49. Cross-compile all platforms, copy to `plugins/octool/bin/`
50. Push to GitHub, test marketplace install: `copilot plugin marketplace add kristiansnts/octool`

---

## Key Design Decisions

1. **GLOBAL DB at `~/.octool/octool.db`** — Not per-project. Conventions learned in one project are available everywhere. The `project_path` column exists for filtering in the dashboard, but the brain queries globally by default.

2. **Hooks are autonomous, skills are on-demand** — Hooks (sessionStart, postToolUse, etc.) fire automatically every session. Skills (`/fetch-session`, `/octool-status`) are user-invoked supplements. The system works without ever touching a skill.

3. **Logging is mandatory** — Every hook script, every arm decision, every DB write is logged. If something breaks, the developer can `cat ~/.octool/logs/errors-*.log` and see exactly what happened.

4. **Hook scripts are thin, binary is fat** — Scripts just pipe JSON to the Go binary. All logic (arms, tracking, DB, parsing) is in compiled Go. This ensures hooks stay within their timeout limits.

5. **`systemMessage` is the injection mechanism** — This is how Copilot CLI allows hooks to inject context. OcTool returns it from `sessionStart` (initial context), `postToolUse` (when view:edit spikes), and `userPromptSubmitted` (coaching tips).

6. **Never block the agent** — OcTool never returns `permissionDecision: "deny"`. All arms inject suggestions. The agent can ignore them.

---

## References

- Research: `token_efficiency_final.json`, `TOKEN_EFFICIENCY_REPORT.md`
- Copilot CLI plugin creation: https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/plugins-creating
- Plugin reference (plugin.json schema): https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-plugin-reference
- Hooks configuration (I/O schemas): https://docs.github.com/en/copilot/reference/hooks-configuration
- Skills creation: https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-skills
- Marketplace creation: https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/plugins-marketplace
- Plugin install docs: https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/plugins-finding-installing
- Copilot session state: `~/.copilot/session-state/`
- Installed plugins location: `~/.copilot/state/installed-plugins/`
