package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func moveOpen(ctx context.Context, c herdr.Client, toTab bool) error {
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	entrypoint := "picker"
	if toTab {
		entrypoint = "tab-picker"
	}
	return c.OpenPane(ctx, envOr("HERDR_PLUGIN_ID", "sunznx.herdr-move"), entrypoint, true, "",
		"HERDR_MOVE_PANE_ID="+pane.PaneID,
		"HERDR_MOVE_WORKSPACE_ID="+pane.WorkspaceID,
		"HERDR_MOVE_TAB_ID="+pane.TabID)
}

func moveWorkspace(ctx context.Context, c herdr.Client) error {
	paneID := os.Getenv("HERDR_MOVE_PANE_ID")
	keep := os.Getenv("HERDR_MOVE_WORKSPACE_ID")
	if paneID == "" || keep == "" {
		return fmt.Errorf("source pane or workspace ID is missing")
	}
	pane, err := c.GetPane(ctx, paneID)
	if err != nil || pane.PaneID != paneID {
		return fmt.Errorf("the source pane is no longer available")
	}
	choiceEnv := "HERDR_MOVE_CHOICE"
	if selectedID := os.Getenv(choiceEnv); selectedID != "" && !strings.HasPrefix(selectedID, "__") && !strings.HasPrefix(selectedID, "/") {
		var response struct {
			Result struct {
				Workspaces []workspaceRow `json:"workspaces"`
			} `json:"result"`
		}
		if err := c.JSON(ctx, &response, "workspace", "list"); err != nil {
			return err
		}
		for _, workspace := range response.Result.Workspaces {
			if workspace.WorkspaceID == selectedID {
				out, err := c.Run(ctx, "pane", "move", paneID, "--new-tab", "--workspace", selectedID, "--focus")
				if err == nil {
					fmt.Printf("Moved pane to %s (%s)\n", selectedID, movedPaneID(out))
				}
				return err
			}
		}
		return fmt.Errorf("destination workspace %q is unavailable", selectedID)
	}
	choice, err := pickWorkspace(ctx, c, "move pane to workspace ▸ ", keep, choiceEnv, "HERDR_MOVE_CANDIDATES_FILE")
	if err != nil || choice == nil {
		return err
	}
	target, bootstrap, _, err := createWorkspace(ctx, c, *choice, false)
	if err != nil {
		return err
	}
	out, err := c.Run(ctx, "pane", "move", paneID, "--new-tab", "--workspace", target, "--focus")
	if err != nil {
		return err
	}
	if bootstrap != "" {
		if _, err := c.Run(ctx, "pane", "close", bootstrap); err != nil {
			return err
		}
	}
	fmt.Printf("Moved pane to %s (%s)\n", target, movedPaneID(out))
	return nil
}

type workspaceRow struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

type tabRow struct {
	TabID       string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	Number      int    `json:"number"`
	Label       string `json:"label"`
}

func moveTab(ctx context.Context, c herdr.Client) error {
	paneID, sourceTab := os.Getenv("HERDR_MOVE_PANE_ID"), os.Getenv("HERDR_MOVE_TAB_ID")
	if paneID == "" || sourceTab == "" {
		return fmt.Errorf("source pane or tab ID is missing")
	}
	pane, err := c.GetPane(ctx, paneID)
	if err != nil || pane.PaneID != paneID || pane.TabID != sourceTab {
		return fmt.Errorf("the source pane is no longer in its original tab")
	}
	var workspaces struct {
		Result struct {
			Workspaces []workspaceRow `json:"workspaces"`
		} `json:"result"`
	}
	var tabs struct {
		Result struct {
			Tabs []tabRow `json:"tabs"`
		} `json:"result"`
	}
	if err := c.JSON(ctx, &workspaces, "workspace", "list"); err != nil {
		return err
	}
	if err := c.JSON(ctx, &tabs, "tab", "list"); err != nil {
		return err
	}
	labels := map[string]string{}
	for _, workspace := range workspaces.Result.Workspaces {
		labels[workspace.WorkspaceID] = workspace.Label
	}
	available := map[string]bool{}
	var rows strings.Builder
	for _, tab := range tabs.Result.Tabs {
		if tab.TabID == sourceTab {
			continue
		}
		available[tab.TabID] = true
		label := labels[tab.WorkspaceID]
		if label == "" {
			label = tab.WorkspaceID
		}
		fmt.Fprintf(&rows, "%s\t%s / #%s %s\n", tab.TabID, label, strconv.Itoa(tab.Number), tab.Label)
	}
	if len(available) == 0 {
		return fmt.Errorf("no destination tab is available")
	}
	destination := os.Getenv("HERDR_MOVE_TAB_CHOICE")
	if destination == "" {
		selected, err := sharedfzf.Pick(ctx, rows.String(), "--delimiter=\t", "--with-nth=2", "--prompt=move pane to tab ▸ ", "--reverse", "--cycle", "--no-multi", "--tiebreak=begin,index")
		if err != nil || selected == "" {
			return err
		}
		destination = strings.SplitN(selected, "\t", 2)[0]
	}
	if !available[destination] {
		return fmt.Errorf("destination tab %q is unavailable", destination)
	}
	var target struct {
		Result struct {
			Tab tabRow `json:"tab"`
		} `json:"result"`
	}
	if err := c.JSON(ctx, &target, "tab", "get", destination); err != nil || target.Result.Tab.TabID != destination {
		return fmt.Errorf("destination tab %q is no longer available", destination)
	}
	pane, err = c.GetPane(ctx, paneID)
	if err != nil || pane.PaneID != paneID || pane.TabID != sourceTab {
		return fmt.Errorf("the source pane is no longer in its original tab")
	}
	out, err := c.Run(ctx, "pane", "move", paneID, "--tab", destination, "--split", "right", "--focus")
	if err != nil {
		return err
	}
	newPane := movedPaneID(out)
	if newPane == "" {
		return fmt.Errorf("Herdr did not return the moved pane ID")
	}
	verified, err := c.GetPane(ctx, newPane)
	if err != nil || verified.TabID != destination {
		return fmt.Errorf("the moved pane could not be verified in destination tab %q", destination)
	}
	fmt.Printf("Moved pane to %s (%s)\n", destination, newPane)
	return nil
}

func movedPaneID(data []byte) string {
	var response struct {
		Result struct {
			MoveResult struct {
				Pane herdr.Pane `json:"pane"`
			} `json:"move_result"`
		} `json:"result"`
	}
	_ = decode(data, &response)
	return response.Result.MoveResult.Pane.PaneID
}
