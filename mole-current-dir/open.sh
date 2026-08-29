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
[ -n "$cwd" ] && [ -d "$cwd" ] || { echo "mole-current-dir: current pane directory is unavailable." >&2; exit 1; }

exec "${HERDR_BIN_PATH:-herdr}" plugin pane open \
  --plugin sunznx.mole-current-dir \
  --entrypoint analyze \
  --placement split \
  --direction right \
  --focus \
  --cwd "$cwd"
