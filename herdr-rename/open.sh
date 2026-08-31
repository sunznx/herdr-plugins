#!/usr/bin/env bash
set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
herdr_bin="${HERDR_BIN_PATH:-herdr}"
plugin_id="${HERDR_PLUGIN_ID:-sunznx.herdr-rename}"
mode="${1:-pane-agent}"

if [ "${1:-}" = "--self-test" ]; then
  exec "$script_dir/test.sh" open
fi

fail() {
  printf 'herdr-rename: %s\n' "$*" >&2
  exit 1
}

command -v jq >/dev/null 2>&1 || fail "jq is not installed or not on PATH."

case "$mode" in
  pane-agent|tab|agent) ;;
  *) fail "unknown rename mode '$mode'." ;;
esac

pane_json=""
candidate="${LIVE_PANE_ID:-}"

if [ -n "$candidate" ]; then
  pane_json="$("$herdr_bin" pane get "$candidate" 2>/dev/null || true)"
fi

if [ -z "$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)" ]; then
  pane_json="$("$herdr_bin" pane current --current 2>/dev/null || true)"
fi

pane_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
[ -n "$pane_id" ] || fail "could not resolve the triggering pane."
tab_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
if [ "$mode" = "tab" ]; then
  [ -n "$tab_id" ] || fail "could not resolve the triggering tab."
fi

exec "$herdr_bin" plugin pane open \
  --plugin "$plugin_id" \
  --entrypoint picker \
  --placement popup \
  --focus \
  --env "HERDR_RENAME_MODE=$mode" \
  --env "HERDR_RENAME_PANE_ID=$pane_id" \
  --env "HERDR_RENAME_TAB_ID=$tab_id"
