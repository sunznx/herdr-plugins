#!/usr/bin/env bash
set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
herdr_bin="${HERDR_BIN_PATH:-herdr}"

if [ "${1:-}" = "--self-test" ]; then
  exec "$script_dir/test.sh" rename
fi

fail() {
  printf '\nherdr-rename: %s\n' "$*" >&2
  if [ -t 0 ]; then
    printf 'Press any key to close…' >&2
    read -r -n1 _ 2>/dev/null || true
  fi
  exit 1
}

command -v jq >/dev/null 2>&1 || fail "jq is not installed or not on PATH."

pane_id="${HERDR_RENAME_PANE_ID:-}"
[ -n "$pane_id" ] || fail "source pane ID is missing."

pane_json="$("$herdr_bin" pane get "$pane_id" 2>/dev/null || true)"
live_pane_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
[ "$live_pane_id" = "$pane_id" ] || fail "the source pane is no longer available."

if [ -n "${HERDR_RENAME_NAME+x}" ]; then
  new_name="$HERDR_RENAME_NAME"
else
  command -v fzf >/dev/null 2>&1 || fail "fzf is not installed or not on PATH."
  selected="$(
    printf '\n' \
      | fzf --print-query --phony --no-info --no-separator --no-multi \
        --prompt='rename pane and agent ▸ '
  )"
  fzf_status=$?
  case "$fzf_status" in
    0) ;;
    1|130) exit 0 ;;
    *) fail "fzf exited with status $fzf_status." ;;
  esac
  new_name="${selected%%$'\n'*}"
fi

[ -n "$new_name" ] || exit 0
if printf '%s' "$new_name" | grep -q '[[:cntrl:]]'; then
  fail "the name must not contain control characters."
fi

pane_json="$("$herdr_bin" pane get "$pane_id" 2>/dev/null || true)"
live_pane_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
[ "$live_pane_id" = "$pane_id" ] || fail "the source pane is no longer available."

agent_json="$("$herdr_bin" agent list 2>/dev/null || true)"
printf '%s' "$agent_json" | jq -e '.result.agents | arrays' >/dev/null 2>&1 \
  || fail "could not list Herdr agents."

has_agent="$({
  printf '%s' "$agent_json" | jq -r --arg pane "$pane_id" \
    'any(.result.agents[]?; .pane_id == $pane)'
} 2>/dev/null)"
[ "$has_agent" = "true" ] || has_agent="false"

if [ "$has_agent" = "true" ]; then
  [[ "$new_name" =~ ^[a-z][a-z0-9_-]{0,31}$ ]] \
    || fail "agent names must match [a-z][a-z0-9_-]{0,31}."
fi

# Agent rename is stricter, so run it first. A conflict or validation error then
# leaves the pane label untouched. Pane rename only needs a live pane and a label.
if [ "$has_agent" = "true" ]; then
  agent_result="$("$herdr_bin" agent rename "$pane_id" "$new_name" 2>&1)"
  agent_status=$?
  [ "$agent_status" -eq 0 ] \
    || fail "agent rename failed (exit $agent_status): $agent_result"
fi

pane_result="$("$herdr_bin" pane rename "$pane_id" "$new_name" 2>&1)"
pane_status=$?
[ "$pane_status" -eq 0 ] \
  || fail "pane rename failed (exit $pane_status): $pane_result"

if [ "$has_agent" = "true" ]; then
  printf 'Renamed pane and agent to %s\n' "$new_name"
else
  printf 'Renamed pane to %s\n' "$new_name"
fi
