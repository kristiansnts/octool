---
name: convention-guard
description: This skill should be used when writing or editing code, when the user asks to add a feature, refactor, or fix a bug — check stored project conventions before making changes. Also activate when the user corrects a previous output by saying "no, use our pattern", "that's not how we do it here", or "follow our conventions".
version: 1.0.0
---

# Convention Guard

Check and apply stored project conventions before writing any code, so the agent gets it right the first time and avoids "no, that's not our pattern" follow-up corrections.

## When to Apply This Skill

Activate before:
- Writing new code (any file edit)
- Refactoring existing code
- Adding dependencies or imports
- Choosing between multiple valid approaches

Also activate after a user correction that suggests a convention violation.

## Convention Check Process

### Step 1: Load conventions before coding

```bash
~/.octool/bin/octool entries --project "$PWD" --type convention
```

Review each convention entry. Apply all that are relevant to the current task.

### Step 2: Check file-map entries for structural patterns

```bash
~/.octool/bin/octool entries --project "$PWD" --type file-map
```

Use file-map entries to understand where code of a certain type belongs (e.g., "all API handlers in src/handlers/").

### Step 3: Apply before writing

Before making any edit, run through the convention checklist mentally:
- Naming: does the name match project conventions?
- Structure: does the file location match the file-map?
- Style: imports, exports, error handling, logging — all per convention?
- Testing: does this project require tests alongside new code?

## When a Convention is Violated (User Corrects You)

When the user says "that's not how we do it here" or similar:

1. Acknowledge and fix the immediate code
2. Save the pattern as a convention so it never happens again:

```bash
~/.octool/bin/octool save \
  --type convention \
  --title "<concise name for the pattern>" \
  --content "<specific rule: when it applies, what to do instead>" \
  --project "$PWD"
```

3. Report: "Saved as a convention — I'll follow this pattern in all future edits."

## Convention Format Best Practices

Good conventions are:
- **Specific** — "Use `Result<T, AppError>` for all service layer return types" not "use good error handling"
- **Actionable** — tells you exactly what to do, not just what to avoid
- **Scoped** — specifies where the rule applies (all files, only in src/api/, etc.)
