#!/usr/bin/env bash
set -eo pipefail

if [ -z "${FZF_DEFAULT_OPTS_FILE:-}" ]; then
  fzf_opts_file="${XDG_CONFIG_HOME:-$HOME/.config}/fzf/fzfrc"
  [ ! -f "$fzf_opts_file" ] || export FZF_DEFAULT_OPTS_FILE="$fzf_opts_file"
fi

case "${1:-}" in
  rg)
    action=(plugin fr rg)
    ;;
  trellis)
    action=(sort natural --reverse=yes)
    ;;
  *)
    echo "yazi-popup: unsupported browser mode '${1:-}'." >&2
    exit 1
    ;;
esac

client_id=$$
(
  for _ in {1..50}; do
    if ya emit-to "$client_id" "${action[@]}" >/dev/null 2>&1; then
      exit 0
    fi
    sleep 0.05
  done
) &

exec yazi --client-id "$client_id" .
