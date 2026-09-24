# herdr-agent-sidebar

为 Herdr Agent 侧边栏安装一个紧凑的任务优先布局，并持续显示 agent 标识：

1. agent 标识、名称、状态文本
2. `herdr-auto-title` 维护的 pane 名称
3. Agent 在当前 Recent 排序中的全局 index

后台 Go daemon 由 Herdr `[[startup]]` 启动，不依赖 `launchctl`。它每两秒读取
`herdr agent list`，通过 pane metadata token 更新
`C`（Claude）、`X`（Codex）、`G`（Gemini）、`O`（OpenCode）或通用标识。
同时参考 `herdr-radar` 持久化最近工作时间，写入 `ws_key`、`tab_key`、`sort_key` 三个
metadata token，通过 `agent.view.set` 按 `Workspace 最近活动 → tab 最近活动 → Agent
最近活动` 排列；每个 Workspace 的首个 pane 显示 `$group`，末个 pane 显示 `$gap`，形成真正的分组。
标题解析交给 `herdr-auto-title`，本插件只负责分组、排序、index 和 logo。

插件只管理自己标记的 `config.toml` 区块，不覆盖已有的
`[ui.sidebar.agents]` 配置；发现已有自定义配置时会停止并提示用户处理。

## 本地开发

```bash
./herdr-plugin agent-sidebar self-test
herdr plugin link /Users/sunx/Dropbox/syncer/proj/github.com/sunznx/herdr-plugins/herdr-agent-sidebar
herdr server reload-config
```

安装后从插件 action 执行 `Configure Agent sidebar` 才会写入布局。插件发现
已有 `[ui.sidebar.agents]` 配置时会拒绝覆盖；先停用其它拥有该配置区块的侧边栏插件，
再执行 configure。

恢复 Herdr 原来的 Agent 侧边栏：

```bash
./herdr-plugin agent-sidebar unconfigure
```

插件会在修改前创建 `config.toml.herdr-agent-sidebar.bak`。

插件更新后无需重启 Herdr；执行 `Restart Agent sidebar` action 即可让新 daemon 接管旧实例。
