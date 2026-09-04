package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func TestMain(m *testing.M) {
	if path := os.Getenv("HERDR_GO_TEST_ASYNC_CALL"); path != "" {
		_ = os.WriteFile(path, []byte(strings.Join(os.Args[1:], " ")+"\n"+os.Getenv(aiRenameWorkerEnv)), 0o600)
		os.Exit(0)
	}
	if os.Getenv("HERDR_GO_TEST_HELPER") == "1" {
		herdrTestHelper(os.Args[1:])
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestAIRenameStartsDetachedWorker(t *testing.T) {
	call := filepath.Join(t.TempDir(), "call")
	t.Setenv("HERDR_GO_TEST_ASYNC_CALL", call)
	if err := startAIRename(false); err != nil {
		t.Fatal(err)
	}
	var data []byte
	for range 100 {
		data, _ = os.ReadFile(call)
		if len(data) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got, want := string(data), "rename spawn ai-current\n1"; got != want {
		t.Fatalf("detached worker call: got %q, want %q", got, want)
	}
}

func herdrTestHelper(args []string) {
	command := strings.Join(args, " ")
	if path := os.Getenv("HERDR_GO_TEST_CALLS"); path != "" {
		if file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			fmt.Fprintln(file, command)
			_ = file.Close()
		}
	}
	switch {
	case command == "plugin config-dir sunznx.command-palette-popup":
		fmt.Println(os.Getenv("HERDR_GO_TEST_CONFIG"))
	case strings.HasPrefix(command, "plugin pane open --plugin sunznx.yazi-popup --entrypoint fzf --placement popup --focus --env HERDR_YAZI_CWD="):
		fmt.Println(`{"result":{}}`)
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
	case command == "pane get w1:p1":
		fmt.Println(`{"result":{"pane":{"pane_id":"w1:p1","agent":"codex","agent_status":"idle"}}}`)
	case command == "pane wait-output w1:p1 --match Archive this session? --source visible --timeout 10000":
		fmt.Println(`{"result":{"matched":true}}`)
	case command == "agent wait w1:p1 --until unknown --timeout 10000":
		fmt.Println(`{"result":{"agent":{"pane_id":"w1:p1","agent_status":"unknown"}}}`)
	case command == "agent prompt w1:p1 /archive", command == "agent send-keys w1:p1 down enter", command == "pane close w1:p1":
		fmt.Println(`{"result":{}}`)
	default:
		fmt.Fprintf(os.Stderr, "unexpected fake herdr command: %s\n", command)
		os.Exit(2)
	}
}

func TestYaziOpenPassesCWDAsPopupEnv(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_PLUGIN_ID", "sunznx.yazi-popup")
	cwd := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", fmt.Sprintf(`{"focused_pane_cwd":%q}`, cwd))
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	if err := yaziOpen(context.Background(), herdr.New(), "fzf"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	row := strings.TrimSpace(string(data))
	if !strings.Contains(row, "--env HERDR_YAZI_CWD="+cwd) {
		t.Fatalf("missing popup cwd env in Herdr call: %s", row)
	}
	if strings.Contains(row, "--cwd") {
		t.Fatalf("popup cwd must not change binary resolution: %s", row)
	}
}

func TestCloseCodexUsesServerSideWaits(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"focused_pane_id":"w1:p1"}`)
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	if err := closeCodex(context.Background(), herdr.New()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	want := "pane get w1:p1\n" +
		"agent prompt w1:p1 /archive\n" +
		"pane wait-output w1:p1 --match Archive this session? --source visible --timeout 10000\n" +
		"agent send-keys w1:p1 down enter\n" +
		"agent wait w1:p1 --until unknown --timeout 10000\n" +
		"pane close w1:p1\n"
	if string(data) != want {
		t.Fatalf("unexpected Herdr calls:\n%s", data)
	}
}

func TestPaletteUsesSharedRuntimeRows(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_GO_TEST_CONFIG", t.TempDir())
	items, _, err := buildPalette(context.Background(), herdr.New(), herdr.TargetContext{Pane: "w1:p1", Tab: "w1:t1", Workspace: "w1"})
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
		"popup-zoxide":                   {`["./herdr-plugin", "zoxide", "picker"]`, `["./herdr-plugin", "zoxide", "open"]`},
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
		for _, forbidden := range []string{`["bash"`, `["sh"`, `.sh"`, `workspace-picker`, `trellis`} {
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
