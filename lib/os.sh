#!/usr/bin/env bash

function abort() {
  echo "$1" >&2
  exit 1
}

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
export OS="macos"
export PKG="brew"
export PKG_LIST="brew list"
export PKG_SEARCH="brew search "
export PKG_INSTALL="brew install -fq "
export PKG_CLEAN="brew cleanup -q >/dev/null 2>&1"
export RUNNING_MACOS=${running_macos}
EOF
  else
    if [ -n "$(command -v eopkg)" ]; then
      cat >"${envfile}" <<EOF
export OS="solus"
export PKG="eopkg"
export PKG_LIST="eopkg li"
export PKG_SEARCH="eopkg sr "
export PKG_INSTALL="sudo eopkg it -y "
export PKG_CLEAN="sudo eopkg rmo -y >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    elif [ -n "$(command -v nala)" ]; then
      cat >"${envfile}" <<EOF
export OS="debian"
export PKG="nala"
export PKG_LIST="dpkg --get-selections"
export PKG_SEARCH="nala search "
export PKG_INSTALL="sudo nala install -y "
export PKG_CLEAN="sudo nala clean -y >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    elif [ -n "$(command -v apt)" ]; then
      cat >"${envfile}" <<EOF
export OS="debian"
export PKG="apt"
export PKG_LIST="dpkg --get-selections"
export PKG_SEARCH="apt search "
export PKG_INSTALL="sudo apt install -y "
export PKG_CLEAN="sudo apt clean >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    elif [ -n "$(command -v pacman)" ]; then
      cat >"${envfile}" <<EOF
export OS="arch"
export PKG="pacman"
export PKG_LIST="pacman -Q"
export PKG_SEARCH="pacman -Ss "
export PKG_INSTALL="sudo pacman -Sy --noconfirm "
export PKG_CLEAN="sudo pacman -Sc --noconfirm >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    elif [ -n "$(command -v dnf)" ]; then
      cat >"${envfile}" <<EOF
export OS="fedora"
export PKG="dnf"
export PKG_LIST="dnf list installed"
export PKG_SEARCH="dnf search "
export PKG_INSTALL="sudo dnf install -y "
export PKG_CLEAN="sudo dnf clean all >/dev/null 2>&1"
export RUNNING_LINUX=${running_linux}
EOF
    else
      abort "This setup only supports Linux distributions with eopkg, nala, apt, pacman, or dnf package managers."
    fi
  fi
}
