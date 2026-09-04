package herdr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Client struct {
	Bin    string
	DryRun bool
}

type Pane struct {
	PaneID                string `json:"pane_id"`
	TabID                 string `json:"tab_id"`
	WorkspaceID           string `json:"workspace_id"`
	CWD                   string `json:"cwd"`
	ForegroundCWD         string `json:"foreground_cwd"`
	Label                 string `json:"label"`
	TerminalTitleStripped string `json:"terminal_title_stripped"`
	Agent                 string `json:"agent"`
	AgentStatus           string `json:"agent_status"`
}

type Context struct {
	FocusedPaneID string `json:"focused_pane_id"`
	TabID         string `json:"tab_id"`
	WorkspaceID   string `json:"workspace_id"`
	FocusedCWD    string `json:"focused_pane_cwd"`
	WorkspaceCWD  string `json:"workspace_cwd"`
}

type TargetContext struct {
	Pane      string `json:"pane"`
	Tab       string `json:"tab"`
	Workspace string `json:"workspace"`
	CWD       string `json:"cwd"`
}

func MergePopupContext(base TargetContext, popup Context) TargetContext {
	if popup.FocusedPaneID == "" {
		return base
	}
	base.Pane = popup.FocusedPaneID
	if popup.TabID != "" {
		base.Tab = popup.TabID
	}
	if popup.WorkspaceID != "" {
		base.Workspace = popup.WorkspaceID
	}
	if popup.FocusedCWD != "" {
		base.CWD = popup.FocusedCWD
	} else if popup.WorkspaceCWD != "" {
		base.CWD = popup.WorkspaceCWD
	}
	return base
}

func New() Client {
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		bin = "herdr"
	}
	return Client{Bin: bin, DryRun: os.Getenv("CPP_DRY_RUN") == "1"}
}

func (c Client) Run(ctx context.Context, args ...string) ([]byte, error) {
	if c.DryRun && !readOnly(args) {
		fmt.Fprintf(os.Stderr, "DRY-RUN: %s %s\n", c.Bin, strings.Join(args, " "))
		return nil, nil
	}
	cmd := exec.CommandContext(ctx, c.Bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("herdr %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return out, nil
}

func readOnly(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "--default-config" || args[0] == "completion" {
		return true
	}
	if len(args) < 2 {
		return false
	}
	key := args[0] + " " + args[1]
	switch key {
	case "pane list", "pane get", "pane current", "pane read", "pane wait-output",
		"tab list", "tab get", "workspace list", "agent list", "agent wait", "worktree list",
		"plugin config-dir":
		return true
	case "plugin action", "plugin log":
		return len(args) >= 3 && args[2] == "list"
	default:
		return false
	}
}

func (c Client) JSON(ctx context.Context, dst any, args ...string) error {
	out, err := c.Run(ctx, args...)
	if err != nil {
		return err
	}
	if c.DryRun && len(out) == 0 {
		return nil
	}
	if err := json.Unmarshal(out, dst); err != nil {
		return fmt.Errorf("decode herdr %s JSON: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (c Client) Pane(ctx context.Context, id string) (Pane, error) {
	if id != "" {
		if pane, err := c.GetPane(ctx, id); err == nil {
			return pane, nil
		}
	}
	return c.CurrentPane(ctx)
}

func (c Client) GetPane(ctx context.Context, id string) (Pane, error) {
	var response struct {
		Result struct {
			Pane Pane `json:"pane"`
		} `json:"result"`
	}
	if id == "" {
		return Pane{}, fmt.Errorf("pane ID is required")
	}
	if err := c.JSON(ctx, &response, "pane", "get", id); err != nil {
		return Pane{}, err
	}
	if response.Result.Pane.PaneID != id {
		return Pane{}, fmt.Errorf("pane %q is unavailable", id)
	}
	return response.Result.Pane, nil
}

func (c Client) CurrentPane(ctx context.Context) (Pane, error) {
	var response struct {
		Result struct {
			Pane Pane `json:"pane"`
		} `json:"result"`
	}
	if err := c.JSON(ctx, &response, "pane", "current", "--current"); err != nil {
		return Pane{}, err
	}
	if response.Result.Pane.PaneID == "" {
		return Pane{}, fmt.Errorf("could not resolve the triggering pane")
	}
	return response.Result.Pane, nil
}

func (c Client) OpenPane(ctx context.Context, plugin, entrypoint string, focus bool, cwd string, env ...string) error {
	args := []string{"plugin", "pane", "open", "--plugin", plugin, "--entrypoint", entrypoint}
	args = append(args, "--placement", "popup")
	if focus {
		args = append(args, "--focus")
	}
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	for _, value := range env {
		args = append(args, "--env", value)
	}
	_, err := c.Run(ctx, args...)
	return err
}

func PluginContext() Context {
	var result Context
	_ = json.Unmarshal([]byte(os.Getenv("HERDR_PLUGIN_CONTEXT_JSON")), &result)
	return result
}

func OriginPane(ctx context.Context, c Client) (Pane, error) {
	candidate := os.Getenv("LIVE_PANE_ID")
	if candidate == "" {
		candidate = PluginContext().FocusedPaneID
	}
	return c.Pane(ctx, candidate)
}

func PaneCWD(p Pane) string {
	if p.ForegroundCWD != "" {
		return p.ForegroundCWD
	}
	return p.CWD
}
