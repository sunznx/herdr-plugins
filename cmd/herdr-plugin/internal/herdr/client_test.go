package herdr

import "testing"

func TestDryRunReadClassification(t *testing.T) {
	for _, args := range [][]string{
		{"pane", "get", "w1:p1"},
		{"plugin", "action", "list"},
		{"plugin", "log", "list"},
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
