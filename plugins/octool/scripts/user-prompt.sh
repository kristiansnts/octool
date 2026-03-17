#!/bin/bash
# Hook: userPromptSubmitted
# Input: {"timestamp":N,"cwd":"/path","prompt":"user message"}
# Output: {"systemMessage":"..."} or {}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib.sh"

INPUT=$(cat)
CWD=$(echo "$INPUT"    | jq -r '.cwd // ""')
PROMPT=$(echo "$INPUT" | jq -r '.prompt // ""')
TS=$(echo "$INPUT"     | jq -r '.timestamp // ""')

PROMPT_LEN=${#PROMPT}
PROMPT_PREVIEW=$(echo "$PROMPT" | cut -c1-100 | tr '\n' ' ')

# ── Hook log (rich) ──────────────────────────────────────────────────────────
octool_hook_section "userPromptSubmitted"
octool_hook_log "userPromptSubmitted" \
  "cwd=$CWD" \
  "timestamp=$TS" \
  "prompt_len=$PROMPT_LEN" \
  "prompt=\"$PROMPT_PREVIEW\""
# ─────────────────────────────────────────────────────────────────────────────

octool_log INFO "user-prompt" "cwd=$CWD len=$PROMPT_LEN"

if [ -z "$OCTOOL_BIN" ] || [ ! -x "$OCTOOL_BIN" ]; then
  octool_log WARN "user-prompt" "binary not found, skipping"
  octool_hook_log "userPromptSubmitted" "status=skipped reason=binary_not_found"
  echo '{}'
  exit 0
fi

OUTPUT=$("$OCTOOL_BIN" prompt-check --cwd "$CWD" --text "$PROMPT" 2>>"$OCTOOL_ERROR_LOG")
MSG=$(echo "$OUTPUT" | jq -r '.systemMessage // ""' 2>/dev/null)

if [ -n "$MSG" ]; then
  TIP_PREVIEW=$(echo "$MSG" | cut -c1-80 | tr '\n' ' ')
  octool_hook_log "userPromptSubmitted" "status=coached tip=\"$TIP_PREVIEW\""
  octool_log INFO "user-prompt" "coached (${#MSG} chars)"
else
  octool_hook_log "userPromptSubmitted" "status=ok quality=acceptable"
fi

echo "$OUTPUT"
