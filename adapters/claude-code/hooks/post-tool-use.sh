#!/bin/bash
# Claude Code PostToolUse hook
# Input (stdin): {"tool_name":"Read","tool_input":{...},"tool_response":{...},...}
# Output: text printed to stdout is injected as context by Claude Code

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib_claude.sh"

INPUT=$(cat)
CLAUDE_TOOL=$(echo "$INPUT"  | jq -r '.tool_name // ""')
TOOL_INPUT=$(echo "$INPUT"   | jq -r '.tool_input // {}' 2>/dev/null || echo '{}')
TOOL_OUTPUT=$(echo "$INPUT"  | jq -r '.tool_response // {}' 2>/dev/null || echo '{}')
CWD=$(echo "$INPUT"          | jq -r '.cwd // ""')

[ -z "$CWD" ] && CWD="$PWD"

COPILOT_TOOL=$(claude_to_copilot_tool "$CLAUDE_TOOL")

# Determine result type from tool output.
# Claude Code doesn't provide a structured error status in tool_response,
# so we use keyword matching as a best-effort heuristic consistent with
# how the Copilot CLI adapter handles result classification.
TOOL_OUTPUT_STR=$(echo "$INPUT" | jq -r '.tool_response | tostring' 2>/dev/null || echo "")
RESULT_TYPE="success"
if echo "$TOOL_OUTPUT_STR" | grep -qi "error\|failed\|exception\|traceback"; then
  RESULT_TYPE="error"
fi

# ── Hook log ──────────────────────────────────────────────────────────────────
octool_hook_section "PostToolUse"
octool_hook_log "PostToolUse" \
  "claude_tool=$CLAUDE_TOOL" \
  "copilot_tool=$COPILOT_TOOL" \
  "cwd=$CWD" \
  "result=$RESULT_TYPE"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "post-tool-use" "tool=$CLAUDE_TOOL ($COPILOT_TOOL) result=$RESULT_TYPE"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_hook_log "PostToolUse" "status=skipped reason=binary_not_found"
  exit 0
fi

OUTPUT=$("$OCTOOL_BIN" track \
  --cwd "$CWD" \
  --tool "$COPILOT_TOOL" \
  --args "$TOOL_INPUT" \
  --result "$RESULT_TYPE" \
  2>>"$OCTOOL_ERROR_LOG")

MSG=$(echo "$OUTPUT" | jq -r '.systemMessage // ""' 2>/dev/null)

if [ -n "$MSG" ]; then
  octool_hook_log "PostToolUse" "status=arm_fired msg_len=${#MSG}"
  echo "$MSG"
else
  octool_hook_log "PostToolUse" "status=tracked"
fi

exit 0
