#!/usr/bin/env bash
set -eo pipefail

client_id=$$

# Wait until Yazi can receive actions, then enter its built-in fzf plugin.
(
  for _ in {1..50}; do
    if ya emit-to "$client_id" plugin fzf >/dev/null 2>&1; then
      exit 0
    fi
    sleep 0.05
  done
) &

exec yazi --client-id "$client_id" .
