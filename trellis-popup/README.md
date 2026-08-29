# trellis-popup

Open the triggering pane's `.trellis` directory in a centered Yazi popup.

Yazi loads the user's normal configuration. After the popup starts, the plugin applies only this session's filename sort order:

```bash
ya emit-to <client-id> sort natural --reverse=yes
```

This keeps the user's `yazi.toml`, `keymap.toml`, theme, plugins, and environment intact instead of replacing `YAZI_CONFIG_HOME`. The popup also starts through the interactive `${SHELL:-/bin/zsh}`, so exported settings such as `FZF_DEFAULT_OPTS_FILE` remain available to Yazi and its built-in fzf integration.

## Requirements

- Herdr ≥ 0.8.0
- Yazi
- A `.trellis` directory under the triggering pane's current directory

## Install

```bash
herdr plugin install sunznx/herdr-plugins/trellis-popup
```

Add a keybinding to `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+t"
type = "plugin_action"
command = "sunznx.trellis-popup.open"
description = "Open .trellis in Yazi"
```

Reload the Herdr configuration, then press `prefix+t` from a project directory containing `.trellis`.

## Credits

The popup lifecycle and triggering-pane context handling follow [alastairsounds/herdr-plugins/yazi-popup](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup). This plugin adds `.trellis` targeting and a session-only reverse filename sort.
