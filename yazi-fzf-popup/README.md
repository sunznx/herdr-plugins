# yazi-fzf-popup

Open Yazi in a centered Herdr popup rooted at the triggering pane's current directory, then immediately enter Yazi's built-in fzf interface.

Yazi loads the user's normal configuration and starts through the interactive `${SHELL:-/bin/zsh}`. This keeps `yazi.toml`, `keymap.toml`, themes, plugins, `FZF_DEFAULT_OPTS`, and `FZF_DEFAULT_OPTS_FILE` available. The plugin triggers the same built-in action as Yazi's default `z` binding:

```bash
ya emit-to <client-id> plugin fzf
```

After choosing an item, Yazi reveals the file or enters the directory and remains open in the popup.

## Requirements

- Herdr ≥ 0.8.0
- Yazi
- `fzf`

## Install

```bash
herdr plugin install sunznx/herdr-plugins/yazi-fzf-popup
```

Add a keybinding to `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+f"
type = "plugin_action"
command = "sunznx.yazi-fzf-popup.open"
description = "Open Yazi fzf"
```

Reload the Herdr configuration, then press `prefix+f` from the directory you want to search.

## Credits

The popup lifecycle and triggering-pane context handling follow [alastairsounds/herdr-plugins/yazi-popup](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup). The search interface is Yazi's built-in [`fzf.lua`](https://yazi-rs.github.io/docs/plugins/builtins/#fzflua) plugin.
