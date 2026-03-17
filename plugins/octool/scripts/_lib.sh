#!/bin/bash
# scripts/_lib.sh — sourced by all hook scripts

export OCTOOL_HOME="${HOME}/.octool"
export OCTOOL_DB="${OCTOOL_HOME}/octool.db"
export OCTOOL_LOG_DIR="${OCTOOL_HOME}/logs"
export OCTOOL_LOG="${OCTOOL_LOG_DIR}/octool-$(date +%Y-%m-%d).log"
export OCTOOL_ERROR_LOG="${OCTOOL_LOG_DIR}/errors-$(date +%Y-%m-%d).log"
export OCTOOL_HOOK_LOG="${OCTOOL_LOG_DIR}/hooks-$(date +%Y-%m-%d).log"

mkdir -p "$OCTOOL_LOG_DIR"

# Resolve binary
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac
export OCTOOL_BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/bin/octool-${OS}-${ARCH}"

# If binary doesn't exist, try fallbacks in order
if [ ! -x "$OCTOOL_BIN" ]; then
  OCTOOL_BIN="${HOME}/.octool/bin/octool"
fi
if [ ! -x "$OCTOOL_BIN" ]; then
  OCTOOL_BIN="$(which octool 2>/dev/null || echo "")"
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

# octool_hook_log: write a detailed hook event record to hooks-YYYY-MM-DD.log
# Usage: octool_hook_log HOOK_NAME key=value key=value ...
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

# Trap errors so they always get logged
trap 'octool_log ERROR "$(basename $0)" "script failed with exit code $?"' ERR
