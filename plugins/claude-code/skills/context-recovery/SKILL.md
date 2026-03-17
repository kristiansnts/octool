---
name: context-recovery
description: This skill should be used when starting a new session on an existing project, when resuming work from a previous session, when the user says "where were we", "continue from last time", "remind me what we were doing", or when the agent is about to read a lot of files to re-discover project structure that should already be known.
version: 1.0.0
---

# Context Recovery

Recover and inject relevant context from previous OcTool sessions before doing expensive file exploration.

## When to Apply This Skill

Activate when:
- A new session starts on a known project (cwd matches a project with stored context)
- The user says "resume", "continue", "where were we", "remind me"
- You are about to issue 3+ Read calls to understand project structure
- OcTool's Recovery Arm or Resume Advisor fires at session start

## Recovery Process

### Step 1: Check what OcTool already knows

```bash
# List all context entries for this project
~/.octool/bin/octool entries --project "$PWD"

# Get recent session summary
~/.octool/bin/octool status
```

### Step 2: Use stored context instead of re-reading files

If OcTool has `file-map` entries, use them to navigate directly to relevant files rather than exploring the directory tree.

If OcTool has `convention` entries, apply them immediately rather than reading style guides or example files.

If there is a `CLAUDE.md` in the project root, read it — it contains the most recent session summary and key project facts.

### Step 3: Fill gaps only

After consuming stored context, identify what's genuinely unknown (not covered by file-maps or conventions) and read only those files.

## Reading Priority Order

1. `CLAUDE.md` (project root) — most recent context summary
2. OcTool `file-map` entries — key file locations
3. OcTool `convention` entries — rules to follow
4. OcTool `decision` entries — choices already made
5. OcTool `gotcha` entries — known traps

Only proceed to raw file reads after exhausting stored context.

## Saving New Context

When you discover something new and important during the session, save it immediately:

```bash
~/.octool/bin/octool save \
  --type file-map \
  --title "Key files for [feature]" \
  --content "[file]: [purpose], [file]: [purpose]" \
  --project "$PWD"
```

This ensures future sessions start with this knowledge pre-loaded.
