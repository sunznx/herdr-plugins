# herdr-move

Move the pane that triggered the action into a new tab in an existing Herdr workspace. Any shell or coding agent running in the pane moves with it; the process is not restarted.

The plugin resolves the live pane ID before opening its picker. This avoids passing a stale workspace-qualified pane ID into a popup, where the source pane's inherited caller context is no longer available.

## Requirements

- Herdr ≥ 0.8.0
- `fzf`
- `jq`

## Install

```bash
herdr plugin install sunznx/herdr-plugins/herdr-move
```

The action is available as `sunznx.herdr-move.open`. It can be selected from a compatible command palette or bound directly:

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "sunznx.herdr-move.open"
description = "Move pane to workspace"
```

When a shortcut invokes the action without `LIVE_PANE_ID`, the plugin resolves the triggering pane with `herdr pane current --current`. It never falls back to a pane focused by another client.

## Development

```bash
bash -n open.sh move.sh test.sh
./open.sh --self-test
./move.sh --self-test
```
