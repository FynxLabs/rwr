#!/usr/bin/env bash
# Copyright (c) 2023 "Levi Smith"
# 
# This software is released under the MIT License.
# https://opensource.org/licenses/MIT


cleanup() {
  trap - SIGINT SIGTERM ERR EXIT
  # script cleanup here
}