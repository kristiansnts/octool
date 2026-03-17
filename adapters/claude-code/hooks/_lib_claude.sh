#!/bin/bash
# adapters/claude-code/hooks/_lib_claude.sh — sourced by all Claude Code hook scripts

export OCTOOL_HOME="${HOME}/.octool"
export OCTOOL_DB="${OCTOOL_HOME}/octool.db"
export OCTOOL_LOG_DIR="${OCTOOL_HOME}/logs"
_OCTOOL_DATE=$(date +%Y-%m-%d)
export OCTOOL_LOG="${OCTOOL_LOG_DIR}/octool-claude-${_OCTOOL_DATE}.log"
export OCTOOL_ERROR_LOG="${OCTOOL_LOG_DIR}/errors-claude-${_OCTOOL_DATE}.log"
export OCTOOL_HOOK_LOG="${OCTOOL_LOG_DIR}/hooks-claude-${_OCTOOL_DATE}.log"
unset _OCTOOL_DATE

mkdir -p "$OCTOOL_LOG_DIR"

# Resolve binary — check ~/.octool/bin/octool first, then PATH
if [ -x "${OCTOOL_HOME}/bin/octool" ]; then
  export OCTOOL_BIN="${OCTOOL_HOME}/bin/octool"
else
  ARCH=$(uname -m)
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
  esac
  PLATFORM_BIN="${OCTOOL_HOME}/bin/octool-${OS}-${ARCH}"
  if [ -x "$PLATFORM_BIN" ]; then
    export OCTOOL_BIN="$PLATFORM_BIN"
  else
    export OCTOOL_BIN="$(which octool 2>/dev/null || echo "")"
  fi
fi

# octool_log: write to main operational log
octool_log() {
  local level="$1" source="$2" msg="$3"
  local ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  echo "[$ts] [$level] [$source] $msg" >> "$OCTOOL_LOG"
  if [ "$level" = "ERROR" ]; then
    echo "[$ts] [$level] [$source] $msg" >> "$OCTOOL_ERROR_LOG"
  fi
}

# octool_hook_log: write a detailed hook event record to hooks-claude-YYYY-MM-DD.log
octool_hook_log() {
  local hook="$1"; shift
  local ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  local fields="$*"
  printf "[%s] [HOOK] [%-20s] %s\n" "$ts" "$hook" "$fields" >> "$OCTOOL_HOOK_LOG"
}

# octool_hook_section: write a visual separator to the hook log
octool_hook_section() {
  local hook="$1"
  local ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  printf "\n[%s] ━━━ %s ━━━\n" "$ts" "$hook" >> "$OCTOOL_HOOK_LOG"
}

# claude_to_copilot_tool: translate Claude Code tool names to Copilot tool names
claude_to_copilot_tool() {
  local claude_tool="$1"
  case "$claude_tool" in
    Read|View|read_file)        echo "view" ;;
    Write|Edit|write_file)      echo "edit" ;;
    Bash|bash|execute_command)  echo "bash" ;;
    Search|Grep|search_files)   echo "grep" ;;
    Glob|ListFiles|list_files)  echo "glob" ;;
    Create|create_file)         echo "create" ;;
    *)                          echo "$(echo "$claude_tool" | tr '[:upper:]' '[:lower:]')" ;;
  esac
}

# Trap errors so they always get logged
trap 'octool_log ERROR "$(basename $0)" "script failed with exit code $?"' ERR
