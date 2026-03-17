---
description: Manually save a context entry (convention, file-map, decision, or gotcha) to OcTool's global database so it is auto-injected into future sessions
argument-hint: --type <type> --title "<title>" --content "<content>"
allowed-tools: [Bash]
---

# Save Context

Manually persist a piece of project knowledge to OcTool's global database.

## Usage

Ask the user for the type, title, and content if not provided, then run:

```bash
~/.octool/bin/octool save \
  --type  "<type>" \
  --title "<title>" \
  --content "<content>" \
  --project "$PWD"
```

## Types

| Type | When to use |
|------|-------------|
| `convention` | Coding style rules, naming patterns, architecture decisions the agent should always follow |
| `file-map` | Which files are most important for a feature or module |
| `decision` | A one-time architectural or design choice that should not be revisited |
| `gotcha` | A known error or trap and its fix |

## Examples

```bash
# Save a coding convention
~/.octool/bin/octool save \
  --type convention \
  --title "Always use named exports" \
  --content "All modules must use named exports, not default exports. Applies to every .ts file." \
  --project "$PWD"

# Save a file map
~/.octool/bin/octool save \
  --type file-map \
  --title "Auth module entry points" \
  --content "src/auth/index.ts (exports), src/auth/session.ts (JWT), src/auth/guards/ (route guards)" \
  --project "$PWD"

# Save a gotcha
~/.octool/bin/octool save \
  --type gotcha \
  --title "prisma generate required after schema change" \
  --content "Always run npx prisma generate after any change to schema.prisma or the TypeScript types will be stale." \
  --project "$PWD"
```

## After saving

Confirm the entry was saved and report the entry ID. Suggest running `/octool-status` to see the updated context count.
