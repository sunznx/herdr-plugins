# mole-current-dir

Open a new Herdr pane in the triggering pane's directory and run:

```bash
mole analyze .
```

The analyzer starts through the interactive `${SHELL:-/bin/zsh}`, so it inherits the same exported environment as the user's shell.

## Requirements

- Herdr ≥ 0.8.0
- [Mole](https://github.com/tw93/Mole)

## Install

```bash
herdr plugin install sunznx/herdr-plugins/mole-current-dir
```

Add a keybinding to `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "sunznx.mole-current-dir.open"
description = "Analyze current directory with Mole"
```

Reload the Herdr configuration, then press `prefix+m` from the pane whose directory you want to inspect.

## Credits

The triggering-pane context and pane-opening pattern are based on [alastairsounds/herdr-plugins/yazi-popup](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup). The Mole integration is specific to this repository.
