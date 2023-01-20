#!/usr/bin/env bash

function setos() {

  local envfile="/tmp/os.env"
  local current_os
  current_os="$(uname)"

  if [[ "${current_os}" == "Linux" ]]; then
    local running_linux=1
  elif [[ "${current_os}" == "Darwin" ]]; then
    local running_macos=1
  else
    abort "This setup only supports macOS and Linux."
  fi

  if [[ -n "${running_macos-}" ]]; then
    cat >"${envfile}" <<EOF
export OS="solus"
export PKG="brew"
export PKG_INSTALL="brew install -fq "
export PKG_CLEAN="brew cleanup -q >/dev/null 2>&1"
export RUNNING_MACOS=${running_macos}
EOF
  else
    if [ -n "$(command -v eopkg)" ]; then
      cat >"${envfile}" <<EOF
export OS="solus"
export PKG="eopkg"
export PKG_INSTALL="sudo eopkg it -y "
export PKG_CLEAN="sudo eopkg rmo -y >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    elif [ -n "$(command -v apt)" ]; then
      cat >"${envfile}" <<EOF
export OS="debian"
export PKG="apt"
export PKG_INSTALL="sudo apt install -y "
export PKG_CLEAN="sudo apt-get clean -y >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    elif [ -n "$(command -v pacman)" ]; then
      cat >"${envfile}" <<EOF
export OS="arch"
export PKG="pacman"
export PKG_INSTALL="sudo pacman -Sy --noconfirm "
export PKG_CLEAN="sudo pacman -Sc --noconfirm >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    elif [ -n "$(command -v dnf)" ]; then
      cat >"${envfile}" <<EOF
export OS="fedora"
export PKG="dnf"
export PKG_INSTALL="sudo dnf install -y "
export PKG_CLEAN="sudo eopkg rmo -y >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    fi
  fi
}
