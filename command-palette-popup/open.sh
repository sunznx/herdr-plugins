#!/usr/bin/env bash
# Action `sunznx.command-palette-popup.open`: open the unified fzf command palette.
#
# This runs on the herdr server (no TTY), so it can't run fzf directly. It opens
# the `palette` popup pane (see herdr-plugin.toml — a small modal centered over
# the active pane), which gets a real terminal and runs palette.sh. Pass the
# placement explicitly as well as declaring it in the manifest: a locally
# linked plugin can keep an older registry snapshot until it is linked again.
#
# Capture the origin pane/tab/workspace before opening the popup as a fallback
# for native commands. palette.sh uses the popup's own invocation context as
# the authoritative origin when moving a pane.
set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
herdr_bin="${HERDR_BIN_PATH:-herdr}"
ctx="${HERDR_PLUGIN_CONTEXT_JSON:-}"

if [ "${1:-}" = "--self-test" ]; then
  exec "$script_dir/test.sh" open
fi

command -v jq >/dev/null 2>&1 || { printf 'command-palette-popup: jq is not installed or not on PATH.\n' >&2; exit 1; }

pane="" tab="" workspace="" cwd=""
if [ -n "$ctx" ]; then
  pane="$(printf '%s' "$ctx" | jq -r '.focused_pane_id // empty' 2>/dev/null)"
  tab="$(printf '%s' "$ctx" | jq -r '.tab_id // empty' 2>/dev/null)"
  workspace="$(printf '%s' "$ctx" | jq -r '.workspace_id // empty' 2>/dev/null)"
  cwd="$(printf '%s' "$ctx" | jq -r '.focused_pane_cwd // .workspace_cwd // empty' 2>/dev/null)"
fi
[ -n "$cwd" ] || cwd="${HERDR_WORKSPACE_CWD:-}"

# Prefer an explicitly forwarded live pane, then the pane captured when the
# action was invoked. `pane current --current` can describe the action runner
# rather than the pane that opened the palette, so use it only as the fallback
# while the inherited caller context is still available.
live_json=""
live_pane=""
candidate="${LIVE_PANE_ID:-}"
if [ -n "$candidate" ]; then
  live_json="$("$herdr_bin" pane get "$candidate" 2>/dev/null || true)"
  live_pane="$(printf '%s' "$live_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
fi
if [ -z "$live_pane" ] && [ -n "$pane" ] && [ "$pane" != "$candidate" ]; then
  live_json="$("$herdr_bin" pane get "$pane" 2>/dev/null || true)"
  live_pane="$(printf '%s' "$live_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
fi
if [ -z "$live_pane" ]; then
  live_json="$("$herdr_bin" pane current --current 2>/dev/null || true)"
  live_pane="$(printf '%s' "$live_json" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
fi

if [ -n "$live_pane" ]; then
  pane="$live_pane"
  live_tab="$(printf '%s' "$live_json" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
  live_workspace="$(printf '%s' "$live_json" | jq -r '.result.pane.workspace_id // empty' 2>/dev/null)"
  [ -n "$live_tab" ] && tab="$live_tab"
  [ -n "$live_workspace" ] && workspace="$live_workspace"
else
  pane=""
fi

ctx_json="$(
  jq -nc --arg pane "$pane" --arg tab "$tab" --arg workspace "$workspace" --arg cwd "$cwd" \
    '{pane: $pane, tab: $tab, workspace: $workspace, cwd: $cwd}' 2>/dev/null
)"
[ -n "$ctx_json" ] || ctx_json='{}'

set -- plugin pane open \
  --plugin sunznx.command-palette-popup \
  --entrypoint palette \
  --placement popup \
  --focus \
  --env "CPP_CONTEXT_JSON=$ctx_json"

# Only forward --cwd when it's a real directory; otherwise the popup falls
# back to the plugin root (irrelevant here since palette.sh only ever reads
# $HERDR_PLUGIN_ROOT and $CPP_CONTEXT_JSON, never its own cwd).
if [ -n "$cwd" ] && [ -d "$cwd" ]; then
  set -- "$@" --cwd "$cwd"
fi

exec "$herdr_bin" "$@"
