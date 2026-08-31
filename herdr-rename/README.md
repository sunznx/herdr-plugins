# herdr-rename

Renames the pane that triggered the action. If Herdr has detected an agent in
that pane, the same name is applied to the agent.

## Install

```bash
herdr plugin install sunznx/herdr-plugins/herdr-rename
```

## Action

| Action ID | Title |
| --- | --- |
| `sunznx.herdr-rename.open` | `Rename pane and agent…` |

The action opens a focused Herdr popup and accepts the new name through fzf.
Press `Esc` to cancel without changing either name.

When an agent is present, the name must match
`[a-z][a-z0-9_-]{0,31}`. A pane without an agent can use a normal Herdr pane
label, including spaces.

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

## Pane context

The action validates `LIVE_PANE_ID` when it is supplied by another plugin such
as `command-palette-popup`. A direct key binding normally has no
`LIVE_PANE_ID`, so the action resolves its caller with
`herdr pane current --current` before opening the popup. The resolved pane ID
is passed into the popup through `HERDR_RENAME_PANE_ID`; the popup itself is
never used as the rename target.

## Requirements

- Herdr `>= 0.8.0`
- `fzf`
- `jq`
