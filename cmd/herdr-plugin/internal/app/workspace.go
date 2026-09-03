package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/workspacepicker"
)

func pickWorkspace(ctx context.Context, c herdr.Client, prompt, keep, choiceEnv, candidatesEnv string) (*workspacepicker.Choice, error) {
	return workspacepicker.Pick(ctx, workspacepicker.PickOptions{
		HerdrBin:        c.Bin,
		Prompt:          prompt,
		KeepWorkspaceID: keep,
		Limit:           100,
		Choice:          os.Getenv(choiceEnv),
		CandidatesFile:  os.Getenv(candidatesEnv),
	})
}

func createWorkspace(ctx context.Context, c herdr.Client, choice workspacepicker.Choice, focus bool) (workspaceID, paneID, cwd string, err error) {
	if choice.Kind == workspacepicker.Workspace || (choice.Kind == workspacepicker.Scratch && choice.WorkspaceID != "") {
		return choice.WorkspaceID, "", choice.Path, nil
	}
	cwd = choice.Path
	args := []string{"workspace", "create"}
	if choice.Kind == workspacepicker.Scratch {
		cwd, err = os.MkdirTemp("", "herdr-scratch-")
		if err != nil {
			return "", "", "", err
		}
		args = append(args, "--label", "scratch")
	} else if choice.Kind != workspacepicker.Directory {
		return "", "", "", fmt.Errorf("invalid workspace choice %q", choice.Kind)
	}
	args = append(args, "--cwd", cwd)
	if focus {
		args = append(args, "--focus")
	} else {
		args = append(args, "--no-focus")
	}
	out, err := c.Run(ctx, args...)
	if err != nil {
		return "", "", "", err
	}
	if c.DryRun {
		return "<new-workspace>", "<new-root-pane>", cwd, nil
	}
	var response struct {
		Result struct {
			Workspace struct {
				WorkspaceID string `json:"workspace_id"`
			} `json:"workspace"`
			RootPane herdr.Pane `json:"root_pane"`
		} `json:"result"`
	}
	if err := decode(out, &response); err != nil {
		return "", "", "", err
	}
	workspaceID = response.Result.Workspace.WorkspaceID
	paneID = response.Result.RootPane.PaneID
	if workspaceID == "" {
		workspaceID = response.Result.RootPane.WorkspaceID
	}
	if workspaceID == "" && strings.Contains(paneID, ":") {
		workspaceID = strings.SplitN(paneID, ":", 2)[0]
	}
	if workspaceID == "" || paneID == "" {
		return "", "", "", fmt.Errorf("Herdr did not return the new workspace")
	}
	return workspaceID, paneID, cwd, nil
}
