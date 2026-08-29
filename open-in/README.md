# open-in

Open the triggering pane's current directory in one of three macOS applications:

- `sunznx.open-in.open-in-emacs` → Emacs
- `sunznx.open-in.open-in-idea` → IntelliJ IDEA
- `sunznx.open-in.open-in-fork` → Fork

## Requirements

- Herdr ≥ 0.8.0
- The corresponding macOS application installed as `Emacs.app`, `IntelliJ IDEA.app`, or `Fork.app`

## Install

```bash
herdr plugin install sunznx/herdr-plugins/open-in
```

The actions can be invoked from `command-palette-popup` without dedicated keybindings. To bind one directly, add a `plugin_action` entry to `~/.config/herdr/config.toml`, for example:

```toml
[[keys.command]]
key = "prefix+e"
type = "plugin_action"
command = "sunznx.open-in.open-in-emacs"
description = "Open current directory in Emacs"
```

## Credits

The triggering-pane context handling follows [alastairsounds/herdr-plugins/yazi-popup](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup). The application actions are specific to this repository.
