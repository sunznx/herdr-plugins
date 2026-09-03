package workspacepicker

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

type Kind string

const (
	Scratch   Kind = "scratch"
	Workspace Kind = "workspace"
	Directory Kind = "directory"
)

type Choice struct {
	Kind        Kind
	WorkspaceID string
	Path        string
}

type PickOptions struct {
	HerdrBin        string
	Prompt          string
	KeepWorkspaceID string
	Limit           int
	Choice          string
	CandidatesFile  string
}

type candidate struct {
	token  string
	row    string
	choice Choice
}

type paneList struct {
	Result struct {
		Panes []struct {
			WorkspaceID string `json:"workspace_id"`
			CWD         string `json:"cwd"`
		} `json:"panes"`
	} `json:"result"`
}

type workspaceList struct {
	Result struct {
		Workspaces []struct {
			WorkspaceID string `json:"workspace_id"`
			Label       string `json:"label"`
		} `json:"workspaces"`
	} `json:"result"`
}

func Pick(ctx context.Context, opts PickOptions) (*Choice, error) {
	if opts.HerdrBin == "" {
		opts.HerdrBin = envOr("HERDR_BIN_PATH", "herdr")
	}
	if opts.Prompt == "" {
		opts.Prompt = "workspace ▸ "
	}
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	candidates, err := loadCandidates(ctx, opts)
	if err != nil {
		return nil, err
	}
	rows := candidateRows(candidates)
	if opts.CandidatesFile != "" {
		if err := os.WriteFile(opts.CandidatesFile, []byte(rows), 0o600); err != nil {
			return nil, fmt.Errorf("write candidates: %w", err)
		}
	}

	token := opts.Choice
	if token == "" {
		token, err = pickWithFZF(ctx, rows, opts.Prompt)
		if err != nil {
			return nil, err
		}
		if token == "" {
			return nil, nil
		}
	}
	for _, item := range candidates {
		if item.token == token {
			choice := item.choice
			return &choice, nil
		}
	}
	return nil, fmt.Errorf("selected workspace %q is unavailable", token)
}

func loadCandidates(ctx context.Context, opts PickOptions) ([]candidate, error) {
	var panes paneList
	if err := commandJSON(ctx, opts.HerdrBin, []string{"pane", "list"}, &panes); err != nil {
		return nil, fmt.Errorf("list panes: %w", err)
	}
	var workspaces workspaceList
	if err := commandJSON(ctx, opts.HerdrBin, []string{"workspace", "list"}, &workspaces); err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}

	items := []candidate{{
		token:  "__scratch__",
		row:    "__scratch__\t[SCRATCH]\tscratch\tTemporary workspace",
		choice: Choice{Kind: Scratch},
	}}
	labels := make(map[string]string)
	for _, workspace := range workspaces.Result.Workspaces {
		labels[workspace.WorkspaceID] = workspace.Label
		if workspace.Label == "scratch" && items[0].choice.WorkspaceID == "" {
			items[0].choice.WorkspaceID = workspace.WorkspaceID
		}
	}
	gitBin, gitErr := exec.LookPath("git")
	zoxideBin, zoxideErr := exec.LookPath("zoxide")
	if gitErr != nil || zoxideErr != nil {
		return items, nil
	}

	rootWorkspace := make(map[string]string)
	workspaceRoot := make(map[string]string)
	for _, pane := range panes.Result.Panes {
		root, ok := canonicalRoot(ctx, gitBin, pane.CWD)
		if !ok || pane.WorkspaceID == "" {
			continue
		}
		if _, exists := rootWorkspace[root]; !exists {
			rootWorkspace[root] = pane.WorkspaceID
		}
		if _, exists := workspaceRoot[pane.WorkspaceID]; !exists {
			workspaceRoot[pane.WorkspaceID] = root
		}
	}
	seen := make(map[string]bool)
	count := 0
	if root := workspaceRoot[opts.KeepWorkspaceID]; root != "" && labels[opts.KeepWorkspaceID] != "scratch" {
		label := labels[opts.KeepWorkspaceID]
		if label == "" {
			label = filepath.Base(root)
		}
		token := "__workspace__:" + opts.KeepWorkspaceID
		items = append(items, candidate{
			token:  token,
			row:    strings.Join([]string{token, "[GOTO]", label, root}, "\t"),
			choice: Choice{Kind: Workspace, WorkspaceID: opts.KeepWorkspaceID, Path: root},
		})
		seen[root] = true
		count++
	}

	out, err := exec.CommandContext(ctx, zoxideBin, "query", "-l").Output()
	if err != nil {
		return items, nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanned := 0; scanner.Scan() && scanned < 500 && count < opts.Limit; scanned++ {
		path, ok := physicalDir(scanner.Text())
		if !ok || seen[path] {
			continue
		}
		if root, err := gitRoot(ctx, gitBin, path); err == nil && root != path {
			continue
		}
		seen[path] = true
		workspaceID := rootWorkspace[path]
		state := "[NEW]"
		kind := Directory
		if workspaceID != "" {
			state = "[GOTO]"
			kind = Workspace
		}
		items = append(items, candidate{
			token:  path,
			row:    strings.Join([]string{path, state, filepath.Base(path), path}, "\t"),
			choice: Choice{Kind: kind, WorkspaceID: workspaceID, Path: path},
		})
		count++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read zoxide output: %w", err)
	}
	return items, nil
}

func canonicalRoot(ctx context.Context, gitBin, path string) (string, bool) {
	physical, ok := physicalDir(path)
	if !ok {
		return "", false
	}
	if root, err := gitRoot(ctx, gitBin, physical); err == nil {
		return root, true
	}
	return physical, true
}

func physicalDir(path string) (string, bool) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		path = home
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return resolved, true
}

func gitRoot(ctx context.Context, gitBin, path string) (string, error) {
	out, err := exec.CommandContext(ctx, gitBin, "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	root, ok := physicalDir(strings.TrimSpace(string(out)))
	if !ok {
		return "", errors.New("git returned an unavailable root")
	}
	return root, nil
}

func commandJSON(ctx context.Context, bin string, args []string, dst any) error {
	return (herdr.Client{Bin: bin}).JSON(ctx, dst, args...)
}

func candidateRows(candidates []candidate) string {
	rows := make([]string, 0, len(candidates))
	for _, item := range candidates {
		rows = append(rows, item.row)
	}
	return strings.Join(rows, "\n") + "\n"
}

func pickWithFZF(ctx context.Context, rows, prompt string) (string, error) {
	selected, err := sharedfzf.Pick(ctx, rows,
		"--delimiter=\t", "--with-nth=2..", "--prompt="+prompt,
		"--header=[GOTO] switch · [NEW] create · [SCRATCH] temporary",
		"--reverse", "--cycle", "--no-multi", "--tiebreak=begin,index",
	)
	if err != nil {
		return "", err
	}
	if selected == "" {
		return "", nil
	}
	return strings.SplitN(selected, "\t", 2)[0], nil
}
