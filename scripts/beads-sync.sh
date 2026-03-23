#!/usr/bin/env bash

# ┏┓ ┏━╸┏━┓╺┳┓┏━┓   ┏━┓╻ ╻┏┓╻┏━╸
# ┣┻┓┣╸ ┣━┫ ┃┃┗━┓   ┗━┓┗┳┛┃┗┫┃
# ┗━┛┗━╸╹ ╹╺┻┛┗━┛   ┗━┛ ╹ ╹ ╹┗━╸
# Pre-commit hook for beads (br), a local-first CLI issue tracker designed for
# AI coding agents. Flushes state to beads.jsonl and stages it.

set -euo pipefail

# Gracefully exit if `br` (beads CLI issue tracker) is not installed.
command -v br > /dev/null 2>&1 || exit 0

BEADS_JSONL=./beads.jsonl br sync \
  --flush-only \
  --allow-external-jsonl \
  --force \
  --quiet

git add beads.jsonl 2> /dev/null
