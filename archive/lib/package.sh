#!/usr/bin/env bash
# Copyright (c) 2023 "Levi Smith"
# 
# This software is released under the MIT License.
# https://opensource.org/licenses/MIT


# Load OS-specific variables
# shellcheck source=/dev/null
source /tmp/os.env

install_package() {
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
  if ! ${list_command} "${package}" >/dev/null 2>&1; then
    info ">>> Installing ${package}"
    eval "${install_command}${package}" >/dev/null 2>&1
  else
    info ">>> ${package} Already Installed"
  fi
}

package_manager() {
  local envfile="/tmp/os.env"
  source "${envfile}"

  if [ "${OS}" == 'arch' ]; then
    if [ -z "$(command -v yay)" ]; then
      info ">>> Installing YAY"
      sudo pacman --noconfirm --needed -Sy git base-devel >/dev/null 2>&1
      cd /tmp
      git clone https://aur.archlinux.org/yay.git
      cd yay
      makepkg -si
    else
      info ">>> YAY Already installed"
    fi
  fi
}
