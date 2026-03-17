---
description: Show current session token efficiency metrics — view:edit ratio, hot files, build cycles, prompt quality distribution, and optimization suggestions
argument-hint: [--project PATH]
allowed-tools: [Bash]
---

# OcTool Status

Show the current session's token efficiency metrics and identify the biggest sources of waste.

## Steps

1. Run the octool binary:

```bash
~/.octool/bin/octool status
```

2. If the binary is not found at `~/.octool/bin/octool`, try the platform-specific path:

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
~/.octool/bin/octool-${OS}-${ARCH} status
```

3. Report the results with clear signal/noise separation:

| Metric | Green | Yellow | Red |
|--------|-------|--------|-----|
| View:Edit ratio | < 0.5 | 0.5–0.7 | > 0.7 |
| Build cycles | 0–1 | 2 | 3+ |
| Prompt quality | HIGH | MEDIUM | LOW |

4. For each RED metric, suggest one concrete fix:
   - **High view:edit ratio** → "Run `/fetch-session` to capture your most-read files as context entries"
   - **Build cycles** → "Batch your edits — fix the root cause before re-running the build"
   - **Low-quality prompts** → "Include the file path and describe the exact change needed"

## Optional: filter by project

```bash
~/.octool/bin/octool entries --project "$ARGUMENTS"
```
