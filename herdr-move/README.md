# herdr-move

Move the pane that triggered the action into an existing workspace's new tab, or into the root tab of a newly created workspace. Any shell or coding agent running in the pane moves with it; the process is not restarted.

The picker accepts a new workspace name typed into the search prompt. If that name does not exist, it creates the workspace with the source pane's directory. After moving to a different workspace path, ordinary panes run `cd <target-path>`, while Codex panes receive `/cd <target-path>` on the moved pane.

When a workspace label and a new directory share the same typed basename, the exact directory candidate wins; selecting the full workspace label or root path still switches to the existing workspace.

The plugin resolves the live pane ID before opening its picker. This avoids passing a stale workspace-qualified pane ID into a popup, where the source pane's inherited caller context is no longer available.

## Requirements

- Herdr ≥ 0.8.0
- `fzf`
- `git` and `zoxide` for directory-backed workspace candidates
- Go 1.24+（仅安装或本地构建时需要）

## Install

```bash
herdr plugin install sunznx/herdr-plugins/herdr-move
```

The actions are:

- `sunznx.herdr-move.open` — move the pane into a new tab in an existing workspace, or reuse the root tab when creating the selected workspace.
- `sunznx.herdr-move.tab` — move the pane into an existing tab, split to the right.

They can be selected from a compatible command palette or bound directly:

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "sunznx.herdr-move.open"
description = "Move pane to workspace"

[[keys.command]]
key = "prefix+shift+m"
type = "plugin_action"
command = "sunznx.herdr-move.tab"
description = "Move pane to tab"
```

When a shortcut invokes the action without `LIVE_PANE_ID`, the plugin resolves the triggering pane with `herdr pane current --current`. It never falls back to a pane focused by another client.

## Development

```bash
go test ./...
```
