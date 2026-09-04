package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/clipboard"
	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/workspacepicker"
	sharedzoxide "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/zoxide"
)

func zoxideOpen(ctx context.Context, c herdr.Client) error {
	target, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	return workspacepicker.OpenPopup(ctx, workspacepicker.PopupOptions{
		PluginID:   envOr("HERDR_PLUGIN_ID", "sunznx.popup-zoxide"),
		Entrypoint: "picker",
		Focus:      true,
		Env:        []string{"HERDR_TARGET_PANE_ID=" + target.PaneID},
	})
}

func zoxidePicker(ctx context.Context, c herdr.Client) error {
	target := os.Getenv("HERDR_TARGET_PANE_ID")
	if target == "" {
		return fmt.Errorf("target pane ID is missing")
	}
	paths, err := sharedzoxide.List(ctx)
	if err != nil {
		return err
	}
	selected, err := sharedfzf.Pick(ctx, strings.Join(paths, "\n"), "--prompt=zoxide ▸ ")
	if err != nil || selected == "" {
		return err
	}
	if err := clipboard.Copy(ctx, selected); err != nil {
		return err
	}
	_, err = c.Run(ctx, "pane", "send-text", target, selected+" ")
	return err
}
