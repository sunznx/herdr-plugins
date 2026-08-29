#!/usr/bin/env bash
set -eo pipefail

extract_cwd() {
  printf '%s' "$1" | grep -o '"focused_pane_cwd":"[^"]*"' | sed -E 's/.*:"(.*)"$/\1/'
}

if [ "${1:-}" = "--self-test" ]; then
  result=$(extract_cwd '{"focused_pane_id":"p1","focused_pane_cwd":"/tmp/some dir"}')
  [ "$result" = "/tmp/some dir" ] || { echo "self-test failed: got '$result'" >&2; exit 1; }
  echo "ok"
  exit 0
fi

mode="${1:-pick}"
plugin_id="${HERDR_PLUGIN_ID:-sunznx.yazi-popup}"
cwd=$(extract_cwd "${HERDR_PLUGIN_CONTEXT_JSON:-{}}")

case "$mode" in
  pick)
    [ -n "${HERDR_PANE_ID:-}" ] || exit 0
    args=(plugin pane open --plugin "$plugin_id" --entrypoint picker --env "HERDR_TARGET_PANE_ID=$HERDR_PANE_ID")
    [ -n "$cwd" ] && args+=(--cwd "$cwd")
    ;;
  fzf|rg)
    [ -n "$cwd" ] && [ -d "$cwd" ] || { echo "yazi-popup: current pane directory is unavailable." >&2; exit 1; }
    args=(plugin pane open --plugin "$plugin_id" --entrypoint "$mode" --focus --cwd "$cwd")
    ;;
  trellis)
    trellis_dir="$cwd/.trellis"
    [ -n "$cwd" ] && [ -d "$trellis_dir" ] || { echo "yazi-popup: '$trellis_dir' does not exist." >&2; exit 1; }
    args=(plugin pane open --plugin "$plugin_id" --entrypoint trellis --focus --cwd "$trellis_dir")
    ;;
  *)
    echo "yazi-popup: unsupported mode '$mode'." >&2
    exit 1
    ;;
esac

herdr_bin="${HERDR_BIN_PATH:-herdr}"
attempt=0
while true; do
  if output="$("$herdr_bin" "${args[@]}" 2>&1)"; then
    [ -z "$output" ] || printf '%s\n' "$output"
    exit 0
  else
    status=$?
  fi

  attempt=$((attempt + 1))
  if [[ "$output" == *"popup already open"* ]] && [ "$attempt" -lt 20 ]; then
    sleep 0.05
    continue
  fi

  printf '%s\n' "$output" >&2
  exit "$status"
done
