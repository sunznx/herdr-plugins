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

app="${1:-}"
cwd="$(extract_cwd "${HERDR_PLUGIN_CONTEXT_JSON:-{}}")"
[ -n "$app" ] || { echo "open-in: app name is required." >&2; exit 1; }
[ -n "$cwd" ] && [ -d "$cwd" ] || { echo "open-in: current pane directory is unavailable." >&2; exit 1; }

exec open -a "$app" "$cwd"
