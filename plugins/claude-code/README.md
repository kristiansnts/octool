# OcTool — Claude Code Plugin

Prompt efficiency layer for Claude Code. OcTool hooks into every session to auto-inject context from previous sessions, coach prompt quality, and help the agent succeed on the first try — reducing follow-up messages and saving Claude Pro quota.

## Why OcTool?

Claude Pro quota is counted **per user message** (not per tool call). The real cost comes from follow-up prompts — every "try again", "no I meant...", "look at this file", or "where was I?" is another quota hit.

OcTool reduces the number of follow-up messages you need to send:

| Without OcTool | With OcTool |
|---|---|
| Message 1: "refactor auth" → agent lacks context, reads 10 files | Message 1: "refactor auth" → context injected, agent knows the codebase |
| Message 2: "no, look at src/auth/..." → guiding the agent | Agent succeeds on the first try |
| Message 3: "use our existing patterns" → fixing conventions | ✅ Done — 1 quota hit |
| Message 4: "fix the build error" → another follow-up | |
| ❌ 4 quota hits for one task | |

Additionally, OcTool's `PreToolUse` hook exits with code 2 to **enforce** blocking of redundant reads (same file read 3+ times). This is unique to Claude Code — unlike Copilot CLI where denies are advisory-only.

---

## Installation

### 1. Add the marketplace and install the plugin

```bash
/plugin marketplace add kristiansnts/octool
/plugin install octool
```

Or browse for it in `/plugin > Discover`.

### 2. Build and install the binary

Requirements: Go 1.21+

```bash
git clone https://github.com/kristiansnts/octool
cd octool/server
go build -o ~/.octool/bin/octool ./cmd/octool/
```

Or use a pre-built binary from [Releases](https://github.com/kristiansnts/octool/releases).

### 3. Install the hook adapter scripts

```bash
mkdir -p ~/.octool/adapters/claude-code/hooks
cp /path/to/octool/adapters/claude-code/hooks/* ~/.octool/adapters/claude-code/hooks/
chmod +x ~/.octool/adapters/claude-code/hooks/*.sh
```

### 4. Set up hooks for your project

In any Claude Code session, run:

```
/octool-setup
```

This creates `.claude/settings.json` wiring OcTool into Claude Code's hook system, and generates an initial `CLAUDE.md` with project context.

---

## Slash Commands

| Command | Description |
|---------|-------------|
| `/octool-status` | Show session metrics and identify waste sources |
| `/fetch-session` | Import past session history into OcTool's DB |
| `/octool-setup` | Install hooks + generate CLAUDE.md for current project |
| `/save-context` | Manually save a convention, file-map, decision, or gotcha |

---

## Auto-Activated Skills

These skills activate automatically based on session context — no command needed:

| Skill | Activates when... |
|-------|-------------------|
| **token-efficiency** | View:edit ratio is high, build loops detected, or user says "still not working" |
| **context-recovery** | Starting a session on a known project, or user asks "where were we" |
| **convention-guard** | Writing/editing code, or user corrects a convention violation |

---

## How the Hooks Work

Once set up via `/octool-setup`, these hooks fire automatically:

| Hook | Trigger | What OcTool does |
|------|---------|-----------------|
| `PreToolUse` | Before any tool call | Blocks redundant reads (file read 3+ times) — **exit 2 enforces the block** |
| `PostToolUse` | After any tool call | Tracks file access, fires Arms 2/6/8 (build watch, schema guard, view:edit ratio) |
| `Stop` | Session end | Runs Arm 1 (file map generator), saves session metrics, updates CLAUDE.md |
| `Notification` | On errors/warnings | Records error patterns for future sessions |

---

## The 8 Arms

| # | Arm | When it fires | What it does |
|---|-----|--------------|--------------|
| 1 | Filemap Generator | Session end | Auto-saves directory snapshot so the agent already knows your project structure |
| 2 | Build Watcher | PostToolUse | Detects build loops (edit→fail×3) and injects a warning to break the cycle |
| 3 | Recovery Arm | Session start | Re-injects high-value context from previous sessions |
| 4 | Convention Enforcer | User message | Checks the prompt against stored coding conventions |
| 5 | Prompt Coach | User message | Scores prompt quality and suggests rewrites |
| 6 | Schema Guard | PostToolUse | Detects drift between tool args and stored schema snapshots |
| 7 | Resume Advisor | Session start (resume) | Summarizes what was in progress when the previous session ended |
| 8 | View:Edit Ratio | PostToolUse | Warns when reading far more than writing |

---

## The Database

All state is stored in `~/.octool/octool.db` (SQLite). This is shared across all projects and both Copilot CLI and Claude Code sessions.

```bash
# View stored context
~/.octool/bin/octool entries

# Start the dashboard
~/.octool/bin/octool serve --port 37888
# → open http://localhost:37888
```

---

## License

MIT
