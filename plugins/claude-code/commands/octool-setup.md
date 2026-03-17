---
description: Set up OcTool hooks for Claude Code in the current project. Installs PreToolUse, PostToolUse, Stop, and Notification hooks into .claude/settings.json so OcTool tracks every session automatically. Run once per project.
argument-hint: [--cwd PATH]
allowed-tools: [Bash]
---

# OcTool Setup for Claude Code

Install OcTool hooks into the current project so every Claude Code session is automatically tracked and optimized.

## Steps

1. Verify the OcTool binary is installed and accessible:

```bash
~/.octool/bin/octool version 2>/dev/null || echo "Binary not found — see README for build instructions"
```

2. Ensure the hook adapter scripts are in `~/.octool/adapters/claude-code/hooks/`:

```bash
ls ~/.octool/adapters/claude-code/hooks/ 2>/dev/null || echo "Adapter scripts not found — copy them from adapters/claude-code/hooks/ in the repo"
```

   If missing, copy them from the cloned repo:
   ```bash
   mkdir -p ~/.octool/adapters/claude-code/hooks
   # Replace REPO_PATH with the path where you cloned kristiansnts/octool
   cp REPO_PATH/adapters/claude-code/hooks/* ~/.octool/adapters/claude-code/hooks/
   chmod +x ~/.octool/adapters/claude-code/hooks/*.sh
   ```

3. Run the setup command to write `.claude/settings.json`:

```bash
~/.octool/bin/octool setup-claude --cwd "${ARGUMENTS:-$PWD}"
```

4. Generate the initial `CLAUDE.md` context file:

```bash
~/.octool/bin/octool generate-claude-md --cwd "${ARGUMENTS:-$PWD}"
```

5. Report the outcome:
   - Path to `.claude/settings.json` that was created
   - That the hooks are now active: `PreToolUse`, `PostToolUse`, `Stop`, `Notification`
   - That `CLAUDE.md` was generated in the project root
   - Remind the user to restart Claude Code for the hooks to take effect

## What gets installed

| Hook | When | What OcTool does |
|------|------|-----------------|
| `PreToolUse` | Before any tool call | Blocks redundant reads (file read 3+ times) — exits 2 to enforce |
| `PostToolUse` | After any tool call | Tracks file access, fires Arms 2/6/8 |
| `Stop` | Session end | Finalizes metrics, saves file maps (Arm 1) |
| `Notification` | On errors | Records error patterns |

## Notes

- Safe to re-run — overwrites the previous `.claude/settings.json`
- Run once per project (not per session)
- The generated `CLAUDE.md` is updated each session end via the `Stop` hook
- Regenerate `CLAUDE.md` at any time with: `~/.octool/bin/octool generate-claude-md`
