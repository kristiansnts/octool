---
name: token-efficiency
description: This skill should be used when the session is reading far more than writing, when the same files are being read repeatedly, when the user says "still not working" or "try again" multiple times, when there are repeated build failures, or when the user asks about token efficiency, quota usage, or session waste.
version: 1.0.0
---

# Token Efficiency Advisor

OcTool tracks 8 waste patterns and fires autonomously. This skill gives you the knowledge to act on what OcTool reports.

## When to Apply This Skill

Activate when you observe any of these signals:
- View:edit ratio climbing above 0.7 (reading far more than writing)
- Same file read 3 or more times without a new edit
- Build cycle: edit → bash fail → re-edit, repeated 3+ times
- User has said "still not working" or "try again" 3+ times
- User asks about token usage, quota, or efficiency

## The 4 Biggest Waste Sources (in order)

### 1. Repeated File Reads (25% of wasted tokens)

**Symptom**: Reading the same file 4-5 times across a session.

**Fix**: Before reading a file again, ask yourself: *"Do I already have this content in context?"* If yes, use the cached version. OcTool's `PreToolUse` hook blocks reads after 3 times — trust that cached content.

### 2. Build Loops (10% of wasted tokens)

**Symptom**: edit → build fail → re-edit → build fail, 3+ cycles.

**Fix**: Stop the loop after cycle 2. Instead of another re-edit, ask the user to share the full error output and diagnose the root cause. One targeted fix beats five guesses.

### 3. Context Loss Between Sessions (8% of wasted tokens)

**Symptom**: Starting fresh without knowing the project structure, conventions, or previous decisions.

**Fix**: The Recovery Arm and Resume Advisor inject previous context automatically. If context is thin, run `/fetch-session` to bootstrap from past sessions.

### 4. Vague Prompts (5% of wasted tokens)

**Symptom**: Short prompts with no file path, no expected outcome, no constraints. These cause the agent to explore broadly before converging.

**Fix**: Before executing, check if the prompt includes:
- A specific file or module
- The expected outcome
- Any constraints (don't change X, use Y pattern)

If missing, ask the user for clarification in one targeted question.

## Quick Reference

```bash
# Check current session metrics
~/.octool/bin/octool status

# See stored context for this project
~/.octool/bin/octool entries --project "$PWD"

# Save a new context entry
~/.octool/bin/octool save --type convention --title "..." --content "..." --project "$PWD"
```
