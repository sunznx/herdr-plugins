package workspacepicker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
)

func TestScratchUsesExistingWorkspaceWithoutZoxide(t *testing.T) {
	tmp := t.TempDir()
	herdr := filepath.Join(tmp, "herdr")
	script := `#!/bin/sh
case "$1 $2" in
  "pane list") printf '%s\n' '{"result":{"panes":[]}}' ;;
  "workspace list") printf '%s\n' '{"result":{"workspaces":[{"workspace_id":"w2","label":"scratch"}]}}' ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(herdr, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	choice, err := Pick(context.Background(), PickOptions{
		HerdrBin: herdr,
		Choice:   "__scratch__",
	})
	if err != nil {
		t.Fatal(err)
	}
	if choice == nil || choice.Kind != Scratch || choice.WorkspaceID != "w2" {
		t.Fatalf("unexpected choice: %#v", choice)
	}
}

func TestPreferWorkspaceMakesCurrentWorkspaceDefault(t *testing.T) {
	items := []candidate{
		{token: "__scratch__", choice: Choice{Kind: Scratch}},
		{token: "__workspace__:w1", choice: Choice{Kind: Workspace, WorkspaceID: "w1"}},
		{token: "__workspace__:w2", choice: Choice{Kind: Workspace, WorkspaceID: "w2"}},
	}
	preferWorkspace(items, "w2")
	if got := items[0].choice.WorkspaceID; got != "w2" {
		t.Fatalf("default workspace = %q, want w2", got)
	}
}

func TestMissingChoiceBecomesNamedWorkspace(t *testing.T) {
	tmp := t.TempDir()
	herdr := filepath.Join(tmp, "herdr")
	script := `#!/bin/sh
case "$1 $2" in
  "pane list") printf '%s\n' '{"result":{"panes":[]}}' ;;
  "workspace list") printf '%s\n' '{"result":{"workspaces":[]}}' ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(herdr, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)
	choice, err := Pick(context.Background(), PickOptions{
		HerdrBin:      herdr,
		Choice:        "gitlab",
		CreateMissing: true,
		CreateCWD:     "/tmp/source",
	})
	if err != nil {
		t.Fatal(err)
	}
	if choice == nil || choice.Kind != Named || choice.Label != "gitlab" || choice.Path != "/tmp/source" {
		t.Fatalf("unexpected choice: %#v", choice)
	}
}

func TestExistingWorkspaceMatchesByLabel(t *testing.T) {
	tmp := t.TempDir()
	herdr := filepath.Join(tmp, "herdr")
	script := `#!/bin/sh
case "$1 $2" in
  "pane list") echo '{"result":{"panes":[]}}' ;;
  "workspace list") echo '{"result":{"workspaces":[{"workspace_id":"w2","label":"registry"}]}}' ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(herdr, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)
	choice, err := Pick(context.Background(), PickOptions{HerdrBin: herdr, Choice: "registry", CreateMissing: true, CreateCWD: "/tmp/source"})
	if err != nil {
		t.Fatal(err)
	}
	if choice == nil || choice.Kind != Workspace || choice.WorkspaceID != "w2" {
		t.Fatalf("unexpected choice: %#v", choice)
	}
}

func TestWorkspacePathUsesFirstPaneOnly(t *testing.T) {
	tmp := t.TempDir()
	first := filepath.Join(tmp, "first")
	second := filepath.Join(tmp, "second")
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatal(err)
	}
	herdr := filepath.Join(tmp, "herdr")
	git := filepath.Join(tmp, "git")
	if err := os.WriteFile(herdr, []byte(`#!/bin/sh
case "$1 $2" in
  "pane list") printf '%s\n' '{"result":{"panes":[{"workspace_id":"w2","cwd":"`+first+`"},{"workspace_id":"w2","cwd":"`+second+`"}]}}' ;;
  "workspace list") printf '%s\n' '{"result":{"workspaces":[{"workspace_id":"w2","label":"registry"}]}}' ;;
  *) exit 2 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(git, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)
	choice, err := Pick(context.Background(), PickOptions{HerdrBin: herdr, Choice: second, CreateMissing: true, CreateCWD: first})
	if err != nil {
		t.Fatal(err)
	}
	resolvedSecond, _ := filepath.EvalSymlinks(second)
	if choice == nil || choice.Kind != Directory || choice.Path != resolvedSecond {
		t.Fatalf("second pane path must be new directory choice: %#v", choice)
	}
	resolvedFirst, _ := filepath.EvalSymlinks(first)
	choice, err = Pick(context.Background(), PickOptions{HerdrBin: herdr, Choice: resolvedFirst, CreateMissing: true, CreateCWD: second})
	if err != nil {
		t.Fatal(err)
	}
	if choice == nil || choice.Kind != Workspace || choice.WorkspaceID != "w2" {
		t.Fatalf("first pane path must goto workspace: %#v", choice)
	}
}

func TestAbsoluteChoiceBecomesDirectory(t *testing.T) {
	tmp := t.TempDir()
	herdr := filepath.Join(tmp, "herdr")
	if err := os.WriteFile(herdr, []byte("#!/bin/sh\ncase \"$1 $2\" in\n  \"pane list\") echo '{\"result\":{\"panes\":[]}}' ;;\n  \"workspace list\") echo '{\"result\":{\"workspaces\":[]}}' ;;\n  *) exit 2 ;;\nesac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)
	choice, err := Pick(context.Background(), PickOptions{HerdrBin: herdr, Choice: tmp, CreateMissing: true, CreateCWD: tmp})
	want, _ := filepath.EvalSymlinks(tmp)
	if err != nil || choice == nil || choice.Kind != Directory || choice.Path != want {
		t.Fatalf("unexpected choice: %#v, err=%v", choice, err)
	}
}

func TestFZFEnvironmentUsesConfigFallback(t *testing.T) {
	configHome := t.TempDir()
	config := filepath.Join(configHome, "fzf", "fzfrc")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("FZF_DEFAULT_OPTS_FILE", "")

	want := "FZF_DEFAULT_OPTS_FILE=" + config
	for _, item := range sharedfzf.Environment() {
		if item == want {
			return
		}
	}
	t.Fatalf("%s missing from environment: %s", want, strings.Join(sharedfzf.Environment(), "\n"))
}

func TestParseFZFSelectionSupportsNewWorkspaceQuery(t *testing.T) {
	if got := parseFZFSelection("gitlab\n"); got != "gitlab" {
		t.Fatalf("query selection = %q", got)
	}
	for name, input := range map[string]string{
		"scratch":   "__scratch__\t[SCRATCH]\tscratch\tTemporary workspace\n",
		"workspace": "__workspace__:w2\t[GOTO]\tregistry\t/root/registry\n",
		"directory": "/repo/registry\t[NEW]\tregistry\t/repo/registry\n",
	} {
		t.Run(name, func(t *testing.T) {
			if got, want := parseFZFSelection(input), strings.SplitN(input, "\t", 2)[0]; got != want {
				t.Fatalf("single-row selection = %q, want %q", got, want)
			}
		})
	}
	if got := parseFZFSelection("git\n/repo/gitlab\t[NEW]\tgitlab\n"); got != "/repo/gitlab" {
		t.Fatalf("row selection = %q", got)
	}
}

func TestResolveFZFSelectionHandlesSingleRowWorkspace(t *testing.T) {
	candidates := []candidate{{
		token: "__workspace__:w2",
		row:   "__workspace__:w2\t[GOTO]\tregistry\t/root/registry",
		choice: Choice{
			Kind:        Workspace,
			WorkspaceID: "w2",
		},
	}}
	if got := resolveFZFSelection("__workspace__:w2\t[GOTO]\tregistry\t/root/registry\n", candidates); got != "__workspace__:w2" {
		t.Fatalf("single-row workspace selection = %q", got)
	}
}

func TestFZFQueryPrefersExactDirectoryOverFuzzyWorkspaceLabel(t *testing.T) {
	candidates := []candidate{
		{token: "__workspace__:wR", row: "__workspace__:wR\t[GOTO]\tgitlab.miniscloud.com\t/root", choice: Choice{Kind: Workspace, WorkspaceID: "wR"}},
		{token: "/root/devops/gitlab", row: "/root/devops/gitlab\t[NEW]\tgitlab\t/root/devops/gitlab", choice: Choice{Kind: Directory, Path: "/root/devops/gitlab"}},
	}
	got := resolveFZFSelection("gitlab\n__workspace__:wR\t[GOTO]\tgitlab.miniscloud.com\t/root\n", candidates)
	if got != "/root/devops/gitlab" {
		t.Fatalf("selection = %q, want exact directory token", got)
	}
}

func TestWorkspaceFZFArgsKeepSearchAndPrintQuery(t *testing.T) {
	args := strings.Join(workspaceFZFArgs("workspace ▸ "), "\x00")
	if !strings.Contains(args, "--print-query") {
		t.Fatal("workspace picker must preserve query-only input")
	}
	if strings.Contains(args, "--phony") {
		t.Fatal("workspace picker must keep normal fzf filtering")
	}
}
