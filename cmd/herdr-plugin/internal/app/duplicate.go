package app

import (
	"context"
	"fmt"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func duplicateTabOrAgent(ctx context.Context, c herdr.Client) error {
	origin, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	agents, err := listAgents(ctx, c)
	if err != nil {
		return err
	}

	args := []string{"tab", "create"}
	if origin.WorkspaceID != "" {
		args = append(args, "--workspace", origin.WorkspaceID)
	}
	if cwd := herdr.PaneCWD(origin); cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	args = append(args, "--focus")
	out, err := c.Run(ctx, args...)
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
	if response.Result.RootPane.PaneID == "" {
		return fmt.Errorf("Herdr did not return a root pane")
	}
	agent := duplicateAgent(origin, agents)
	if agent == "" {
		return nil
	}
	_, err = c.Run(ctx, "pane", "run", response.Result.RootPane.PaneID, "exec "+agent)
	return err
}

func duplicateAgent(origin herdr.Pane, agents []agentRow) string {
	if origin.Agent == "codex" || origin.Agent == "claude" {
		return origin.Agent
	}
	for _, agent := range []string{"codex", "claude"} {
		if tabHasAgent(agents, origin.TabID, agent) {
			return agent
		}
	}
	return ""
}

func tabHasAgent(agents []agentRow, tabID, name string) bool {
	if tabID == "" {
		return false
	}
	for _, agent := range agents {
		if agent.TabID == tabID && agent.Agent == name {
			return true
		}
	}
	return false
}
