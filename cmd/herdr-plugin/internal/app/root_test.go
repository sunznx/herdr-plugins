package app

import "testing"

func TestCommandTree(t *testing.T) {
	routes := [][]string{
		{"gitui", "open"}, {"mole", "open"}, {"open-in", "open", "Emacs"},
		{"copy", "output"}, {"copy", "command-and-output"},
		{"move", "open"}, {"move", "open-tab"}, {"move", "workspace"}, {"move", "tab"},
		{"rename", "ai-current"}, {"rename", "ai-all"}, {"rename", "open", "pane-agent"}, {"rename", "picker"},
		{"new-codex", "open"}, {"new-codex", "picker"}, {"new-codex", "close"},
		{"yazi", "open", "pick"}, {"yazi", "open", "fzf"}, {"yazi", "open", "rg"}, {"yazi", "picker"}, {"yazi", "browser", "fzf"},
		{"palette", "open"}, {"palette", "run"},
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
