# herdr-copy-last-command-output

Copies the most recent completed command block from the triggering Herdr pane to the macOS clipboard.

## Actions

- `sunznx.herdr-copy-last-command-output.copy-last-command-output` copies only the command output.
- `sunznx.herdr-copy-last-command-output.copy-last-command-and-output` copies the command text without its prompt, followed by its output.

After a successful clipboard write, the plugin asks Herdr to show its standard `[ui.toast.clipboard]` feedback. Its enabled state and position therefore use the normal Herdr configuration. Feedback errors are ignored, so an unavailable toast does not turn a successful copy into a failed action.

## Installation

```bash
herdr plugin install sunznx/herdr-plugins/herdr-copy-last-command-output
```

Example key binding:

```toml
[[keys.command]]
key = "cmd+shift+c"
type = "plugin_action"
command = "sunznx.herdr-copy-last-command-output.copy-last-command-output"
description = "Copy last command output"
```

## Dependencies

- Herdr `>= 0.8.0`
- macOS `pbcopy`

## Limitations

Herdr `pane read` exposes terminal text but not shell semantic prompt markers. The plugin therefore identifies the last command from an empty prompt at the bottom of recent scrollback. It supports Oh My Zsh `robbyrussell` prompts when the working directory, Git branch, or dirty marker changes, prompts with a stable prefix, common `$`, `%`, `#`, `>`, `❯`, and `➜` markers, and common two-line p10k prompts. Other custom prompts that change between commands, a non-empty current prompt, wrapped or multiline commands, and output that resembles a prompt may not be recognized reliably. When it cannot identify a completed command with non-empty output, it fails without changing the clipboard.
