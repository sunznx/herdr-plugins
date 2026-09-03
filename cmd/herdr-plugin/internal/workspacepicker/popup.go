package workspacepicker

import (
	"context"
	"fmt"
	"os"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

type PopupOptions struct {
	HerdrBin   string
	PluginID   string
	Entrypoint string
	Placement  string
	Focus      bool
	Env        []string
}

func OpenPopup(ctx context.Context, opts PopupOptions) error {
	if opts.HerdrBin == "" {
		opts.HerdrBin = envOr("HERDR_BIN_PATH", "herdr")
	}
	if opts.PluginID == "" {
		opts.PluginID = os.Getenv("HERDR_PLUGIN_ID")
	}
	if opts.PluginID == "" || opts.Entrypoint == "" {
		return fmt.Errorf("popup plugin and entrypoint are required")
	}
	if opts.Placement != "" && opts.Placement != "popup" {
		return fmt.Errorf("unsupported placement %q", opts.Placement)
	}
	return (herdr.Client{Bin: opts.HerdrBin}).OpenPane(ctx, opts.PluginID, opts.Entrypoint, opts.Focus, "", opts.Env...)
}

func RunWithFZFEnv(ctx context.Context, command string, args ...string) error {
	return sharedfzf.Run(ctx, command, args...)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
