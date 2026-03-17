---
description: Import past Claude Code or Copilot CLI sessions into OcTool's database. Extracts file maps, conventions, and decision patterns so future sessions start with full context.
argument-hint: [--limit N] [--project PATH] [--all] [--dry-run]
allowed-tools: [Bash]
---

# Fetch Session

Parse previous session state files and import discovered patterns into OcTool's global database.

## Steps

1. Preview what would be imported (dry run first):

```bash
~/.octool/bin/octool fetch-session --dry-run $ARGUMENTS
```

2. If the output looks good, import for real:

```bash
~/.octool/bin/octool fetch-session $ARGUMENTS
```

## Common usages

```bash
# Import the 10 most recent sessions (default)
~/.octool/bin/octool fetch-session

# Import up to 25 sessions
~/.octool/bin/octool fetch-session --limit 25

# Import only sessions for this project
~/.octool/bin/octool fetch-session --project "$PWD"

# Import everything
~/.octool/bin/octool fetch-session --all

# Preview without saving
~/.octool/bin/octool fetch-session --dry-run
```

## After importing

Report:
- Sessions scanned and entries created (by type: file-map, convention, gotcha)
- Duplicates skipped
- Suggest running `/octool-status` to see the newly loaded context in action

## What gets extracted

| Data | Becomes |
|------|---------|
| Files read/edited 3+ times | `file-map` context entry |
| Repeated message patterns | `convention` suggestion |
| Recurring errors + fixes | `gotcha` entry |
