# herdr-ai-rename

Uses Codex with `gpt-5.3-codex-spark` to generate a short task slug from recent terminal output.

Actions:

- `sunznx.herdr-ai-rename.rename`: rename the triggering pane and its detected agent.
- `sunznx.herdr-ai-rename.rename-all`: rename every pane and its detected agent with up to four concurrent Codex calls.

Both actions return immediately so a calling popup can close while the rename runs in the background.

Install locally:

```bash
herdr plugin link ./herdr-ai-rename --enabled
```
