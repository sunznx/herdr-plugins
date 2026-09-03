package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/workspacepicker"
)

func newCodex(ctx context.Context, c herdr.Client) error {
	choice, err := pickWorkspace(ctx, c, "new Codex tab ▸ ", "", "HERDR_NEW_CODEX_CHOICE", "HERDR_NEW_CODEX_CANDIDATES_FILE")
	if err != nil || choice == nil {
		return err
	}
	scratch := choice.Kind == workspacepicker.Scratch
	var out []byte
	if scratch {
		cwd, err := os.MkdirTemp("", "herdr-scratch-")
		if err != nil {
			return err
		}
		if choice.WorkspaceID == "" {
			out, err = c.Run(ctx, "workspace", "create", "--label", "scratch", "--cwd", cwd, "--focus")
		} else {
			out, err = c.Run(ctx, "tab", "create", "--workspace", choice.WorkspaceID, "--cwd", cwd, "--focus")
		}
	} else if choice.Kind == workspacepicker.Workspace {
		out, err = c.Run(ctx, "tab", "create", "--workspace", choice.WorkspaceID, "--cwd", choice.Path, "--focus")
	} else if choice.Kind == workspacepicker.Directory {
		out, err = c.Run(ctx, "workspace", "create", "--cwd", choice.Path, "--focus")
	} else {
		return fmt.Errorf("invalid workspace choice %q", choice.Kind)
	}
	if err != nil {
		return err
	}
	var response struct {
		Result struct {
			RootPane herdr.Pane `json:"root_pane"`
		} `json:"result"`
	}
	if err := decode(out, &response); err != nil {
		return err
	}
	paneID := response.Result.RootPane.PaneID
	if paneID == "" {
		return fmt.Errorf("Herdr did not return a root pane")
	}
	if _, err := c.Run(ctx, "pane", "run", paneID, "exec codex"); err != nil {
		return err
	}
	if !scratch {
		return nil
	}
	for range 100 {
		screen, _ := c.Run(ctx, "pane", "read", paneID, "--source", "visible", "--lines", "60")
		if strings.Contains(string(screen), "Do you trust the contents of this directory?") {
			_, err := c.Run(ctx, "pane", "send-keys", paneID, "enter")
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("Codex trust prompt did not appear within 10 seconds")
}

func closeCodex(ctx context.Context, c herdr.Client) error {
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	if pane.Agent != "codex" {
		return fmt.Errorf("the current pane is not running Codex")
	}
	closePane := func() error {
		out, err := c.Run(ctx, "pane", "close", pane.PaneID)
		if err == nil || strings.Contains(string(out), `"code":"pane_not_found"`) || strings.Contains(err.Error(), "pane_not_found") {
			return nil
		}
		return err
	}
	if pane.AgentStatus == "working" || pane.AgentStatus == "blocked" {
		if _, err := c.Run(ctx, "agent", "send-keys", pane.PaneID, "esc"); err != nil {
			return err
		}
		stopped := false
		for range 100 {
			current, getErr := c.GetPane(ctx, pane.PaneID)
			if getErr != nil || current.Agent != "codex" {
				return closePane()
			}
			if current.AgentStatus == "idle" || current.AgentStatus == "done" {
				stopped = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !stopped {
			return fmt.Errorf("Codex did not stop the current task within 10 seconds; pane left open")
		}
	}
	if _, err := c.Run(ctx, "agent", "prompt", pane.PaneID, "/archive"); err != nil {
		return err
	}
	confirmed := false
	for range 100 {
		current, getErr := c.GetPane(ctx, pane.PaneID)
		if getErr != nil || current.Agent != "codex" {
			return closePane()
		}
		recent, _ := c.Run(ctx, "pane", "read", pane.PaneID, "--source", "recent-unwrapped", "--lines", "60")
		text := string(recent)
		if !confirmed && (current.AgentStatus == "blocked" || strings.Contains(text, "Archive this session?")) {
			if _, err := c.Run(ctx, "agent", "send-keys", pane.PaneID, "down", "enter"); err != nil {
				return err
			}
			confirmed = true
		} else if confirmed && strings.Contains(text, "Failed to archive current thread") {
			return closePane()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("Codex did not finish /archive within 10 seconds; pane left open")
}
