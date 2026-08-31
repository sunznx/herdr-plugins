# yazi-popup

Open [Yazi](https://yazi-rs.github.io/) over the triggering Herdr pane with four focused workflows:

| Action | Behavior |
| --- | --- |
| `sunznx.yazi-popup.pick` | Pick files and type them back into the triggering pane as unsubmitted `@path` references. |
| `sunznx.yazi-popup.fzf` | Open Yazi in the current directory and immediately enter its built-in fzf file navigator. |
| `sunznx.yazi-popup.rg` | Open Yazi in the current directory and immediately enter `fr.yazi` real-time `rg` content search. |
| `sunznx.yazi-popup.trellis` | Open the current directory's `.trellis` folder with filenames sorted naturally in reverse order. |

![Picking a file in the popup, typed back as @path into the composer](../_demo/yazi-popup/picker.gif)

Yazi loads the user's normal configuration without starting another interactive shell. Existing `FZF_DEFAULT_OPTS` and `FZF_DEFAULT_OPTS_FILE` values are preserved; when `FZF_DEFAULT_OPTS_FILE` is unset, the plugin automatically uses `${XDG_CONFIG_HOME:-$HOME/.config}/fzf/fzfrc` if that file exists.

## Requirements

- Herdr ≥ 0.8.0
- Yazi
- `fzf` for the `fzf` action and `fr.yazi` search interface
- [`fr.yazi`](https://github.com/lpnh/fg.yazi), installed with `ya pkg add lpnh/fr`
- `rg` and `bat` for the `rg` action
- A `.trellis` directory for the `trellis` action

## Install

```bash
herdr plugin install sunznx/herdr-plugins/yazi-popup
```

Example keybindings for `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+f"
type = "plugin_action"
command = "sunznx.yazi-popup.fzf"
description = "Open Yazi fzf"

[[keys.command]]
key = "prefix+r"
type = "plugin_action"
command = "sunznx.yazi-popup.rg"
description = "Search contents with fr.yazi"

[[keys.command]]
key = "prefix+t"
type = "plugin_action"
command = "sunznx.yazi-popup.trellis"
description = "Open .trellis in Yazi"
```

Reload the Herdr configuration after installing or changing keybindings.

## Credits

The original popup picker was created in [alastairsounds/herdr-plugins](https://github.com/alastairsounds/herdr-plugins/tree/main/yazi-popup). This fork adds the `fzf`, `fr.yazi` real-time `rg`, and `trellis` workflows, uses the `sunznx.yazi-popup` plugin ID, and avoids interactive-shell startup latency.
