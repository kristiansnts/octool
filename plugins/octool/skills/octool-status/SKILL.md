---
name: octool-status
description: Show current session token efficiency metrics including view:edit ratio, waste breakdown, hot files, and optimization suggestions. Use when the user asks about token usage, efficiency, waste, or when the session seems inefficient.
---

# OcTool Status

Run the octool binary to get current session metrics:

```bash
~/.octool/bin/octool status
```

Report the results:
- Current view:edit ratio (RED if > 0.7, YELLOW if > 0.5, GREEN if < 0.5)
- Hot files: files read 3+ times this session
- Build cycles: edit→fail→re-edit count
- Prompt quality distribution (LOW/MEDIUM/HIGH)
- Estimated tokens wasted vs useful work
- Context entries currently loaded for this project

Suggest specific fixes based on findings.
