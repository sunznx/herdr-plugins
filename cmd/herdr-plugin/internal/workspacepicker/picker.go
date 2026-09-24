package workspacepicker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
	sharedzoxide "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/zoxide"
)

type Kind string

const (
	Scratch   Kind = "scratch"
	Workspace Kind = "workspace"
	Directory Kind = "directory"
	Named     Kind = "named"
)

type Choice struct {
	Kind        Kind
	WorkspaceID string
	Label       string
	Path        string
}

type PickOptions struct {
	HerdrBin        string
	Prompt          string
	KeepWorkspaceID string
	Limit           int
	Choice          string
	CandidatesFile  string
	CreateMissing   bool
	CreateCWD       string
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
		token, err = pickWithFZF(ctx, rows, opts.Prompt, candidates)
		if err != nil {
			return nil, err
		}
		if token == "" {
			return nil, nil
		}
	}
	for _, item := range candidates {
		fields := strings.Split(item.row, "\t")
		if item.token == token || (len(fields) > 2 && (fields[2] == token || (len(fields) > 3 && fields[3] == token))) {
			choice := item.choice
			return &choice, nil
		}
	}
	if opts.CreateMissing && token != "" && opts.CreateCWD != "" && !strings.HasPrefix(token, "__") && !strings.HasPrefix(token, "/") {
		return &Choice{Kind: Named, Label: token, Path: opts.CreateCWD}, nil
	}
	if opts.CreateMissing && filepath.IsAbs(token) {
		if path, ok := physicalDir(token); ok {
			return &Choice{Kind: Directory, Path: path}, nil
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
	for _, workspace := range workspaces.Result.Workspaces {
		if workspace.Label == "scratch" {
			continue
		}
		token := "__workspace__:" + workspace.WorkspaceID
		items = append(items, candidate{
			token:  token,
			row:    strings.Join([]string{token, "[GOTO]", workspace.Label, ""}, "\t"),
			choice: Choice{Kind: Workspace, WorkspaceID: workspace.WorkspaceID},
		})
	}
	gitBin, gitErr := exec.LookPath("git")
	if gitErr != nil {
		return items, nil
	}

	workspaceRoot := make(map[string]string)
	for _, pane := range panes.Result.Panes {
		root, ok := physicalDir(pane.CWD)
		if !ok || pane.WorkspaceID == "" {
			continue
		}
		if _, exists := workspaceRoot[pane.WorkspaceID]; !exists {
			workspaceRoot[pane.WorkspaceID] = root
		}
	}
	seen := make(map[string]bool)
	count := 0
	for i := range items {
		if items[i].choice.Kind != Workspace {
			continue
		}
		root := workspaceRoot[items[i].choice.WorkspaceID]
		items[i].choice.Path = root
		fields := strings.Split(items[i].row, "\t")
		if len(fields) > 3 {
			fields[3] = root
			items[i].row = strings.Join(fields, "\t")
		}
		if root != "" {
			seen[root] = true
		}
	}

	paths, err := sharedzoxide.List(ctx)
	if err != nil {
		return items, nil
	}
	for scanned, rawPath := range paths {
		if scanned >= 500 || count >= opts.Limit {
			break
		}
		path, ok := physicalDir(rawPath)
		if !ok || seen[path] {
			continue
		}
		if root, err := gitRoot(ctx, gitBin, path); err == nil && root != path {
			continue
		}
		seen[path] = true
		state := "[NEW]"
		items = append(items, candidate{
			token:  path,
			row:    strings.Join([]string{path, state, filepath.Base(path), path}, "\t"),
			choice: Choice{Kind: Directory, Path: path},
		})
		count++
	}
	preferWorkspace(items, opts.KeepWorkspaceID)
	return items, nil
}

func preferWorkspace(items []candidate, workspaceID string) {
	if workspaceID == "" || len(items) < 2 || items[0].choice.WorkspaceID == workspaceID {
		return
	}
	for i := 1; i < len(items); i++ {
		if items[i].choice.Kind == Workspace && items[i].choice.WorkspaceID == workspaceID {
			item := items[i]
			copy(items[1:i+1], items[0:i])
			items[0] = item
			return
		}
	}
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

func pickWithFZF(ctx context.Context, rows, prompt string, candidates []candidate) (string, error) {
	selected, err := sharedfzf.Pick(ctx, rows, workspaceFZFArgs(prompt)...)
	if err != nil {
		return "", err
	}
	if selected == "" {
		return "", nil
	}
	return resolveFZFSelection(selected, candidates), nil
}

func workspaceFZFArgs(prompt string) []string {
	return []string{
		"--delimiter=\t", "--with-nth=2..", "--prompt=" + prompt, "--print-query",
		"--header=[GOTO] switch · [NEW] create · [SCRATCH] temporary",
		"--reverse", "--cycle", "--no-multi", "--tiebreak=begin,index",
	}
}

func parseFZFSelection(selected string) string {
	lines := strings.Split(strings.TrimRight(selected, "\n"), "\n")
	if len(lines) == 1 && strings.Contains(lines[0], "\t") {
		return strings.SplitN(lines[0], "\t", 2)[0]
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
		return strings.SplitN(lines[1], "\t", 2)[0]
	}
	return strings.TrimSpace(lines[0])
}

func resolveFZFSelection(selected string, candidates []candidate) string {
	token := parseFZFSelection(selected)
	lines := strings.Split(strings.TrimRight(selected, "\n"), "\n")
	if len(lines) < 2 || !strings.HasPrefix(token, "__workspace__:") {
		return token
	}
	query := strings.TrimSpace(lines[0])
	for _, item := range candidates {
		if item.choice.Kind == Directory && filepath.Base(item.choice.Path) == query {
			return item.token
		}
	}
	return token
}
