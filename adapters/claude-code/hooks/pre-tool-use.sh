#!/bin/bash
# Claude Code PreToolUse hook
# Input (stdin): {"tool_name":"Read","tool_input":{"file_path":"src/app.ts"},...}
# Output: text printed to stdout is injected as context by Claude Code
# Exit code 2 = deny (tool call blocked); exit 0 = allow

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib_claude.sh"

INPUT=$(cat)
CLAUDE_TOOL=$(echo "$INPUT" | jq -r '.tool_name // ""')
TOOL_INPUT=$(echo "$INPUT"  | jq -r '.tool_input // {}' 2>/dev/null || echo '{}')
CWD=$(echo "$INPUT"         | jq -r '.cwd // ""')

# Fall back to $PWD when the hook payload doesn't include cwd
[ -z "$CWD" ] && CWD="$PWD"

COPILOT_TOOL=$(claude_to_copilot_tool "$CLAUDE_TOOL")

# ── Hook log ──────────────────────────────────────────────────────────────────
octool_hook_section "PreToolUse"
octool_hook_log "PreToolUse" \
  "claude_tool=$CLAUDE_TOOL" \
  "copilot_tool=$COPILOT_TOOL" \
  "cwd=$CWD"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "pre-tool-use" "tool=$CLAUDE_TOOL ($COPILOT_TOOL)"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_hook_log "PreToolUse" "status=skipped reason=binary_not_found"
  exit 0
fi

OUTPUT=$("$OCTOOL_BIN" pre-check \
  --cwd "$CWD" \
  --tool "$COPILOT_TOOL" \
  --args "$TOOL_INPUT" \
  2>>"$OCTOOL_ERROR_LOG")

DECISION=$(echo "$OUTPUT"  | jq -r '.permissionDecision // ""'       2>/dev/null)
REASON=$(echo "$OUTPUT"    | jq -r '.permissionDecisionReason // ""'  2>/dev/null)
MSG=$(echo "$OUTPUT"       | jq -r '.systemMessage // ""'             2>/dev/null)

if [ "$DECISION" = "deny" ]; then
  octool_hook_log "PreToolUse" "status=denied reason=$REASON"
  octool_log INFO "pre-tool-use" "DENIED tool=$CLAUDE_TOOL reason=$REASON"
  # Print denial reason so Claude can see it; exit 2 enforces the block
  if [ -n "$REASON" ]; then
    echo "$REASON"
  fi
  exit 2
fi

if [ -n "$MSG" ]; then
  octool_hook_log "PreToolUse" "status=injected msg_len=${#MSG}"
  echo "$MSG"
else
  octool_hook_log "PreToolUse" "status=allowed"
fi

exit 0
