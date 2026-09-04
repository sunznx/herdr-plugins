# herdr-ai-rename

Renames panes, tabs, and agents manually or uses Codex with
Codex to generate a short task slug from recent terminal output. It uses
`gpt-5.3-codex-spark` by default and falls back to `gpt-5.4-mini` when the primary model
reports an exhausted quota. Override them with `HERDR_AI_RENAME_MODEL` and
`HERDR_AI_RENAME_FALLBACK_MODEL` when needed.

Actions:

- `sunznx.herdr-ai-rename.rename`: rename the triggering pane and its detected agent.
- `sunznx.herdr-ai-rename.rename-all`: rename every pane and its detected agent with up to four concurrent Codex calls.
- `sunznx.herdr-ai-rename.open`: manually rename the triggering pane and its detected agent.
- `sunznx.herdr-ai-rename.tab`: manually rename the triggering tab.
- `sunznx.herdr-ai-rename.agent`: manually rename the detected agent in the triggering pane.

AI action 由 Herdr 异步执行，但 action 会保持 running，直到重命名真正完成；失败会记录在 plugin log，而不会提前报告成功。手动 action 会打开 fzf popup，按 `Esc` 取消。Agent 名称必须匹配 `[a-z][a-z0-9_-]{0,31}`。

Install from GitHub:

```bash
herdr plugin install sunznx/herdr-plugins/herdr-ai-rename -y
```

Requires Herdr `>= 0.8.0`, `fzf`, `codex` for AI actions, and Go 1.24+ when installing or building.
