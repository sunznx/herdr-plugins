package app

import (
	"strings"
	"testing"
)

func TestCommandTree(t *testing.T) {
	routes := [][]string{
		{"gitui", "open"}, {"mole", "open"}, {"open-in", "open", "Emacs"},
		{"copy", "output"}, {"copy", "command-and-output"},
		{"copy", "current-dir"}, {"copy", "current-agent-session"},
		{"copy", "fork-current-agent-session-in-new-tab"}, {"copy", "resume-current-agent-session-in-new-tab"},
		{"copy", "zoxide-directory"},
		{"move", "open"}, {"move", "open-tab"}, {"move", "workspace"}, {"move", "tab"},
		{"rename", "ai-current"}, {"rename", "ai-all"}, {"rename", "open", "pane-agent"}, {"rename", "picker"},
		{"new-codex", "open"}, {"new-codex", "open-tab"}, {"new-codex", "claude"}, {"new-codex", "open-picker", "picker"}, {"new-codex", "picker"}, {"new-codex", "tab-picker"}, {"new-codex", "claude-picker"}, {"new-codex", "close"},
		{"duplicate", "tab-or-agent"},
		{"yazi", "open", "pick"}, {"yazi", "open", "fzf"}, {"yazi", "open", "rg"}, {"yazi", "picker"}, {"yazi", "browser", "fzf"},
		{"palette", "open"}, {"palette", "run"},
		{"agent-sidebar", "configure"}, {"agent-sidebar", "unconfigure"}, {"agent-sidebar", "restart"}, {"agent-sidebar", "self-test"}, {"agent-sidebar", "daemon"},
	}
	root := NewCommand()
	for _, route := range routes {
		cmd, args, err := root.Find(route)
		if err != nil {
			t.Fatalf("find %v: %v", route, err)
		}
		if cmd.RunE == nil {
			t.Fatalf("%v resolves to non-runnable command %q", route, cmd.CommandPath())
		}
		if err := cmd.Args(cmd, args); err != nil {
			t.Fatalf("validate %v: %v", route, err)
		}
	}
}

func TestCommandTreeRejectsInvalidMode(t *testing.T) {
	for _, mode := range []string{"invalid", "trellis"} {
		root := NewCommand()
		cmd, args, err := root.Find([]string{"yazi", "open", mode})
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Args(cmd, args); err == nil {
			t.Fatalf("expected mode %q to be rejected", mode)
		}
	}
}

func TestDisplayKeyReadsIndexedArrayBinding(t *testing.T) {
	keys := map[string]string{}
	parseKeyTable("[keys]\nfocus_agent = [\"alt+1..9\"]\nprefix = \"ctrl+x\"\n", false, keys)
	if got := displayKey(keys, "focus_agent", 2); got != "alt+2" {
		t.Fatalf("displayKey() = %q, want alt+2", got)
	}
}

func TestWorkspacePathChanged(t *testing.T) {
	if !workspacePathChanged("/repo", "/repo/sub") {
		t.Fatal("expected different workspace paths to require /cd")
	}
	if workspacePathChanged("/repo", "/repo/./") {
		t.Fatal("equivalent paths should not require /cd")
	}
	if workspacePathChanged("", "/repo") {
		t.Fatal("missing source path should not require /cd")
	}
}

func TestPaneMoveArgsReuseNewWorkspaceRootTab(t *testing.T) {
	if got := paneMoveArgs("w1:p1", "w2", "w2:p1"); strings.Contains(strings.Join(got, " "), "--new-tab") {
		t.Fatalf("new workspace should reuse bootstrap tab: %v", got)
	}
	if got := paneMoveArgs("w1:p1", "w2", ""); !strings.Contains(strings.Join(got, " "), "--new-tab") {
		t.Fatalf("existing workspace should create a new tab: %v", got)
	}
}
