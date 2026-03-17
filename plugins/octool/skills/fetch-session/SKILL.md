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
2. **Convention hints** — Repeated patterns in messages (e.g., always using a specific approach) → suggests `convention` entries
3. **Error patterns** — Recurring errors and their fixes → creates `gotcha` entries

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
