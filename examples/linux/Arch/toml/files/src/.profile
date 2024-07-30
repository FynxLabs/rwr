#!/bin/sh

# if running bash
if [ -n "$BASH_VERSION" ]; then
  # include .bashrc if it exists
  if [ -f "{{ .User.home }}/.bashrc" ]; then
    # shellcheck disable=SC1091
    . "{{ .User.home }}/.bashrc"
  fi
fi