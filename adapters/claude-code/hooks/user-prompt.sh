#!/bin/bash
# Claude Code UserPromptSubmit hook
# Input (stdin): {"prompt":"...","session_id":"...","cwd":"...",...}
# Output: text printed to stdout is injected as context by Claude Code

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib_claude.sh"

INPUT=$(cat)
PROMPT=$(echo "$INPUT" | jq -r '.prompt // ""')
CWD=$(echo "$INPUT"    | jq -r '.cwd // ""')

[ -z "$CWD" ] && CWD="$PWD"

# ── Hook log ──────────────────────────────────────────────────────────────────
octool_hook_section "UserPromptSubmit"
octool_hook_log "UserPromptSubmit" \
  "cwd=$CWD" \
  "prompt_len=${#PROMPT}"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "user-prompt" "cwd=$CWD"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_hook_log "UserPromptSubmit" "status=skipped reason=binary_not_found"
  exit 0
fi

OUTPUT=$("$OCTOOL_BIN" prompt-check \
  --cwd "$CWD" \
  --text "$PROMPT" \
  2>>"$OCTOOL_ERROR_LOG")

MSG=$(echo "$OUTPUT" | jq -r '.systemMessage // ""' 2>/dev/null)

if [ -n "$MSG" ]; then
  octool_hook_log "UserPromptSubmit" "status=arm_fired msg_len=${#MSG}"
  echo "$MSG"
else
  octool_hook_log "UserPromptSubmit" "status=ok"
fi

exit 0
