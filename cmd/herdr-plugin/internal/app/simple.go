package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func openFromCWD(ctx context.Context, c herdr.Client, plugin, entrypoint string, split bool) error {
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	cwd := herdr.PaneCWD(pane)
	info, err := os.Stat(cwd)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("triggering pane cwd is unavailable: %s", cwd)
	}
	if split {
		_, err = c.Run(ctx, "plugin", "pane", "open", "--plugin", plugin, "--entrypoint", entrypoint, "--placement", "split", "--direction", "right", "--focus", "--cwd", cwd)
		return err
	}
	return c.OpenPane(ctx, plugin, entrypoint, true, cwd)
}

func openIn(ctx context.Context, c herdr.Client, application string) error {
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	cwd := herdr.PaneCWD(pane)
	info, err := os.Stat(cwd)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("triggering pane cwd is unavailable: %s", cwd)
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/open", "-a", application, cwd)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func yaziOpen(ctx context.Context, c herdr.Client, mode string) error {
	plugin := envOr("HERDR_PLUGIN_ID", "sunznx.yazi-popup")
	context := herdr.PluginContext()
	cwd := context.FocusedCWD
	if cwd == "" {
		cwd = context.WorkspaceCWD
	}
	if mode == "pick" {
		target := os.Getenv("HERDR_PANE_ID")
		if target == "" {
			return nil
		}
		return c.OpenPane(ctx, plugin, "picker", true, cwd, "HERDR_TARGET_PANE_ID="+target)
	}
	if mode != "fzf" && mode != "rg" && mode != "trellis" {
		return fmt.Errorf("unknown Yazi mode %q", mode)
	}
	if mode == "trellis" {
		cwd = filepath.Join(cwd, ".trellis")
	}
	info, err := os.Stat(cwd)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("directory is unavailable: %s", cwd)
	}
	return c.OpenPane(ctx, plugin, mode, true, cwd)
}

func yaziPicker(ctx context.Context, c herdr.Client) error {
	target := os.Getenv("HERDR_TARGET_PANE_ID")
	if target == "" {
		return fmt.Errorf("target pane ID is missing")
	}
	file, err := os.CreateTemp("", "herdr-yazi-chooser-*")
	if err != nil {
		return err
	}
	path := file.Name()
	_ = file.Close()
	defer os.Remove(path)
	if err := sharedfzf.Run(ctx, "yazi", "--chooser-file="+path); err != nil {
		return err
	}
	selected, err := os.Open(path)
	if err != nil {
		return err
	}
	defer selected.Close()
	home, _ := os.UserHomeDir()
	var parts []string
	scanner := bufio.NewScanner(selected)
	for scanner.Scan() {
		item := scanner.Text()
		if home != "" && strings.HasPrefix(item, home+string(os.PathSeparator)) {
			item = "~" + strings.TrimPrefix(item, home)
		}
		parts = append(parts, "@"+item)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(parts) == 0 {
		return nil
	}
	_, err = c.Run(ctx, "pane", "send-text", target, strings.Join(parts, " ")+" ")
	return err
}

func yaziBrowser(ctx context.Context, mode string) error {
	pid := strconv.Itoa(os.Getpid())
	var emit []string
	switch mode {
	case "fzf":
		emit = []string{"emit-to", pid, "plugin", "fzf"}
	case "rg":
		emit = []string{"emit-to", pid, "plugin", "fr", "rg"}
	case "trellis":
		emit = []string{"emit-to", pid, "sort", "natural", "--reverse=yes"}
	default:
		return fmt.Errorf("unknown Yazi browser mode %q", mode)
	}
	emitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 50 {
			if exec.CommandContext(emitCtx, "ya", emit...).Run() == nil {
				return
			}
			select {
			case <-emitCtx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()
	err := sharedfzf.Run(ctx, "yazi", "--client-id", pid, ".")
	cancel()
	<-done
	return err
}
