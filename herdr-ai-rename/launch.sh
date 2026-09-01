#!/usr/bin/env bash
set -uo pipefail

mode="${1:-current}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "$mode" in current|all) ;; *) exit 2 ;; esac

/usr/bin/nohup "$script_dir/rename.sh" "$mode" \
  </dev/null >/dev/null 2>&1 &

printf 'Started AI rename in the background.\n'
