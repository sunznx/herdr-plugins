package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/workspacepicker"
)

// newTabKinds are the herdr-new tab kinds; each has a "<kind>-picker" pane.
var newTabKinds = []string{"tab", "codex", "claude", "agent"}

func newTab(ctx context.Context, c herdr.Client, kind string) error {
	switch kind {
	case "tab":
		return newPlainTab(ctx, c)
	case "codex":
		return newCodex(ctx, c)
	case "claude":
		return newClaude(ctx, c)
	case "agent":
		return newAgentTab(ctx, c)
	}
	return fmt.Errorf("unknown new tab kind %q", kind)
}

func newCodex(ctx context.Context, c herdr.Client) error {
	paneID, scratch, err := newTabInWorkspace(ctx, c, "new Codex tab ▸ ", "HERDR_NEW_CODEX_CHOICE", "HERDR_NEW_CODEX_CANDIDATES_FILE")
	if err != nil || paneID == "" {
		return err
	}
	return startAgentInPane(ctx, c, paneID, "codex", scratch)
}

// newAgentTab picks an agent, then a workspace, and starts it in a new tab there.
func newAgentTab(ctx context.Context, c herdr.Client) error {
	kinds := []string{"codex"}
	for _, kind := range append(supportedAgentKinds(ctx, c), "doubao") {
		_, err := exec.LookPath(kind)
		if !slices.Contains(kinds, kind) && (err == nil || len(agentInstallCommand(kind)) > 0) {
			kinds = append(kinds, kind)
		}
	}
	var rows strings.Builder
	for _, kind := range kinds {
		label := kind
		if _, err := exec.LookPath(kind); err != nil {
			label += " [未安装]"
		}
		fmt.Fprintf(&rows, "%s\t%s\n", kind, label)
	}
	kind := os.Getenv("HERDR_NEW_AGENT_KIND")
	if kind == "" {
		selected, err := sharedfzf.Pick(ctx, rows.String(), "--delimiter=\t", "--with-nth=2", "--prompt=new agent tab ▸ ", "--no-header", "--no-preview", "--reverse", "--cycle", "--no-multi", "--tiebreak=begin,index")
		if err != nil || selected == "" {
			return err
		}
		kind, _, _ = strings.Cut(selected, "\t")
	}
	if !slices.Contains(kinds, kind) {
		return fmt.Errorf("unknown agent %q", kind)
	}
	if _, err := exec.LookPath(kind); err != nil {
		if err := installAgent(ctx, kind); err != nil {
			return err
		}
	}
	paneID, scratch, err := newTabInWorkspace(ctx, c, "new "+kind+" tab ▸ ", "HERDR_NEW_AGENT_CHOICE", "HERDR_NEW_AGENT_CANDIDATES_FILE")
	if err != nil || paneID == "" {
		return err
	}
	return startAgentInPane(ctx, c, paneID, kind, scratch)
}

// startAgentInPane runs an agent kind, whose name is its canonical executable.
// Codex asks to trust each fresh scratch directory, so accept that prompt.
func startAgentInPane(ctx context.Context, c herdr.Client, paneID, kind string, scratch bool) error {
	if _, err := c.Run(ctx, "pane", "run", paneID, "exec "+kind); err != nil {
		return err
	}
	if kind != "codex" || !scratch {
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
	paneID, scratch, err := newTabInWorkspace(ctx, c, "new Claude tab ▸ ", "HERDR_NEW_CLAUDE_CHOICE", "HERDR_NEW_CLAUDE_CANDIDATES_FILE")
	if err != nil || paneID == "" {
		return err
	}
	return startAgentInPane(ctx, c, paneID, "claude", scratch)
}

func openNewTabPicker(ctx context.Context, c herdr.Client, kind string) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := c.OpenPane(ctx, "sunznx.herdr-new", kind+"-picker", true, "")
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
