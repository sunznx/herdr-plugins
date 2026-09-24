# Herdr Copy

Provides actions for copying directories, agent session references, command output, and zoxide paths, plus resuming/forking supported agent sessions in a new tab.

The zoxide action copies the selected directory only; it does not insert text into the triggering pane.

Fork/resume actions opened from the command palette use the original pane context and run the agent command directly in the new tab.

Actions:

- `copy-current-dir`
- `copy-current-agent-session`
- `copy-last-command-output`
- `copy-last-command-and-output`
- `copy-zoxide-directory`
- `fork-current-agent-session-in-new-tab`
- `resume-current-agent-session-in-new-tab`
