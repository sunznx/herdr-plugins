package app

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func NewCommand() *cobra.Command {
	c := herdr.New()
	root := &cobra.Command{
		Use:           "herdr-plugin",
		Short:         "Shared runtime for Herdr plugins",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		group("gitui",
			leaf("open", func(ctx context.Context) error {
				return openFromCWD(ctx, c, "sunznx.gitui-popup", "gitui", false)
			}),
		),
		group("mole",
			leaf("open", func(ctx context.Context) error {
				return openFromCWD(ctx, c, "sunznx.mole-current-dir", "analyze", true)
			}),
		),
		group("open-in",
			argument("open application", nil, cobra.ExactArgs(1), func(ctx context.Context, application string) error {
				return openIn(ctx, c, application)
			}),
		),
		group("copy",
			leaf("output", func(ctx context.Context) error { return copyLast(ctx, c, false) }),
			leaf("command-and-output", func(ctx context.Context) error { return copyLast(ctx, c, true) }),
		),
		group("move",
			leaf("open", func(ctx context.Context) error { return moveOpen(ctx, c, false) }),
			leaf("open-tab", func(ctx context.Context) error { return moveOpen(ctx, c, true) }),
			leaf("workspace", func(ctx context.Context) error { return moveWorkspace(ctx, c) }),
			leaf("tab", func(ctx context.Context) error { return moveTab(ctx, c) }),
		),
		group("rename",
			leaf("ai-current", func(ctx context.Context) error { return aiRename(ctx, c, false) }),
			leaf("ai-all", func(ctx context.Context) error { return aiRename(ctx, c, true) }),
			argument("open [mode]", []string{"pane-agent", "tab", "agent"}, cobra.MaximumNArgs(1), func(ctx context.Context, mode string) error {
				if mode == "" {
					mode = "pane-agent"
				}
				return renameOpen(ctx, c, mode)
			}),
			leaf("picker", func(ctx context.Context) error { return renamePicker(ctx, c) }),
		),
		group("new-codex",
			leaf("open", func(ctx context.Context) error {
				return c.OpenPane(ctx, envOr("HERDR_PLUGIN_ID", "sunznx.herdr-new-codex"), "picker", true, "")
			}),
			leaf("picker", func(ctx context.Context) error { return newCodex(ctx, c) }),
			leaf("close", func(ctx context.Context) error { return closeCodex(ctx, c) }),
		),
		group("yazi",
			argument("open [mode]", []string{"pick", "fzf", "rg", "trellis"}, cobra.MaximumNArgs(1), func(ctx context.Context, mode string) error {
				if mode == "" {
					mode = "pick"
				}
				return yaziOpen(ctx, c, mode)
			}),
			leaf("picker", func(ctx context.Context) error { return yaziPicker(ctx, c) }),
			argument("browser [mode]", []string{"fzf", "rg", "trellis"}, cobra.MaximumNArgs(1), func(ctx context.Context, mode string) error {
				if mode == "" {
					mode = "fzf"
				}
				return yaziBrowser(ctx, mode)
			}),
		),
		group("palette",
			leaf("open", func(ctx context.Context) error { return paletteOpen(ctx, c) }),
			leaf("run", func(ctx context.Context) error { return palette(ctx, c) }),
		),
	)
	return root
}

func group(name string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{Use: name, Args: cobra.NoArgs}
	cmd.AddCommand(children...)
	return cmd
}

func leaf(name string, run func(context.Context) error) *cobra.Command {
	return &cobra.Command{
		Use:  name,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return run(cmd.Context()) },
	}
}

func argument(use string, valid []string, count cobra.PositionalArgs, run func(context.Context, string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:       use,
		ValidArgs: valid,
		Args:      cobra.MatchAll(count, cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return run(cmd.Context(), "")
			}
			return run(cmd.Context(), args[0])
		},
	}
	return cmd
}
