#!/bin/bash
# Claude Code Notification hook
# Input (stdin): {"message":"...","title":"...","session_id":"...","cwd":"...",...}
# Called when Claude Code emits a notification (often errors or warnings)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib_claude.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT"     | jq -r '.cwd // ""')
ERROR_NAME=$(echo "$INPUT"  | jq -r '.title // "notification"')
ERROR_MSG=$(echo "$INPUT"   | jq -r '.message // ""')

[ -z "$CWD" ] && CWD="$PWD"

# ── Hook log ──────────────────────────────────────────────────────────────────
octool_hook_section "Notification"
octool_hook_log "Notification" \
  "cwd=$CWD" \
  "name=$ERROR_NAME"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "notification" "name=$ERROR_NAME cwd=$CWD"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_hook_log "Notification" "status=skipped reason=binary_not_found"
  exit 0
fi

"$OCTOOL_BIN" track-error \
  --cwd "$CWD" \
  --name "$ERROR_NAME" \
  --message "$ERROR_MSG" \
  2>>"$OCTOOL_ERROR_LOG" >/dev/null

octool_hook_log "Notification" "status=tracked"

exit 0
