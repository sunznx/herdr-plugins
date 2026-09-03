# command-palette-popup

An [fzf](https://github.com/junegunn/fzf) command palette shown in a temporary [Herdr](https://herdr.dev) popup.

The list combines:

- Herdr native tab, pane, workspace, agent, and config commands
- live targets for open tabs and agents
- actions exposed by every installed Herdr plugin
- effective Herdr keybindings for native commands
- `Move pane to workspace`, `Move pane to tab`, `Rename pane and agent`,
  `Rename tab`, and `Rename agent` flows that keep the source pane's live ID

Native commands and plugin actions are ranked by usage frequency and recency. Plugin action failures are read from the Herdr action log and shown before the popup closes.

`New workspace` 与 `Move pane to workspace` 使用同一套 fzf 目录选择：候选优先来自 zoxide 历史，保留 Git 根目录和非 Git 目录，排除 Git 仓库内部的子目录；两者都提供最多 100 个目录供模糊搜索。已有对应 workspace 时切换或移动到该 workspace，否则由 Herdr 创建并自动命名。`scratch` 固定在第一项。

The `open` action resolves the triggering pane before creating the popup. The popup is a session-modal terminal, not a movable Herdr pane, and its authoritative `HERDR_PLUGIN_CONTEXT_JSON` still identifies the tiled pane underneath it. Pane moves and renames therefore cannot target the command palette itself. Their nested selection or input happens inside the existing popup, so the palette never tries to open a second popup.

The palette replaces Herdr's native `Rename tab` and generic agent picker with
the `sunznx.herdr-ai-rename.tab` and `sunznx.herdr-ai-rename.agent` actions. They run
inline in the existing palette popup and target the tab or agent under it.

The popup runs through the repository's shared Go plugin runtime without loading an interactive shell.

## Requirements

- Herdr ≥ 0.8.0
- `fzf`
- `git`
- `zoxide`
- Go 1.24+（仅安装或本地构建时需要）

## Install

```bash
herdr plugin install sunznx/herdr-plugins/command-palette-popup
```

Add a keybinding to `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+p"
type = "plugin_action"
command = "sunznx.command-palette-popup.open"
description = "Command palette"
```

Reload the Herdr configuration, then press `prefix+p`.

## Configuration

Plugin configuration lives at:

```text
~/.config/herdr/plugins/config/sunznx.command-palette-popup/config.toml
```

Worktree actions and unopened-worktree targets are hidden by default. Enable them with:

```toml
enable_worktree = true
```

## Debugging

The picker can be exercised without opening fzf:

```bash
CPP_CONTEXT_JSON='{"pane":"w1:p1","tab":"w1:t1","workspace":"w1","cwd":"/repo"}' \
CPP_LIST_ONLY=1 \
HERDR_PANE_ID= \
./herdr-plugin palette run
```

Set `CPP_CHOICE` to `<kind><TAB><payload>` and `CPP_DRY_RUN=1` to test dispatch without running the selected command. `CPP_PICK_VALUE` supplies the result of a nested picker such as the workspace chooser.

## Credits

This plugin combines the native-command and live-target workflow from [enekos/herdr-quick-actions](https://github.com/enekos/herdr-quick-actions) with installed-plugin action discovery and failure reporting from [JanTvrdik/herdr-command-palette](https://github.com/JanTvrdik/herdr-command-palette). Both projects are MIT licensed; their copyright notices are retained in [LICENSE](LICENSE).
