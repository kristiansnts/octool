#!/bin/bash
# Hook: preToolUse
# Input: {"timestamp":N,"cwd":"/path","toolName":"view","toolArgs":"{...}"}
# Output: {"systemMessage":"..."} or {} — never deny

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT"  | jq -r '.cwd // ""')
TOOL=$(echo "$INPUT" | jq -r '.toolName // ""')
ARGS=$(echo "$INPUT" | jq -r '.toolArgs // "{}"')
TS=$(echo "$INPUT"   | jq -r '.timestamp // ""')

FILE_PATH=$(echo "$ARGS" | jq -r '.path // .file // ""' 2>/dev/null)

# ── Hook log (rich) ──────────────────────────────────────────────────────────
octool_hook_section "preToolUse"
octool_hook_log "preToolUse" \
  "tool=$TOOL" \
  "cwd=$CWD" \
  "timestamp=$TS" \
  "file=$FILE_PATH"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "pre-tool-use" "tool=$TOOL"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_hook_log "preToolUse" "status=skipped reason=binary_not_found"
  echo '{}'
  exit 0
fi

OUTPUT=$("$OCTOOL_BIN" pre-check --cwd "$CWD" --tool "$TOOL" --args "$ARGS" 2>>"$OCTOOL_ERROR_LOG")
MSG=$(echo "$OUTPUT" | jq -r '.systemMessage // ""' 2>/dev/null)

if [ -n "$MSG" ]; then
  octool_hook_log "preToolUse" "status=injected msg_len=${#MSG}"
else
  octool_hook_log "preToolUse" "status=ok"
fi

echo "$OUTPUT"
