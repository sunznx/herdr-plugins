#!/usr/bin/env bash
set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
herdr_bin="${HERDR_BIN_PATH:-herdr}"

if [ "${1:-}" = "--self-test" ]; then
  exec "$script_dir/test.sh" tab
fi

fail() {
  printf '\nherdr-move: %s\n' "$*" >&2
  if [ -t 0 ]; then
    printf 'Press any key to close…' >&2
    read -r -n1 _ 2>/dev/null || true
  fi
  exit 1
}

command -v jq >/dev/null 2>&1 || fail "jq is not installed or not on PATH."

pane_id="${HERDR_MOVE_PANE_ID:-}"
source_tab="${HERDR_MOVE_TAB_ID:-}"
[ -n "$pane_id" ] || fail "source pane ID is missing."
[ -n "$source_tab" ] || fail "source tab ID is missing."

pane_json="$("$herdr_bin" pane get "$pane_id" 2>/dev/null || true)"
live_pane_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
live_tab_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
[ "$live_pane_id" = "$pane_id" ] || fail "the source pane is no longer available."
[ "$live_tab_id" = "$source_tab" ] || fail "the source pane is no longer in its original tab."

workspace_json="$("$herdr_bin" workspace list 2>/dev/null || true)"
tab_json="$("$herdr_bin" tab list 2>/dev/null || true)"
candidates="$(
  jq -nr \
    --argjson workspaces "$workspace_json" \
    --argjson tabs "$tab_json" \
    --arg source "$source_tab" '
      ($workspaces.result.workspaces // [] | map({key: .workspace_id, value: (.label // .workspace_id)}) | from_entries) as $labels
      | $tabs.result.tabs[]?
      | select(.tab_id != $source)
      | [
          .tab_id,
          (($labels[.workspace_id] // .workspace_id) + " / #" + (.number | tostring) + " " + (.label // .tab_id))
        ]
      | @tsv
    ' 2>/dev/null
)"
[ -n "$candidates" ] || fail "no destination tab is available."

if [ -n "${HERDR_MOVE_TAB_CHOICE:-}" ]; then
  destination="${HERDR_MOVE_TAB_CHOICE}"
else
  command -v fzf >/dev/null 2>&1 || fail "fzf is not installed or not on PATH."
  selected="$(
    printf '%s\n' "$candidates" \
      | fzf --delimiter=$'\t' --with-nth=2 --prompt='move pane to tab ▸ ' --reverse --cycle --no-multi --tiebreak=begin,index
  )"
  fzf_status=$?
  case "$fzf_status" in
    0) ;;
    1|130) exit 0 ;;
    *) fail "fzf exited with status $fzf_status." ;;
  esac
  [ -n "$selected" ] || exit 0
  destination="${selected%%$'\t'*}"
fi

printf '%s\n' "$candidates" | awk -F $'\t' -v id="$destination" '$1 == id { found = 1 } END { exit !found }' \
  || fail "destination tab '$destination' is unavailable."

target_json="$("$herdr_bin" tab get "$destination" 2>/dev/null || true)"
target_tab_id="$(printf '%s' "$target_json" | jq -r '.result.tab.tab_id // empty' 2>/dev/null)"
[ "$target_tab_id" = "$destination" ] || fail "destination tab '$destination' is no longer available."

pane_json="$("$herdr_bin" pane get "$pane_id" 2>/dev/null || true)"
live_pane_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
live_tab_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
[ "$live_pane_id" = "$pane_id" ] || fail "the source pane is no longer available."
[ "$live_tab_id" = "$source_tab" ] || fail "the source pane is no longer in its original tab."

move_json="$("$herdr_bin" pane move "$pane_id" --tab "$destination" --split right --focus 2>&1)"
move_status=$?
[ "$move_status" -eq 0 ] || fail "pane move failed (exit $move_status): $move_json"

new_pane_id="$(printf '%s' "$move_json" | jq -r '.result.move_result.pane.pane_id // empty' 2>/dev/null)"
[ -n "$new_pane_id" ] || fail "Herdr did not return the moved pane ID."

moved_json="$("$herdr_bin" pane get "$new_pane_id" 2>/dev/null || true)"
moved_tab_id="$(printf '%s' "$moved_json" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
[ "$moved_tab_id" = "$destination" ] || fail "the moved pane could not be verified in destination tab '$destination'."

printf 'Moved pane to %s (%s)\n' "$destination" "$new_pane_id"
