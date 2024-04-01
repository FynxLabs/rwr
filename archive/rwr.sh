#!/usr/bin/env bash
# Copyright (c) 2023 "Levi Smith"
# 
# This software is released under the MIT License.
# https://opensource.org/licenses/MIT


# Exit on error. Append "|| true" if you expect an error.
set -o errexit
# Exit on error inside any functions or subshells.
set -o errtrace
# Do not allow use of undefined vars. Use ${VAR:-} to use an undefined VAR
set -o nounset
# Catch the error in case mysqldump fails (but gzip succeeds) in `mysqldump |gzip`
set -o pipefail

trap cleanup SIGINT SIGTERM ERR EXIT

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)
export SCRIPT_DIR

# Include the logging library
# shellcheck source=/dev/null
source "${SCRIPT_DIR}/lib/logging.sh"

usage() {
  cat <<EOF
Usage: $(basename "${BASH_SOURCE[0]}") [-h] [-v] [-f] -p param_value arg1 [arg2...]

Script description here.

Available options:

-h, --help      Print this help and exit
-v, --verbose   Print script debug info
-f, --flag      Some flag description
-p, --param     Some param description
EOF
  exit
}

cleanup() {
  trap - SIGINT SIGTERM ERR EXIT
  # script cleanup here
}

parse_params() {
  # default values of variables set from params
  flag=0
  param=''

  while :; do
    case "${1-}" in
    -h | --help) usage ;;
    -v | --verbose) set -x ;;
    -f | --flag) flag=1 ;; # example flag
    -p | --param)          # example named parameter
      param="${2-}"
      shift
      ;;
    -?*) die "Unknown Option: $1" ;;
    *) break ;;
    esac
    shift
  done

  args=("$@")

  # check required params and arguments
  [[ -z "${param-}" ]] && die "Missing required parameter: param"
  [[ ${#args[@]} -eq 0 ]] && die "Missing script arguments"

  return 0
}

parse_params "$@"

# Parse YAML file
declare -a depends=($(yq r config.yaml 'depends.*'))
declare -a packages=($(yq r config.yaml 'package.install.*'))

# Check for dependencies
for dep in "${depends[@]}"; do
  until grep -q "$dep" status_file; do
    echo "$dep has not finished running. Waiting..."
    sleep 10
  done
  echo "$dep has finished running."
done

# Install packages
for pkg in "${packages[@]}"; do
  package_install "$pkg" # Assuming package_install is the function from lib/package.sh
done