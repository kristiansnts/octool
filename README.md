# OcTool

Prompt efficiency layer for Copilot CLI. OcTool runs as a plugin that hooks into every session, auto-injecting context from previous sessions, coaching prompt quality, and helping the agent succeed on the first try — so you send fewer follow-up messages and use fewer premium requests.

## How it works

OcTool installs as a Copilot CLI plugin and fires 8 autonomous arms across the session lifecycle:

- **Session start**: injects recovered context and resume summaries
- **Post-tool-use**: watches for build loops, schema drift, and view/edit ratio spikes
- **User prompt**: enforces coding conventions and coaches prompt quality
- **Session end**: auto-generates file maps for future sessions

All state is persisted in a local SQLite database (`~/.octool/octool.db`).

---

## Why OcTool?

GitHub Copilot premium requests are counted **per user message**, not per tool call. When you send one prompt and the agent runs 20 tool calls autonomously, that's still one premium request. The real cost comes from **follow-up prompts** — every time you say "try again", "no I meant...", "look at this file", or "where was I?", that's another premium request.

OcTool reduces the number of follow-up prompts you need to send:

| Without OcTool | With OcTool |
|---|---|
| Prompt 1: "refactor auth module" → agent lacks context, reads wrong files | Prompt 1: "refactor auth module" → context already injected, agent knows the codebase |
| Prompt 2: "no, look at src/auth/..." → guiding the agent | Agent succeeds on first try |
| Prompt 3: "use our existing patterns" → fixing conventions | ✅ Done — 1 premium request |
| Prompt 4: "fix the build error" → another follow-up | |
| ❌ 4 premium requests for one task | |

---

## Installation

### 1. Install via marketplace

```bash
copilot plugin install octool
```

### 2. Enable the plugin

```bash
copilot plugin enable octool
```

### 3. Restart your Copilot CLI session

The plugin hooks activate automatically on the next session start.

---

## The 8 Arms

| # | Arm | Trigger | Description |
|---|-----|---------|-------------|
| 1 | Filemap Generator | Session end | Auto-saves a directory tree snapshot so the agent already knows your project structure in future sessions — no "look at src/..." follow-ups needed |
| 2 | Build Watcher | Post-tool-use | Detects repeated build failures and injects a warning to break the loop — helps fix the root cause in one follow-up instead of five |
| 3 | Recovery Arm | Session start | Re-injects high-value context entries from previous sessions — eliminates "remind me" and "I was working on..." follow-up prompts |
| 4 | Convention Enforcer | User prompt | Checks the prompt against stored coding conventions — prevents "no, use our coding style" follow-up corrections |
| 5 | Prompt Coach | User prompt | Scores prompt quality and suggests rewrites that help the agent succeed on the first try |
| 6 | Schema Guard | Post-tool-use | Detects drift between tool arguments and stored schema snapshots — keeps the agent aligned within the current turn |
| 7 | Resume Advisor | Session start (resume) | Summarizes what was in-progress when the previous session ended — no "what was I working on?" prompt needed |
| 8 | View:Edit Ratio | Post-tool-use | Warns when the session is reading far more than writing — surfaces inefficiency so you can course-correct early |

---

## Available Skills

Skills can be invoked with `/skill-name` inside a Copilot CLI session.

### `/fetch-session`

Imports recent Copilot CLI session logs into the OcTool database, extracting context entries (decisions, patterns, file maps) for use by the arms.

Options:
- `--limit N` — number of sessions to import (default: 10)
- `--project PATH` — filter by project directory
- `--all` — import all available sessions
- `--dry-run` — preview without saving

### `/octool-status`

Displays the current session's token efficiency metrics: view count, edit count, view:edit ratio, and a summary of which arms have fired.

---

## Dashboard

The OcTool dashboard provides a web UI for browsing context entries, session metrics, and arm activity.

Start the dashboard:

```bash
octool serve --port 37888
```

Then open [http://localhost:37888](http://localhost:37888) in your browser.

---

## CLI Reference

```
octool [command]

Commands:
  inject         Inject context at session start (fires Arms 3 & 7)
  track          Record a tool call and run post-tool-use arms (Arms 2, 6, 8)
  prompt-check   Analyze a user prompt (fires Arms 4 & 5)
  finalize       Run session-end arms and save metrics (Arm 1)
  entries        List stored context entries
  save           Save a context entry manually
  delete         Delete a context entry by ID
  fetch-session  Import session logs from Copilot CLI history
  status         Show current session token efficiency metrics
  serve          Start the dashboard HTTP server
  version        Print version
```

---

## Build from Source

Requirements: Go 1.21+

```bash
git clone https://github.com/kristiansnts/octool
cd octool/server

# Build for current platform
go build -o ../plugins/octool/bin/octool-$(go env GOOS)-$(go env GOARCH) ./cmd/octool/

# Cross-compile for all platforms
GOOS=linux  GOARCH=amd64 go build -o ../plugins/octool/bin/octool-linux-amd64    ./cmd/octool/
GOOS=linux  GOARCH=arm64 go build -o ../plugins/octool/bin/octool-linux-arm64    ./cmd/octool/
GOOS=darwin GOARCH=amd64 go build -o ../plugins/octool/bin/octool-darwin-amd64   ./cmd/octool/
GOOS=darwin GOARCH=arm64 go build -o ../plugins/octool/bin/octool-darwin-arm64   ./cmd/octool/
GOOS=windows GOARCH=amd64 go build -o ../plugins/octool/bin/octool-windows-amd64.exe ./cmd/octool/

# Run tests
go test ./...
```

---

## License

MIT
