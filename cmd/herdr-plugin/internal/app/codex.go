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
	paneID, scratch, err := newTabInWorkspace(ctx, c, "new Codex tab ▸ ", "HERDR_NEW_CODEX_CHOICE", "HERDR_NEW_CODEX_CANDIDATES_FILE")
	if err != nil || paneID == "" {
		return err
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

func newPlainTab(ctx context.Context, c herdr.Client) error {
	paneID, _, err := newTabInWorkspace(ctx, c, "new tab ▸ ", "HERDR_NEW_TAB_CHOICE", "HERDR_NEW_TAB_CANDIDATES_FILE")
	if err != nil || paneID == "" {
		return err
	}
	return nil
}

func newClaude(ctx context.Context, c herdr.Client) error {
	paneID, _, err := newTabInWorkspace(ctx, c, "new Claude tab ▸ ", "HERDR_NEW_CLAUDE_CHOICE", "HERDR_NEW_CLAUDE_CANDIDATES_FILE")
	if err != nil || paneID == "" {
		return err
	}
	_, err = c.Run(ctx, "pane", "run", paneID, "exec claude")
	return err
}

func openNewCodexPicker(ctx context.Context, c herdr.Client, entrypoint string) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := c.OpenPane(ctx, "sunznx.herdr-new-codex", entrypoint, true, "")
		if err == nil || !strings.Contains(err.Error(), `"code":"ui_busy"`) || time.Now().After(deadline) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func newTabInWorkspace(ctx context.Context, c herdr.Client, prompt, choiceEnv, candidatesEnv string) (string, bool, error) {
	keepWorkspaceID := herdr.PluginContext().WorkspaceID
	if keepWorkspaceID == "" && (os.Getenv("LIVE_PANE_ID") != "" || herdr.PluginContext().FocusedPaneID != "") {
		if origin, originErr := herdr.OriginPane(ctx, c); originErr == nil {
			keepWorkspaceID = origin.WorkspaceID
		}
	}
	choice, err := pickWorkspace(ctx, c, prompt, keepWorkspaceID, choiceEnv, candidatesEnv, false, "")
	if err != nil || choice == nil {
		return "", false, err
	}
	scratch := choice.Kind == workspacepicker.Scratch
	var out []byte
	if scratch {
		cwd, err := os.MkdirTemp("", "herdr-scratch-")
		if err != nil {
			return "", false, err
		}
		if choice.WorkspaceID == "" {
			out, err = c.Run(ctx, "workspace", "create", "--label", "scratch", "--cwd", cwd, "--focus")
		} else {
			out, err = createTabInWorkspace(ctx, c, choice.WorkspaceID, cwd)
		}
	} else if choice.Kind == workspacepicker.Workspace {
		out, err = createTabInWorkspace(ctx, c, choice.WorkspaceID, choice.Path)
	} else if choice.Kind == workspacepicker.Directory {
		out, err = c.Run(ctx, "workspace", "create", "--cwd", choice.Path, "--focus")
	} else {
		return "", false, fmt.Errorf("invalid workspace choice %q", choice.Kind)
	}
	if err != nil {
		return "", false, err
	}
	var response struct {
		Result struct {
			RootPane herdr.Pane `json:"root_pane"`
		} `json:"result"`
	}
	if err := decode(out, &response); err != nil {
		return "", false, err
	}
	paneID := response.Result.RootPane.PaneID
	if paneID == "" {
		return "", false, fmt.Errorf("Herdr did not return a root pane")
	}
	return paneID, scratch, nil
}

func createTabInWorkspace(ctx context.Context, c herdr.Client, workspaceID, cwd string) ([]byte, error) {
	args := []string{"tab", "create", "--workspace", workspaceID}
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	args = append(args, "--focus")
	return c.Run(ctx, args...)
}

func waitAgent(ctx context.Context, c herdr.Client, paneID string, states ...string) (herdr.Pane, error) {
	args := []string{"agent", "wait", paneID}
	for _, state := range states {
		args = append(args, "--until", state)
	}
	args = append(args, "--timeout", "10000")
	var response struct {
		Result struct {
			Agent herdr.Pane `json:"agent"`
		} `json:"result"`
	}
	if err := c.JSON(ctx, &response, args...); err != nil {
		return herdr.Pane{}, err
	}
	return response.Result.Agent, nil
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
		current, err := waitAgent(ctx, c, pane.PaneID, "idle", "done", "unknown")
		if err != nil {
			return fmt.Errorf("Codex did not stop the current task within 10 seconds; pane left open")
		}
		if current.Agent != "codex" {
			return closePane()
		}
	}
	if _, err := c.Run(ctx, "agent", "prompt", pane.PaneID, "/archive"); err != nil {
		return err
	}
	if _, err := c.Run(ctx, "pane", "wait-output", pane.PaneID, "--match", "Archive this session?", "--source", "visible", "--timeout", "10000"); err != nil {
		current, getErr := c.GetPane(ctx, pane.PaneID)
		if getErr != nil || current.Agent != "codex" {
			return closePane()
		}
		return fmt.Errorf("Codex did not ask to archive within 10 seconds; pane left open")
	}
	if _, err := c.Run(ctx, "agent", "send-keys", pane.PaneID, "down", "enter"); err != nil {
		return err
	}
	if _, err := waitAgent(ctx, c, pane.PaneID, "unknown"); err == nil {
		return closePane()
	}
	recent, _ := c.Run(ctx, "pane", "read", pane.PaneID, "--source", "recent-unwrapped", "--lines", "60")
	if strings.Contains(string(recent), "Failed to archive current thread") {
		return closePane()
	}
	return fmt.Errorf("Codex did not finish /archive within 10 seconds; pane left open")
}
