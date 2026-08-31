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
mode="${HERDR_RENAME_MODE:-pane-agent}"

case "$mode" in
  pane-agent|tab|agent) ;;
  *) fail "unknown rename mode '$mode'." ;;
esac

prompt_target="$mode"
[ "$mode" = "pane-agent" ] && prompt_target="pane and agent"

pane_json="$("$herdr_bin" pane get "$pane_id" 2>/dev/null || true)"
live_pane_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
[ "$live_pane_id" = "$pane_id" ] || fail "the source pane is no longer available."

tab_id="${HERDR_RENAME_TAB_ID:-}"
if [ "$mode" = "tab" ]; then
  live_tab_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
  [ -n "$tab_id" ] && [ "$live_tab_id" = "$tab_id" ] \
    || fail "the source tab is no longer available."
fi

if [ -n "${HERDR_RENAME_NAME+x}" ]; then
  new_name="$HERDR_RENAME_NAME"
else
  command -v fzf >/dev/null 2>&1 || fail "fzf is not installed or not on PATH."
  selected="$(
    printf '\n' \
      | fzf --print-query --phony --no-info --no-separator --no-multi \
        --prompt="rename ${prompt_target} ▸ "
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

if [ "$mode" = "tab" ]; then
  live_tab_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
  [ "$live_tab_id" = "$tab_id" ] || fail "the source tab is no longer available."
  tab_json="$("$herdr_bin" tab get "$tab_id" 2>/dev/null || true)"
  [ "$(printf '%s' "$tab_json" | jq -r '.result.tab.tab_id // empty' 2>/dev/null)" = "$tab_id" ] \
    || fail "the source tab is no longer available."
  tab_result="$("$herdr_bin" tab rename "$tab_id" "$new_name" 2>&1)"
  tab_status=$?
  [ "$tab_status" -eq 0 ] \
    || fail "tab rename failed (exit $tab_status): $tab_result"
  printf 'Renamed tab to %s\n' "$new_name"
  exit 0
fi

agent_json="$("$herdr_bin" agent list 2>/dev/null || true)"
printf '%s' "$agent_json" | jq -e '.result.agents | arrays' >/dev/null 2>&1 \
  || fail "could not list Herdr agents."

has_agent="$({
  printf '%s' "$agent_json" | jq -r --arg pane "$pane_id" \
    'any(.result.agents[]?; .pane_id == $pane)'
} 2>/dev/null)"
[ "$has_agent" = "true" ] || has_agent="false"

if [ "$mode" = "agent" ] && [ "$has_agent" != "true" ]; then
  fail "the source pane does not have a detected agent."
fi

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

if [ "$mode" = "agent" ]; then
  printf 'Renamed agent to %s\n' "$new_name"
  exit 0
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
