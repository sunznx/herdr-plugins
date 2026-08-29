# lazygit-popup

Open [lazygit](https://github.com/jesseduffield/lazygit) in an `80% × 80%` popup rooted at the directory of the Herdr pane that triggered it.

## Requirements

- Herdr ≥ 0.8.0
- lazygit

## Install

Install from GitHub:

```bash
herdr plugin install sunznx/herdr-plugins/lazygit-popup
```

Link a local checkout while developing:

```bash
herdr plugin link ~/Dropbox/syncer/proj/github.com/sunznx/herdr-plugins/lazygit-popup
```

Add the shortcut to `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+g"
type = "plugin_action"
command = "sunznx.lazygit-popup.open"
description = "Open lazygit"
```

Reload the Herdr configuration, then press `prefix+g` from the repository you want to inspect.

## Credits

The triggering-pane context and popup-opening pattern follow [yazi-popup](../yazi-popup), which originates from [alastairsounds/herdr-plugins](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup).
