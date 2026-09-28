#!/bin/sh
set -eu

herdr plugin action list | jq -e '
  .result.actions as $actions |
  ([$actions[] | select(.plugin_id == "sunznx.herdr-new") | .action_id] | sort) ==
    ["claude", "codex", "herdr-close-codex", "tab"] and
  ([$actions[] | select(.plugin_id == "sunznx.herdr-new-codex")] | length) == 0
' >/dev/null
