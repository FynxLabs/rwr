#!/usr/bin/env bash
# Copyright (c) 2023 "Levi Smith"
# 
# This software is released under the MIT License.
# https://opensource.org/licenses/MIT


# Exit on error. Append "|| true" if you expect an error.
set -o errexit
# Do not allow use of undefined vars. Use ${VAR:-} to use an undefined VAR
set -o nounset
# Catch the error in case mysqldump fails (but gzip succeeds) in `mysqldump |gzip`
set -o pipefail

template() {
  local path="$1"
  local data="$2"
  local filename="$3"

  cat >"${path}/${filename}.2" <<EOF
${data}
EOF

  if [ -f "${path}/${filename}" ]; then
    echo ">>> ${filename/./}: File detected - Looking for changes"
    if [ -n "$(diff -y --suppress-common-lines "${path}/${filename}" "${path}/${filename}.2")" ]; then
      echo ">>> ${filename/./}: Changes detected, printing side by side diff"
      diff -y --suppress-common-lines "${path}/${filename}" "${path}/${filename}.2" || true
      mv "${path}/${filename}.2" "${path}/${filename}"
    else
      echo ">>> ${filename/./}: No changes detected"
      rm "${path}/${filename}.2"
    fi
  else
    echo ">>> ${filename/./}: No file detected, creating new file"
    mv "${path}/${filename}.2" "${path}/${filename}"
  fi
}
