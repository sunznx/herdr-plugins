package app

import (
	"context"
	"strings"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/clipboard"
	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/workspacepicker"
	sharedzoxide "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/zoxide"
)

func zoxideOpen(ctx context.Context, c herdr.Client) error {
	return workspacepicker.OpenPopup(ctx, workspacepicker.PopupOptions{
		PluginID:   envOr("HERDR_PLUGIN_ID", "sunznx.herdr-copy"),
		Entrypoint: "picker",
		Focus:      true,
	})
}

func zoxidePicker(ctx context.Context, c herdr.Client) error {
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
	return nil
}
