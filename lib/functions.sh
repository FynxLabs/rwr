#!/usr/bin/env bash

cleanup() {
  trap - SIGINT SIGTERM ERR EXIT
  # script cleanup here
}