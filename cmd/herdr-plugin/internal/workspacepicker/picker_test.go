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
