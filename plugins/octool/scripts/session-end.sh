#!/bin/bash
# Hook: sessionEnd
# Input: {"timestamp":N,"cwd":"/path"}
# Output: {} (no systemMessage needed)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT" | jq -r '.cwd // ""')
TS=$(echo "$INPUT"  | jq -r '.timestamp // ""')

# ── Hook log (rich) ──────────────────────────────────────────────────────────
octool_hook_section "sessionEnd"
octool_hook_log "sessionEnd" \
  "cwd=$CWD" \
  "timestamp=$TS"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "session-end" "cwd=$CWD"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_log WARN "session-end" "binary not found, skipping"
  octool_hook_log "sessionEnd" "status=skipped reason=binary_not_found"
  echo '{}'
  exit 0
fi

"$OCTOOL_BIN" finalize --cwd "$CWD" 2>>"$OCTOOL_ERROR_LOG"
octool_hook_log "sessionEnd" "status=finalized"
octool_log INFO "session-end" "finalize complete"
echo '{}'
