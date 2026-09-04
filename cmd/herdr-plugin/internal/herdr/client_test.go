package herdr

import "testing"

func TestDryRunReadClassification(t *testing.T) {
	for _, args := range [][]string{
		{"pane", "get", "w1:p1"},
		{"pane", "wait-output", "w1:p1", "--match", "ready"},
		{"plugin", "action", "list"},
		{"plugin", "log", "list"},
		{"agent", "wait", "w1:p1"},
		{"--default-config"},
	} {
		if !readOnly(args) {
			t.Errorf("expected read-only: %v", args)
		}
	}
	for _, args := range [][]string{
		{"pane", "move", "w1:p1"},
		{"plugin", "action", "invoke", "x.y"},
		{"server", "reload-config"},
	} {
		if readOnly(args) {
			t.Errorf("expected mutation: %v", args)
		}
	}
}

func TestMergePopupContext(t *testing.T) {
	base := TargetContext{Pane: "origin-pane", Tab: "origin-tab", Workspace: "origin-workspace", CWD: "/origin"}
	if got := MergePopupContext(base, Context{TabID: "ignored-tab"}); got != base {
		t.Fatalf("context without a focused pane should be ignored: %#v", got)
	}
	want := TargetContext{Pane: "popup-pane", Tab: "origin-tab", Workspace: "popup-workspace", CWD: "/popup"}
	got := MergePopupContext(base, Context{FocusedPaneID: "popup-pane", WorkspaceID: "popup-workspace", WorkspaceCWD: "/popup"})
	if got != want {
		t.Fatalf("merge popup context: got %#v, want %#v", got, want)
	}
}
