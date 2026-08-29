#!/usr/bin/env bash
set -eo pipefail

extract_cwd() {
  printf '%s' "$1" | grep -o '"focused_pane_cwd":"[^"]*"' | sed -E 's/.*:"(.*)"$/\1/'
}

if [ "${1:-}" = "--self-test" ]; then
  result="$(extract_cwd '{"focused_pane_cwd":"/tmp/some dir"}')"
  [ "$result" = "/tmp/some dir" ] || { echo "self-test failed: got '$result'" >&2; exit 1; }
  echo "ok"
  exit 0
fi

cwd="$(extract_cwd "${HERDR_PLUGIN_CONTEXT_JSON:-{}}")"
trellis_dir="$cwd/.trellis"
[ -n "$cwd" ] && [ -d "$trellis_dir" ] || { echo "trellis-popup: '$trellis_dir' does not exist." >&2; exit 1; }

exec "${HERDR_BIN_PATH:-herdr}" plugin pane open \
  --plugin sunznx.trellis-popup \
  --entrypoint browser \
  --focus \
  --cwd "$trellis_dir"
