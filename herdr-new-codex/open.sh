#!/usr/bin/env bash
set -euo pipefail

herdr_bin="${HERDR_BIN_PATH:-herdr}"
plugin_id="${HERDR_PLUGIN_ID:-sunznx.herdr-new-codex}"

exec "$herdr_bin" plugin pane open \
  --plugin "$plugin_id" \
  --entrypoint picker \
  --focus
