# gitui-popup

Open [GitUI](https://github.com/gitui-org/gitui) in an `80% × 80%` popup rooted at the directory of the Herdr pane that triggered it.

## Requirements

- Herdr ≥ 0.8.0
- [GitUI](https://github.com/gitui-org/gitui)

Install GitUI with Homebrew:

```bash
brew install gitui
```

## Install

Install the plugin from GitHub:

```bash
herdr plugin install sunznx/herdr-plugins/gitui-popup
```

Link a local checkout while developing:

```bash
herdr plugin link ~/Dropbox/syncer/proj/github.com/sunznx/herdr-plugins/gitui-popup
```

Add the shortcut to `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+g"
type = "plugin_action"
command = "sunznx.gitui-popup.open"
description = "Open GitUI"
```

Reload the Herdr configuration, then press `prefix+g` from the repository you want to inspect.

## Credits

The triggering-pane context and popup-opening pattern follow [yazi-popup](../yazi-popup), which originates from [alastairsounds/herdr-plugins](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup). The popup runs [GitUI](https://github.com/gitui-org/gitui).
