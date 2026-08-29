# herdr-plugins

Personal [Herdr](https://herdr.dev) plugins.

## Plugins

| Plugin | Description |
| --- | --- |
| [yazi-popup](yazi-popup) | Pops [Yazi](https://yazi-rs.github.io/) open over the triggering pane and types picks back as `@path`. |
| [yazi-fzf-popup](yazi-fzf-popup) | Opens Yazi over the triggering pane and immediately enters its built-in fzf interface. |
| [command-palette-popup](command-palette-popup) | Searches Herdr native commands, live targets, and installed plugin actions from one fzf popup. |
| [mole-current-dir](mole-current-dir) | Opens a pane and runs `mole analyze .` in the triggering pane's directory. |
| [trellis-popup](trellis-popup) | Opens the current project's `.trellis` directory in a reverse-name-sorted Yazi popup. |
| [open-in](open-in) | Opens the triggering pane's current directory in Emacs, IntelliJ IDEA, or Fork. |

### yazi-popup

![Picking a file in the popup, typed back as @path into the composer](_demo/yazi-popup/picker.gif)

## Installation

Each plugin lives in its own subdirectory. Install with:

```bash
herdr plugin install sunznx/herdr-plugins/yazi-popup
herdr plugin install sunznx/herdr-plugins/yazi-fzf-popup
herdr plugin install sunznx/herdr-plugins/command-palette-popup
herdr plugin install sunznx/herdr-plugins/mole-current-dir
herdr plugin install sunznx/herdr-plugins/trellis-popup
herdr plugin install sunznx/herdr-plugins/open-in
```

See each plugin's own README for setup details.

## Credits

- `yazi-popup` originates from [alastairsounds/herdr-plugins](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup) and is modified here to launch through the user's interactive shell.
- `command-palette-popup` combines ideas and code from [enekos/herdr-quick-actions](https://github.com/enekos/herdr-quick-actions) and [JanTvrdik/herdr-command-palette](https://github.com/JanTvrdik/herdr-command-palette). Their MIT notices are retained in the plugin's [LICENSE](command-palette-popup/LICENSE).
- `yazi-fzf-popup`, `mole-current-dir`, `trellis-popup`, and `open-in` reuse the Herdr context and pane-launching patterns demonstrated by `yazi-popup`; their task-specific behavior was added in this repository.
