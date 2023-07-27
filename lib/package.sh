#!/usr/bin/env bash

# Load OS-specific variables
# shellcheck source=/dev/null
source /tmp/os.env

function info() {
  echo "$1"
}

function install_package() {
  local package="$1"
  local install_command="${PKG_INSTALL}"
  local list_command="${PKG_LIST}"

  if [[ ${package} == http://* ]] || [[ ${package} == https://* ]]; then
    # The package is a URL
    local filename
    filename=$(basename "${package}")

    if [ ! -f "/tmp/${filename}" ]; then
      info ">>> Downloading ${package}"
      wget -P /tmp "${package}" >/dev/null 2>&1
    else
      info ">>> ${filename} already downloaded"
    fi

    package="/tmp/${filename}"
  fi

  info ">>> Checking ${package}"
  if ! ${list_command} "${package}" >/dev/null 2>&1 ; then
    info ">>> Installing ${package}"
    eval "${install_command}${package}" >/dev/null 2>&1
  else
    info ">>> ${package} Already Installed"
  fi
}
