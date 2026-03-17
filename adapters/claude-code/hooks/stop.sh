#!/bin/bash
# Claude Code Stop hook (maps to sessionEnd)
# Input (stdin): {"session_id":"...","stop_reason":"...","cwd":"...",...}
# Called when the Claude Code agent stops/completes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib_claude.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT" | jq -r '.cwd // ""')

[ -z "$CWD" ] && CWD="$PWD"

# ── Hook log ──────────────────────────────────────────────────────────────────
octool_hook_section "Stop"
octool_hook_log "Stop" "cwd=$CWD"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "stop" "cwd=$CWD"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_hook_log "Stop" "status=skipped reason=binary_not_found"
  exit 0
fi

"$OCTOOL_BIN" finalize --cwd "$CWD" 2>>"$OCTOOL_ERROR_LOG" >/dev/null

octool_hook_log "Stop" "status=finalized"
octool_log INFO "stop" "session finalized"

exit 0
