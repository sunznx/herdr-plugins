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
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/workspacepicker"
)

func TestMain(m *testing.M) {
	if path := os.Getenv("HERDR_GO_TEST_ASYNC_CALL"); path != "" {
		file, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if file != nil {
			_, _ = file.WriteString(strings.Join(os.Args[1:], " ") + "\n" + os.Getenv(aiRenameWorkerEnv))
			if os.Getenv("HERDR_GO_TEST_RECORD_CONTEXT") == "1" {
				for _, key := range []string{"HERDR_PLUGIN_CONTEXT_JSON", "HERDR_PLUGIN_ID", "HERDR_PANE_ID", "HERDR_TAB_ID", "HERDR_WORKSPACE_ID"} {
					if _, ok := os.LookupEnv(key); ok {
						_, _ = file.WriteString("\n" + key + "=present")
					}
				}
			}
			_ = file.Close()
		}
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
	case os.Getenv("HERDR_TEST_MOVE") == "1" && strings.HasPrefix(command, "workspace create --cwd "):
		fmt.Println(`{"result":{"workspace":{"workspace_id":"w2"},"root_pane":{"pane_id":"w2:p1"}}}`)
	case os.Getenv("HERDR_TEST_MOVE") == "1" && strings.HasPrefix(command, "pane move w1:p0 --workspace w2 --focus"):
		fmt.Println(`{"result":{"move_result":{"pane":{"pane_id":"w2:p2"}}}}`)
	case os.Getenv("HERDR_TEST_MOVE") == "1" && strings.HasPrefix(command, "agent prompt w2:p2 /cd "):
		fmt.Println(`{"result":{}}`)
	case os.Getenv("HERDR_TEST_MOVE") == "1" && strings.HasPrefix(command, "pane run w2:p2 cd "):
		fmt.Println(`{"result":{}}`)
	case command == "plugin config-dir sunznx.command-palette-popup":
		fmt.Println(os.Getenv("HERDR_GO_TEST_CONFIG"))
	case strings.HasPrefix(command, "plugin pane open --plugin sunznx.yazi-popup --entrypoint fzf --placement popup --focus --env HERDR_YAZI_CWD="):
		fmt.Println(`{"result":{}}`)
	case command == "--default-config":
		fmt.Println("[keys]\n# prefix = \"ctrl+b\"\n# new_tab = \"prefix+c\"")
	case command == "plugin action list":
		fmt.Println(`{"result":{"actions":[{"plugin_id":"sunznx.herdr-move","action_id":"open","title":"Move pane to workspace"},{"plugin_id":"sunznx.herdr-new-codex","action_id":"codex","title":"New Codex tab in workspace"},{"plugin_id":"sunznx.herdr-new-codex","action_id":"tab","title":"New tab in workspace"}]}}`)
	case command == "pane list":
		fmt.Printf("{\"result\":{\"panes\":[{\"workspace_id\":\"w1\",\"cwd\":%q}]}}\n", os.Getenv("HERDR_TEST_REPO"))
	case command == "pane get w1:p0":
		fmt.Printf("{\"result\":{\"pane\":{\"pane_id\":\"w1:p0\",\"tab_id\":\"w1:t1\",\"workspace_id\":\"w1\",\"cwd\":%q}}}\n", os.Getenv("HERDR_TEST_REPO"))
	case strings.HasPrefix(command, "tab list"):
		fmt.Println(`{"result":{"tabs":[{"tab_id":"w1:t1","workspace_id":"w1","number":1,"label":"current"},{"tab_id":"w1:t2","workspace_id":"w1","number":2,"label":"logs"}]}}`)
	case command == "workspace list":
		fmt.Println(`{"result":{"workspaces":[{"workspace_id":"w1","label":"repo"}]}}`)
	case strings.HasPrefix(command, "tab create --workspace w1 --cwd "):
		fmt.Println(`{"result":{"root_pane":{"pane_id":"w1:p2"}}}`)
	case command == "tab create --workspace w1 --focus":
		fmt.Println(`{"result":{"root_pane":{"pane_id":"w1:p2"}}}`)
	case command == "pane run w1:p2 exec codex":
		fmt.Println(`{"result":{}}`)
	case command == "plugin action invoke sunznx.herdr-new-codex.codex":
		fmt.Println(`{"result":{"log":{"log_id":"test-codex-log","plugin_id":"sunznx.herdr-new-codex"}}}`)
	case command == "plugin log list --plugin sunznx.herdr-new-codex --limit 20":
		fmt.Println(`{"result":{"logs":[{"log_id":"test-codex-log","status":"succeeded"}]}}`)
	case os.Getenv("HERDR_TEST_POPUP_BUSY") == "1" && strings.HasPrefix(command, "plugin pane open --plugin sunznx.herdr-new-codex "):
		data, _ := os.ReadFile(os.Getenv("HERDR_GO_TEST_CALLS"))
		if strings.Count(string(data), "plugin pane open --plugin sunznx.herdr-new-codex ") <= 2 {
			fmt.Fprint(os.Stderr, `{"error":{"code":"ui_busy","message":"a popup pane is already open"}}`)
			os.Exit(1)
		}
		fmt.Println(`{"result":{}}`)
	case command == "agent list":
		if os.Getenv("HERDR_TEST_CODEX") == "1" {
			fmt.Println(`{"result":{"agents":[{"pane_id":"w1:p1","tab_id":"w1:t1","agent":"codex"}]}}`)
		} else {
			fmt.Println(`{"result":{"agents":[]}}`)
		}
	case command == "pane get w1:p1":
		fmt.Println(`{"result":{"pane":{"pane_id":"w1:p1","agent":"codex","agent_status":"idle"}}}`)
	case os.Getenv("HERDR_TEST_FORK") == "1" && command == "pane get w1:p-origin":
		fmt.Println(`{"result":{"pane":{"pane_id":"w1:p-origin","workspace_id":"w1","cwd":"/repo","agent":"codex","agent_session":{"source":"herdr:codex","agent":"codex","kind":"id","value":"s1"}}}}`)
	case os.Getenv("HERDR_TEST_FORK") == "1" && strings.HasPrefix(command, "tab create --workspace w1 --cwd /repo --focus"):
		fmt.Println(`{"result":{"root_pane":{"pane_id":"w1:p-fork"}}}`)
	case os.Getenv("HERDR_TEST_FORK") == "1" && command == "pane run w1:p2 codex fork s1":
		fmt.Println(`{"result":{}}`)
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

func TestNewCodexActionsDetachPicker(t *testing.T) {
	for _, tc := range []struct{ action, entrypoint string }{
		{"open", "picker"}, {"open-tab", "tab-picker"}, {"claude", "claude-picker"},
	} {
		t.Run(tc.action, func(t *testing.T) {
			call := filepath.Join(t.TempDir(), "call")
			t.Setenv("HERDR_GO_TEST_ASYNC_CALL", call)
			t.Setenv("CPP_DRY_RUN", "0")
			root := NewCommand()
			root.SetArgs([]string{"new-codex", tc.action})
			if err := root.Execute(); err != nil {
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
			if got, want := strings.TrimSpace(string(data)), "new-codex open-picker "+tc.entrypoint; got != want {
				t.Fatalf("detached picker = %q, want %q", got, want)
			}
		})
	}
}

func TestNewCodexPickerWaitsForPopupToClose(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_TEST_POPUP_BUSY", "1")
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	c := herdr.Client{Bin: os.Args[0]}
	if err := openNewCodexPicker(context.Background(), c, "claude-picker"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "plugin pane open --plugin sunznx.herdr-new-codex --entrypoint claude-picker --placement popup --focus"); got != 3 {
		t.Fatalf("picker open attempts = %d, want 3; calls: %s", got, data)
	}
}

func TestMovePlainPaneToNewWorkspace(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_TEST_MOVE", "1")
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	choice := workspacepicker.Choice{Kind: workspacepicker.Directory, Path: t.TempDir()}
	pane := herdr.Pane{PaneID: "w1:p0", WorkspaceID: "w1", CWD: t.TempDir()}
	if _, _, err := movePaneToWorkspace(context.Background(), herdr.New(), pane, choice); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	want := "workspace create --cwd " + choice.Path + " --no-focus\n" +
		"pane move w1:p0 --workspace w2 --focus\n" +
		"pane run w2:p2 cd " + choice.Path + "\n"
	if string(data) != want {
		t.Fatalf("unexpected Herdr calls:\n%s\nwant:\n%s", data, want)
	}
}

func TestMoveCodexPaneSendsCDAfterMove(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_TEST_MOVE", "1")
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	target := t.TempDir()
	pane := herdr.Pane{PaneID: "w1:p0", WorkspaceID: "w1", CWD: t.TempDir(), Agent: "codex"}
	if _, _, err := movePaneToWorkspace(context.Background(), herdr.New(), pane, workspacepicker.Choice{Kind: workspacepicker.Directory, Path: target}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	want := "workspace create --cwd " + target + " --no-focus\n" +
		"pane move w1:p0 --workspace w2 --focus\n" +
		"agent prompt w2:p2 /cd " + target + "\n"
	if string(data) != want {
		t.Fatalf("unexpected Herdr calls:\n%s\nwant:\n%s", data, want)
	}
}

func TestDuplicateTabWithoutCodex(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"focused_pane_id":"w1:p0"}`)
	cwd := t.TempDir()
	t.Setenv("HERDR_TEST_REPO", cwd)
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	if err := duplicateTabOrAgent(context.Background(), herdr.New()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	want := "pane get w1:p0\nagent list\ntab create --workspace w1 --cwd " + cwd + " --focus\n"
	if string(data) != want {
		t.Fatalf("unexpected Herdr calls:\n%swant:\n%s", data, want)
	}
}

func TestDuplicateTabStartsCodexWhenCurrentTabHasCodex(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_TEST_CODEX", "1")
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"focused_pane_id":"w1:p0"}`)
	cwd := t.TempDir()
	t.Setenv("HERDR_TEST_REPO", cwd)
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	if err := duplicateTabOrAgent(context.Background(), herdr.New()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	want := "pane get w1:p0\nagent list\ntab create --workspace w1 --cwd " + cwd + " --focus\npane run w1:p2 exec codex\n"
	if string(data) != want {
		t.Fatalf("unexpected Herdr calls:\n%s\nwant:\n%s", data, want)
	}
}

func TestNewPlainTabDoesNotStartCodex(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	cwd := t.TempDir()
	realCWD, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_TEST_REPO", cwd)
	t.Setenv("HERDR_NEW_TAB_CHOICE", "__workspace__:w1")
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	if err := newPlainTab(context.Background(), herdr.New()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	want := "pane list\nworkspace list\ntab create --workspace w1 --cwd " + realCWD + " --focus\n"
	if string(data) != want {
		t.Fatalf("unexpected Herdr calls:\n%s\nwant:\n%s", data, want)
	}
}

func TestCreateTabInWorkspaceOmitsEmptyCWD(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	if _, err := createTabInWorkspace(context.Background(), herdr.New(), "w1", ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "tab create --workspace w1 --focus\n"; got != want {
		t.Fatalf("calls = %q, want %q", got, want)
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

func TestPaletteOpensNewCodexPickerDetached(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "herdr")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$HERDR_TEST_SCRIPT_CALLS\"\nenv | sort >> \"$HERDR_TEST_SCRIPT_CALLS\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", bin)
	t.Setenv("CPP_DRY_RUN", "")
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"pane":"w1:p-popup"}`)
	t.Setenv("HERDR_PLUGIN_ID", "sunznx.command-palette-popup")
	t.Setenv("HERDR_PANE_ID", "w1:p-popup")
	t.Setenv("HERDR_TAB_ID", "w1:t-popup")
	t.Setenv("HERDR_WORKSPACE_ID", "w1")
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_TEST_SCRIPT_CALLS", calls)
	asyncCall := filepath.Join(t.TempDir(), "async-call")
	t.Setenv("HERDR_GO_TEST_ASYNC_CALL", asyncCall)
	t.Setenv("HERDR_GO_TEST_RECORD_CONTEXT", "1")
	if err := dispatchPalette(context.Background(), herdr.New(), herdr.TargetContext{}, "", "plugin", "sunznx.herdr-new-codex.codex", paletteState{}); err != nil {
		t.Fatal(err)
	}
	var data []byte
	for range 100 {
		data, _ = os.ReadFile(asyncCall)
		if len(data) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got := string(data)
	open := "new-codex open-picker picker\n"
	if !strings.Contains(got, open) {
		t.Fatalf("new Codex picker was not opened: %s", got)
	}
	actionOutput := got[strings.LastIndex(got, open)+len(open):]
	if strings.Contains(actionOutput, "=present") {
		t.Fatalf("detached action inherited Herdr runtime context: %s", actionOutput)
	}
}

func TestPaletteForkUsesOriginalPaneContext(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_TEST_FORK", "1")
	state := paletteState{OriginPane: "w1:p-origin"}
	if err := dispatchPalette(context.Background(), herdr.New(), herdr.TargetContext{Pane: "w1:p-popup"}, "w1:p-popup", "plugin", "sunznx.herdr-copy.fork-current-agent-session-in-new-tab", state); err != nil {
		t.Fatal(err)
	}
}

func TestAllManifestsBuildAndRunGoRuntime(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	plugins := map[string][]string{
		"command-palette-popup": {`["./herdr-plugin", "palette", "open"]`, `["./herdr-plugin", "palette", "run"]`},
		"gitui-popup":           {`["./herdr-plugin", "gitui", "open"]`},
		"herdr-ai-rename":       {`["./herdr-plugin", "rename", "ai-current"]`, `["./herdr-plugin", "rename", "picker"]`},
		"herdr-move":            {`["./herdr-plugin", "move", "open"]`, `["./herdr-plugin", "move", "workspace"]`, `["./herdr-plugin", "move", "tab"]`},
		"herdr-new-codex":       {`["./herdr-plugin", "new-codex", "open"]`, `["./herdr-plugin", "new-codex", "open-tab"]`, `["./herdr-plugin", "new-codex", "claude"]`, `["./herdr-plugin", "new-codex", "picker"]`, `["./herdr-plugin", "new-codex", "tab-picker"]`, `["./herdr-plugin", "new-codex", "claude-picker"]`, `["./herdr-plugin", "new-codex", "close"]`},
		"herdr-duplicate":       {`["./herdr-plugin", "duplicate", "tab-or-agent"]`},
		"herdr-copy":            {`["./herdr-plugin", "copy", "current-dir"]`, `["./herdr-plugin", "copy", "current-agent-session"]`, `["./herdr-plugin", "copy", "fork-current-agent-session-in-new-tab"]`, `["./herdr-plugin", "copy", "resume-current-agent-session-in-new-tab"]`, `["./herdr-plugin", "copy", "output"]`, `["./herdr-plugin", "copy", "command-and-output"]`, `["./herdr-plugin", "copy", "zoxide-directory"]`, `["./herdr-plugin", "zoxide", "picker"]`},
		"mole-current-dir":      {`["./herdr-plugin", "mole", "open"]`},
		"open-in":               {`["./herdr-plugin", "open-in", "open", "Emacs"]`},
		"yazi-popup":            {`["./herdr-plugin", "yazi", "picker"]`, `["./herdr-plugin", "yazi", "browser", "rg"]`},
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
