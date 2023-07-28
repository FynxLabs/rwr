#!/bin/bash

# Setup logging for the script
# Includes colors and descriptive log levels

# Colors
# shellcheck disable=SC2034  # Unused variables left for readability
RED="$(tput setaf 1)"
GREEN="$(tput setaf 2)"
YELLOW="$(tput setaf 3)"
BLUE="$(tput setaf 4)"
MAGENTA="$(tput setaf 5)"
CYAN="$(tput setaf 6)"
WHITE="$(tput setaf 7)"
RESET="$(tput sgr0)"

# Log Levels
DEBUG=0
INFO=1
WARN=2
ERROR=3

# Default log level
LOG_LEVEL="$DEBUG"

# Functions
log() {
  local log_message="$1"
  local log_level="$2"

  if [ -z "$log_level" ]; then
    log_level="$DEBUG"
  fi

  if [ "$log_level" -ge "$LOG_LEVEL" ]; then
    echo -e "$(date) [$(log_level_name "$log_level")]: $log_message"
  fi
}

log_level_name() {
  local log_level="$1"

  case "$log_level" in
  0) echo "${GREEN}DEBUG${RESET}" ;;
  1) echo "INFO " ;;
  2) echo "${YELLOW}WARN${RESET}" ;;
  3) echo "${RED}ERROR${RESET}" ;;
  *) echo "UNKNOWN" ;;
  esac
}

debug() {
  log "$1" "$DEBUG"
}

info() {
  log "$1" "$INFO"
}

warn() {
  log "$1" "$WARN"
}

error() {
  log "$1" "$ERROR"
}

# Check if messages are to be logged silently
silent() {
  LOG_LEVEL="$ERROR"
}

# Enable debug mode
debug_mode() {
  LOG_LEVEL="$DEBUG"
}

# Reset logging
reset_mode() {
  LOG_LEVEL="$DEBUG"
}
