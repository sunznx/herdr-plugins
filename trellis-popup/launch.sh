#!/usr/bin/env bash
set -eo pipefail

client_id=$$

# Keep the user's normal Yazi configuration and override only this session's
# sort order after the instance is ready to receive actions.
(
  for _ in {1..50}; do
    if ya emit-to "$client_id" sort natural --reverse=yes >/dev/null 2>&1; then
      exit 0
    fi
    sleep 0.05
  done
) &

exec yazi --client-id "$client_id" .
