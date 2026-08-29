# command-palette-popup

A centered [fzf](https://github.com/junegunn/fzf) command palette for [Herdr](https://herdr.dev).

The list combines:

- Herdr native tab, pane, workspace, worktree, agent, and config commands
- live targets for open tabs, workspaces, agents, and unopened worktrees
- actions exposed by every installed Herdr plugin
- effective Herdr keybindings for native commands

Native commands and plugin actions are ranked by usage frequency and recency. Plugin action failures are read from the Herdr action log and shown before the popup closes.

The popup starts through the interactive `${SHELL:-/bin/zsh}`, so exported settings such as `FZF_DEFAULT_OPTS_FILE` are available to fzf.

## Requirements

- Herdr ≥ 0.8.0
- `fzf`
- `jq`

## Install

```bash
herdr plugin install sunznx/herdr-plugins/command-palette-popup
```

Add a keybinding to `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+p"
type = "plugin_action"
command = "sunznx.command-palette-popup.open"
description = "Command palette"
```

Reload the Herdr configuration, then press `prefix+p`.

## Debugging

The picker can be exercised without opening fzf:

```bash
CPP_CONTEXT_JSON='{"pane":"w1:p1","tab":"w1:t1","workspace":"w1","cwd":"/repo"}' \
CPP_LIST_ONLY=1 \
HERDR_PANE_ID= \
bash palette.sh
```

Set `CPP_CHOICE` to `<kind><TAB><payload>` and `CPP_DRY_RUN=1` to test dispatch without running the selected command.

## Credits

This plugin combines the native-command and live-target workflow from [enekos/herdr-quick-actions](https://github.com/enekos/herdr-quick-actions) with installed-plugin action discovery and failure reporting from [JanTvrdik/herdr-command-palette](https://github.com/JanTvrdik/herdr-command-palette). Both projects are MIT licensed; their copyright notices are retained in [LICENSE](LICENSE).
