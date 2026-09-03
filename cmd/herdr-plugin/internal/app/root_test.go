package app

import "testing"

func TestCommandTree(t *testing.T) {
	routes := [][]string{
		{"gitui", "open"}, {"mole", "open"}, {"open-in", "open", "Emacs"},
		{"copy", "output"}, {"copy", "command-and-output"},
		{"move", "open"}, {"move", "open-tab"}, {"move", "workspace"}, {"move", "tab"},
		{"rename", "ai-current"}, {"rename", "ai-all"}, {"rename", "open", "pane-agent"}, {"rename", "picker"},
		{"new-codex", "open"}, {"new-codex", "picker"}, {"new-codex", "close"},
		{"yazi", "open", "pick"}, {"yazi", "picker"}, {"yazi", "browser", "fzf"},
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
	root := NewCommand()
	cmd, args, err := root.Find([]string{"yazi", "open", "invalid"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Args(cmd, args); err == nil {
		t.Fatal("expected invalid mode to be rejected")
	}
}
