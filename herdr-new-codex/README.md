# herdr-new-codex

通过 fzf 从最近访问的 zoxide 历史中选择目录：保留 Git 根目录和非 Git 目录，排除 Git 仓库内部的子目录。已有对应 workspace 时在其中新建 tab，否则创建 workspace。列表提供最多 100 个目录供模糊搜索，`scratch` 固定在第一项。

选择 `scratch` 时，每个 tab 都会使用 `mktemp` 创建新的 `herdr-scratch-XXXXXX` 临时目录。如果 `scratch` workspace 尚不存在，插件会先创建它。插件只会在检测到 Codex 的目录信任提示时自动选择 `Yes, continue`，不会向 `~/.codex/config.toml` 写入临时路径。

## Actions

- `sunznx.herdr-new-codex.codex`：选择 workspace 并新建 Codex tab。
- `sunznx.herdr-new-codex.herdr-close-codex`：如果 Codex 正在执行任务，先按 `Esc` 中止；随后发送 `/archive`、确认归档，并在 Codex 退出后只关闭触发 action 的 pane，不影响同 tab 的其他 pane。

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
- `git`
- `zoxide`
- Go 1.24+（仅安装或本地构建时需要）
