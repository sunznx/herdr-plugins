# herdr-plugins

Personal [Herdr](https://herdr.dev) plugins.

## Plugins

| Plugin | Description |
| --- | --- |
| [yazi-popup](yazi-popup) | Provides Yazi popups for `@path` picking, `fr.yazi` real-time content search, and reverse-sorted `.trellis` browsing. |
| [lazygit-popup](lazygit-popup) | Opens lazygit in an `80% × 80%` popup rooted at the triggering pane's directory. |
| [command-palette-popup](command-palette-popup) | Searches Herdr native commands, live targets, and installed plugin actions from one fzf popup. |
| [herdr-move](herdr-move) | Moves the triggering pane and its running agent into an existing workspace without reusing stale pane IDs. |
| [mole-current-dir](mole-current-dir) | Opens a pane and runs `mole analyze .` in the triggering pane's directory. |
| [open-in](open-in) | Opens the triggering pane's current directory in Emacs, IntelliJ IDEA, or Fork. |

### yazi-popup

![Picking a file in the popup, typed back as @path into the composer](_demo/yazi-popup/picker.gif)

## Installation

Each plugin lives in its own subdirectory. Install with:

```bash
herdr plugin install sunznx/herdr-plugins/yazi-popup
herdr plugin install sunznx/herdr-plugins/lazygit-popup
herdr plugin install sunznx/herdr-plugins/command-palette-popup
herdr plugin install sunznx/herdr-plugins/herdr-move
herdr plugin install sunznx/herdr-plugins/mole-current-dir
herdr plugin install sunznx/herdr-plugins/open-in
```

See each plugin's own README for setup details.

## Credits

- `yazi-popup` originates from [alastairsounds/herdr-plugins](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup) and is extended here with `fr.yazi` real-time rg and `.trellis` workflows.
- `command-palette-popup` combines ideas and code from [enekos/herdr-quick-actions](https://github.com/enekos/herdr-quick-actions) and [JanTvrdik/herdr-command-palette](https://github.com/JanTvrdik/herdr-command-palette). Their MIT notices are retained in the plugin's [LICENSE](command-palette-popup/LICENSE).
- `lazygit-popup`, `herdr-move`, `mole-current-dir`, and `open-in` reuse the Herdr context and pane-launching patterns demonstrated by `yazi-popup`; their task-specific behavior was added in this repository.
