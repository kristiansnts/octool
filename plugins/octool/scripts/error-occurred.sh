#!/bin/bash
# Hook: errorOccurred
# Input: {"timestamp":N,"cwd":"/path","error":{"name":"ErrorType","message":"description"}}
# Output: {} (no systemMessage)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT"     | jq -r '.cwd // ""')
ERR_NAME=$(echo "$INPUT" | jq -r '.error.name // "Unknown"')
ERR_MSG=$(echo "$INPUT"  | jq -r '.error.message // ""')
TS=$(echo "$INPUT"       | jq -r '.timestamp // ""')

ERR_PREVIEW=$(echo "$ERR_MSG" | cut -c1-120 | tr '\n' ' ')

# ── Hook log (rich) ──────────────────────────────────────────────────────────
octool_hook_section "errorOccurred"
octool_hook_log "errorOccurred" \
  "cwd=$CWD" \
  "timestamp=$TS" \
  "error_name=$ERR_NAME" \
  "message=\"$ERR_PREVIEW\""
# ─────────────────────────────────────────────────────────────────────────────

octool_log ERROR "error-occurred" "cwd=$CWD name=$ERR_NAME message=$ERR_MSG"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_hook_log "errorOccurred" "status=skipped reason=binary_not_found"
  echo '{}'
  exit 0
fi

"$OCTOOL_BIN" track-error --cwd "$CWD" --name "$ERR_NAME" --message "$ERR_MSG" 2>>"$OCTOOL_ERROR_LOG" || true
octool_hook_log "errorOccurred" "status=recorded"
echo '{}'
