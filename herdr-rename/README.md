# herdr-rename

Renames the tab, pane, or agent associated with the pane that triggered an
action.

## Install

```bash
herdr plugin install sunznx/herdr-plugins/herdr-rename
```

## Action

| Action ID | Title |
| --- | --- |
| `sunznx.herdr-rename.open` | `Rename pane and agent…` |
| `sunznx.herdr-rename.tab` | `Rename tab…` |
| `sunznx.herdr-rename.agent` | `Rename agent…` |

Each action opens a focused Herdr popup and accepts the new name through fzf.
Press `Esc` to cancel without changing a name. Tab and pane labels may contain
spaces. Agent names must match `[a-z][a-z0-9_-]{0,31}`.

`Rename agent…` targets the agent in the triggering pane and fails without
changing anything when that pane has no detected agent.

The plugin renames the agent first because that operation has the stricter
validation and uniqueness checks. If it fails, the pane label stays unchanged.
The pane is renamed only after the agent rename succeeds.

## Key binding

```toml
[[keys.command]]
key = "prefix+n"
type = "plugin_action"
command = "sunznx.herdr-rename.open"
description = "rename pane and agent"
```

Use either standalone action the same way:

```toml
[[keys.command]]
key = "prefix+r"
type = "plugin_action"
command = "sunznx.herdr-rename.tab"
description = "rename tab"

[[keys.command]]
key = "prefix+a"
type = "plugin_action"
command = "sunznx.herdr-rename.agent"
description = "rename agent"
```

## Pane context

The action validates `LIVE_PANE_ID` when it is supplied by another plugin such
as `command-palette-popup`. A direct key binding normally has no
`LIVE_PANE_ID`, so the action resolves its caller with
`herdr pane current --current` before opening the popup. The resolved pane and
tab IDs are passed into the popup through `HERDR_RENAME_PANE_ID` and
`HERDR_RENAME_TAB_ID`; the popup itself is never used as the rename target.
Targets are checked again immediately before renaming.

## Requirements

- Herdr `>= 0.8.0`
- `fzf`
- `jq`
