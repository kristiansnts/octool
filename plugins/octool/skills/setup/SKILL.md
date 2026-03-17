---
name: octool-setup
description: Set up OcTool hooks for the current project. Creates .github/hooks/octool.json with absolute paths so Copilot CLI triggers the dashboard and tracking automatically on every session start. Run this once per project.
---

# OcTool Setup

Install OcTool hooks into the current project so the dashboard auto-starts and all session events are tracked.

```bash
~/.octool/bin/octool setup --cwd "$PWD"
```

After running, report:
- The path where the hooks file was written (`.github/hooks/octool.json`)
- That hooks are now active for this project
- Remind the user to restart Copilot CLI in this directory for the hooks to take effect
- The dashboard will be available at http://localhost:37888

## What this does

Creates `.github/hooks/octool.json` in the project's git root with absolute paths to the OcTool hook scripts. Copilot CLI reads this file on every session start and fires the hooks automatically.

Hooks installed:
- **sessionStart** — auto-starts the dashboard, injects saved context
- **userPromptSubmitted** — coaches low-quality prompts
- **preToolUse** — pre-tool checks
- **postToolUse** — tracks file reads/edits, fires arms
- **sessionEnd** — finalizes and saves session metrics
- **errorOccurred** — records errors

## Notes

- Safe to re-run — overwrites the previous hooks file with fresh paths
- Run once per project (not per session)
- The `.github/hooks/octool.json` file can be committed to share hooks with the team
