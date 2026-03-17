#!/bin/bash
# Hook: postToolUse
# Input: {"timestamp":N,"cwd":"/path","toolName":"view","toolArgs":"{...}","toolResult":{"resultType":"success"}}
# Output: {"systemMessage":"..."} or {}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT"         | jq -r '.cwd // ""')
TOOL=$(echo "$INPUT"        | jq -r '.toolName // ""')
ARGS=$(echo "$INPUT"        | jq -r '.toolArgs // "{}"')
RESULT_TYPE=$(echo "$INPUT" | jq -r '.toolResult.resultType // "success"')
TS=$(echo "$INPUT"          | jq -r '.timestamp // ""')

FILE_PATH=$(echo "$ARGS" | jq -r '.path // .file // ""' 2>/dev/null)

# ── Hook log (rich) ──────────────────────────────────────────────────────────
octool_hook_section "postToolUse"
octool_hook_log "postToolUse" \
  "tool=$TOOL" \
  "cwd=$CWD" \
  "timestamp=$TS" \
  "file=$FILE_PATH" \
  "result=$RESULT_TYPE"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "post-tool-use" "tool=$TOOL file=$FILE_PATH"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_log WARN "post-tool-use" "binary not found, skipping"
  octool_hook_log "postToolUse" "status=skipped reason=binary_not_found"
  echo '{}'
  exit 0
fi

OUTPUT=$("$OCTOOL_BIN" track --cwd "$CWD" --tool "$TOOL" --args "$ARGS" --result "$RESULT_TYPE" 2>>"$OCTOOL_ERROR_LOG")
MSG=$(echo "$OUTPUT" | jq -r '.systemMessage // ""' 2>/dev/null)

if [ -n "$MSG" ]; then
  octool_hook_log "postToolUse" "status=arm_fired msg_len=${#MSG}"
else
  octool_hook_log "postToolUse" "status=tracked"
fi

echo "$OUTPUT"
