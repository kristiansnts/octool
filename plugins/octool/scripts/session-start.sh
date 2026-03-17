#!/bin/bash
# Hook: sessionStart
# Input: {"timestamp":N,"cwd":"/path","source":"new|resume","initialPrompt":"..."}
# Output: {"systemMessage":"..."} or {}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT"    | jq -r '.cwd // ""')
SOURCE=$(echo "$INPUT" | jq -r '.source // "new"')
PROMPT=$(echo "$INPUT" | jq -r '.initialPrompt // ""')
TS=$(echo "$INPUT"     | jq -r '.timestamp // ""')

# ── Hook log (rich) ──────────────────────────────────────────────────────────
octool_hook_section "sessionStart"
octool_hook_log "sessionStart" \
  "source=$SOURCE" \
  "cwd=$CWD" \
  "timestamp=$TS" \
  "initialPrompt=$(echo "$PROMPT" | cut -c1-80)"
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "session-start" "source=$SOURCE cwd=$CWD"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_log WARN "session-start" "binary not found, skipping"
  octool_hook_log "sessionStart" "status=skipped reason=binary_not_found"
  echo '{}'
  exit 0
fi

# Start dashboard in background if not already running
if ! lsof -i :37888 -sTCP:LISTEN -t >/dev/null 2>&1; then
  "$OCTOOL_BIN" serve --port 37888 --background >>"$OCTOOL_LOG" 2>&1 &
  octool_log INFO "session-start" "dashboard started at http://localhost:37888"
  octool_hook_log "sessionStart" "dashboard=started port=37888"
else
  octool_hook_log "sessionStart" "dashboard=already_running port=37888"
fi

OUTPUT=$("$OCTOOL_BIN" inject --cwd "$CWD" --source "$SOURCE" 2>>"$OCTOOL_ERROR_LOG")
MSG=$(echo "$OUTPUT" | jq -r '.systemMessage // ""' 2>/dev/null)

if [ -n "$MSG" ]; then
  octool_hook_log "sessionStart" "status=injected msg_len=${#MSG}"
  octool_log INFO "session-start" "injected context (${#MSG} chars)"
else
  octool_hook_log "sessionStart" "status=ok no_injection"
fi

echo "$OUTPUT"
