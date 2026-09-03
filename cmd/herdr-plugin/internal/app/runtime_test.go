package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func TestMain(m *testing.M) {
	if os.Getenv("HERDR_GO_TEST_HELPER") == "1" {
		herdrTestHelper(os.Args[1:])
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func herdrTestHelper(args []string) {
	command := strings.Join(args, " ")
	switch {
	case command == "plugin config-dir sunznx.command-palette-popup":
		fmt.Println(os.Getenv("HERDR_GO_TEST_CONFIG"))
	case command == "--default-config":
		fmt.Println("[keys]\n# prefix = \"ctrl+b\"\n# new_tab = \"prefix+c\"")
	case command == "plugin action list":
		fmt.Println(`{"result":{"actions":[{"plugin_id":"sunznx.herdr-move","action_id":"open","title":"Move pane to workspace"},{"plugin_id":"sunznx.herdr-new-codex","action_id":"codex","title":"New Codex tab"}]}}`)
	case strings.HasPrefix(command, "tab list"):
		fmt.Println(`{"result":{"tabs":[{"tab_id":"w1:t1","workspace_id":"w1","number":1,"label":"current"},{"tab_id":"w1:t2","workspace_id":"w1","number":2,"label":"logs"}]}}`)
	case command == "workspace list":
		fmt.Println(`{"result":{"workspaces":[{"workspace_id":"w1","label":"repo"}]}}`)
	case command == "agent list":
		fmt.Println(`{"result":{"agents":[]}}`)
	default:
		fmt.Fprintf(os.Stderr, "unexpected fake herdr command: %s\n", command)
		os.Exit(2)
	}
}

func TestPaletteUsesSharedRuntimeRows(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_GO_TEST_CONFIG", t.TempDir())
	items, _, err := buildPalette(context.Background(), herdr.New(), paletteContext{Pane: "w1:p1", Tab: "w1:t1", Workspace: "w1"})
	if err != nil {
		t.Fatal(err)
	}
	rows := renderPalette(items)
	for _, want := range []string{"static\tnew_workspace\t", "static\tmove_pane_workspace\t", "plugin\tsunznx.herdr-new-codex.codex\t"} {
		if !strings.Contains(rows, want) {
			t.Fatalf("missing %q in palette rows:\n%s", want, rows)
		}
	}
	for _, unwanted := range []string{"Plugin:", "goto_workspace", "plugin\tsunznx.herdr-move.open\t"} {
		if strings.Contains(rows, unwanted) {
			t.Fatalf("unexpected %q in palette rows", unwanted)
		}
	}
}

func TestAllManifestsBuildAndRunGoRuntime(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	plugins := map[string][]string{
		"command-palette-popup":          {`["./herdr-plugin", "palette", "open"]`, `["./herdr-plugin", "palette", "run"]`},
		"gitui-popup":                    {`["./herdr-plugin", "gitui", "open"]`},
		"herdr-ai-rename":                {`["./herdr-plugin", "rename", "ai-current"]`, `["./herdr-plugin", "rename", "picker"]`},
		"herdr-copy-last-command-output": {`["./herdr-plugin", "copy", "output"]`, `["./herdr-plugin", "copy", "command-and-output"]`},
		"herdr-move":                     {`["./herdr-plugin", "move", "open"]`, `["./herdr-plugin", "move", "workspace"]`, `["./herdr-plugin", "move", "tab"]`},
		"herdr-new-codex":                {`["./herdr-plugin", "new-codex", "open"]`, `["./herdr-plugin", "new-codex", "picker"]`, `["./herdr-plugin", "new-codex", "close"]`},
		"mole-current-dir":               {`["./herdr-plugin", "mole", "open"]`},
		"open-in":                        {`["./herdr-plugin", "open-in", "open", "Emacs"]`},
		"yazi-popup":                     {`["./herdr-plugin", "yazi", "picker"]`, `["./herdr-plugin", "yazi", "browser", "rg"]`},
	}
	for plugin, commands := range plugins {
		data, err := os.ReadFile(filepath.Join(root, plugin, "herdr-plugin.toml"))
		if err != nil {
			t.Fatal(err)
		}
		manifest := string(data)
		if !strings.Contains(manifest, `command = ["go", "build", "-trimpath", "-o", "herdr-plugin", "../cmd/herdr-plugin"]`) {
			t.Errorf("%s does not build the shared Go runtime", plugin)
		}
		for _, forbidden := range []string{`["bash"`, `["sh"`, `.sh"`, `workspace-picker`} {
			if strings.Contains(manifest, forbidden) {
				t.Errorf("%s still contains runtime shell reference %q", plugin, forbidden)
			}
		}
		for _, command := range commands {
			if !strings.Contains(manifest, command) {
				t.Errorf("%s is missing runtime route %s", plugin, command)
			}
		}
	}
}
