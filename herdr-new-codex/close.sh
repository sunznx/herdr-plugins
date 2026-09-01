#!/usr/bin/env bash
set -uo pipefail

herdr_bin="${HERDR_BIN_PATH:-herdr}"

fail() {
  printf 'herdr-new-codex: %s\n' "$*" >&2
  exit 1
}

command -v jq >/dev/null 2>&1 || fail "jq is not installed or not on PATH."

candidate="${LIVE_PANE_ID:-}"
if [ -z "$candidate" ]; then
  ctx="${HERDR_PLUGIN_CONTEXT_JSON:-}"
  [ -n "$ctx" ] || ctx='{}'
  candidate="$(printf '%s' "$ctx" | jq -r '.focused_pane_id // empty' 2>/dev/null)"
fi

pane_json=""
if [ -n "$candidate" ]; then
  pane_json="$("$herdr_bin" pane get "$candidate" 2>/dev/null || true)"
fi
if [ -z "$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)" ]; then
  pane_json="$("$herdr_bin" pane current --current 2>/dev/null || true)"
fi

pane_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.pane_id // empty')"
tab_id="$(printf '%s' "$pane_json" | jq -r '.result.pane.tab_id // empty')"
agent="$(printf '%s' "$pane_json" | jq -r '.result.pane.agent // empty')"
[ -n "$pane_id" ] && [ -n "$tab_id" ] || fail "could not resolve the current pane."
[ "$agent" = "codex" ] || fail "the current pane is not running Codex."

"$herdr_bin" agent prompt "$pane_id" /archive >/dev/null \
  || fail "could not send /archive to Codex."

confirmed=false
for _ in {1..100}; do
  pane_json="$("$herdr_bin" pane get "$pane_id" 2>/dev/null || true)"
  agent="$(printf '%s' "$pane_json" | jq -r '.result.pane.agent // empty' 2>/dev/null)"
  [ "$agent" = "codex" ] || exec "$herdr_bin" tab close "$tab_id"
  status="$(printf '%s' "$pane_json" | jq -r '.result.pane.agent_status // empty' 2>/dev/null)"
  if [ "$status" = "blocked" ] && [ "$confirmed" = false ]; then
    "$herdr_bin" agent send-keys "$pane_id" down enter >/dev/null \
      || fail "could not confirm /archive."
    confirmed=true
  fi
  sleep 0.1
done

fail "Codex did not finish /archive within 10 seconds; tab left open."
