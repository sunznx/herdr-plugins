#!/usr/bin/env bash
# Action `sunznx.command-palette-popup.open`: open the unified fzf command palette.
#
# This runs on the herdr server (no TTY), so it can't run fzf directly. It opens
# the `palette` popup pane (see herdr-plugin.toml — a small modal centered over
# the active pane), which gets a real terminal and runs palette.sh. Placement
# and size are declared once in the manifest, not passed here, so there's a
# single source of truth for how the popup looks.
#
# Capture the origin pane/tab/workspace before opening the popup so native
# commands target the pane where the palette was requested.
set -uo pipefail

herdr_bin="${HERDR_BIN_PATH:-herdr}"
ctx="${HERDR_PLUGIN_CONTEXT_JSON:-}"

pane="" tab="" workspace="" cwd=""
if [ -n "$ctx" ] && command -v jq >/dev/null 2>&1; then
  pane="$(printf '%s' "$ctx" | jq -r '.focused_pane_id // empty' 2>/dev/null)"
  tab="$(printf '%s' "$ctx" | jq -r '.tab_id // empty' 2>/dev/null)"
  workspace="$(printf '%s' "$ctx" | jq -r '.workspace_id // empty' 2>/dev/null)"
  cwd="$(printf '%s' "$ctx" | jq -r '.focused_pane_cwd // .workspace_cwd // empty' 2>/dev/null)"
fi
[ -n "$cwd" ] || cwd="${HERDR_WORKSPACE_CWD:-}"

ctx_json="$(
  jq -nc --arg pane "$pane" --arg tab "$tab" --arg workspace "$workspace" --arg cwd "$cwd" \
    '{pane: $pane, tab: $tab, workspace: $workspace, cwd: $cwd}' 2>/dev/null
)"
[ -n "$ctx_json" ] || ctx_json='{}'

set -- plugin pane open \
  --plugin sunznx.command-palette-popup \
  --entrypoint palette \
  --focus \
  --env "CPP_CONTEXT_JSON=$ctx_json"

# Only forward --cwd when it's a real directory; otherwise the popup falls
# back to the plugin root (irrelevant here since palette.sh only ever reads
# $HERDR_PLUGIN_ROOT and $CPP_CONTEXT_JSON, never its own cwd).
if [ -n "$cwd" ] && [ -d "$cwd" ]; then
  set -- "$@" --cwd "$cwd"
fi

exec "$herdr_bin" "$@"
