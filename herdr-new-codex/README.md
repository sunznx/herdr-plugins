# herdr-new-codex

通过 fzf 选择 workspace，在其中新建一个 tab 并启动 Codex。`scratch` 固定显示在列表末尾。

选择 `scratch` 时，每个 tab 都会使用 `mktemp` 创建新的 `herdr-scratch-XXXXXX` 临时目录。如果 `scratch` workspace 尚不存在，插件会先创建它。插件只会在检测到 Codex 的目录信任提示时自动选择 `Yes, continue`，不会向 `~/.codex/config.toml` 写入临时路径。

## Actions

- `sunznx.herdr-new-codex.codex`：选择 workspace 并新建 Codex tab。
- `sunznx.herdr-new-codex.herdr-close-codex`：向当前 Codex 发送 `/archive`，确认归档，等待 Codex 退出后关闭当前 tab。

## 安装

```bash
herdr plugin install sunznx/herdr-plugins/herdr-new-codex
```

在 `~/.config/herdr/config.toml` 中绑定 `prefix+c`：

```toml
[[keys.command]]
key = "prefix+c"
type = "plugin_action"
command = "sunznx.herdr-new-codex.codex"
description = "New Codex tab"
```

重新加载 Herdr 配置后，按 `prefix+c` 选择 workspace。

## 要求

- Herdr ≥ 0.8.0
- Codex CLI
- `fzf`
- `jq`
