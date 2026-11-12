# herdr-new

通过 fzf 从最近访问的 zoxide 历史中选择目录：保留 Git 根目录和非 Git 目录，排除 Git 仓库内部的子目录。已有对应 workspace 时在其中新建 tab，否则创建 workspace。列表提供最多 100 个目录供模糊搜索，`scratch` 固定在第一项。

选择 `scratch` 时，每个 tab 都会使用 `mktemp` 创建新的 `herdr-scratch-XXXXXX` 临时目录。如果 `scratch` workspace 尚不存在，插件会先创建它。插件只会在检测到 Codex 的目录信任提示时自动选择 `Yes, continue`，不会向 `~/.codex/config.toml` 写入临时路径。

## Actions

- `sunznx.herdr-new.tab`：选择 workspace 并新建普通 tab。
- `sunznx.herdr-new.codex`：选择 workspace 并新建 Codex tab。
- `sunznx.herdr-new.claude`：选择 workspace 并新建 Claude tab。
- `sunznx.herdr-new.agent`：先用 fzf 选择 agent，再选择 workspace，在新 tab 中启动该 agent。`codex` 固定在第一项。
- `sunznx.herdr-new.close-codex`：如果 Codex 正在执行任务，先按 `Esc` 中止；随后发送 `/archive`、确认归档，并在 Codex 退出后只关闭触发 action 的 pane，不影响同 tab 的其他 pane。

从 `command-palette-popup` 选择以上任一新建 tab 的 action 时会打开对应的 picker popup；在 popup 中确认目录后才创建 tab。

`agent` 列表通过 `PATH` 检查 CLI 是否可用。列表只显示名称和 `[未安装]` 状态，不显示说明或预览。选中缺失项后，插件自动安装，成功后继续选择 workspace；安装失败或命令仍不可用时停止，不创建 tab。

支持自动安装 Codex、Claude、Gemini、OpenCode 和豆包 CLI。Claude 使用 Homebrew，其余使用 npm。列表也保留 Herdr 支持且已安装的其他 agent。

## 豆包 CLI

按[豆包 CLI 官方说明](https://www.doubao.com/work/docs/zh-cn/doubao-cli#hc-cli-get-started)准备 Node.js 18+，然后安装并登录：

```bash
npm install -g '@bytedance/doubao-cli' --registry=https://registry.npmjs.org/
doubao login
doubao auth status --verify
```

也可以在 `New Agent tab in workspace` 中直接选择标有 `[未安装]` 的 `doubao`，由插件执行安装。豆包 CLI 当前需要豆包团队或企业订阅。首次启动后的登录由 CLI 引导。插件在新 tab 中执行 `doubao`；Herdr 的原生 agent 状态检测和会话恢复取决于核心对豆包的支持。

## 安装

```bash
herdr plugin install sunznx/herdr-plugins/herdr-new
```

在 `~/.config/herdr/config.toml` 中绑定 `prefix+c`：

```toml
[[keys.command]]
key = "prefix+c"
type = "plugin_action"
command = "sunznx.herdr-new.codex"
description = "New Codex tab in workspace"
```

重新加载 Herdr 配置后，按 `prefix+c` 选择 workspace。

## 要求

- Herdr ≥ 0.8.0
- 所选 agent 的 CLI，或对应的安装工具（npm 或 Homebrew）；`tab` action 不需要 agent CLI
- `fzf`
- `git`
- `zoxide`
- Go 1.24+（仅安装或本地构建时需要）
