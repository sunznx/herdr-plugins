package app

import (
	"testing"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func TestDuplicateAgentSupportsClaude(t *testing.T) {
	origin := herdr.Pane{TabID: "tab-1", Agent: "claude"}
	if got := duplicateAgent(origin, nil); got != "claude" {
		t.Fatalf("duplicateAgent() = %q, want claude", got)
	}
}

func TestDuplicateAgentFindsClaudeInTab(t *testing.T) {
	origin := herdr.Pane{TabID: "tab-1"}
	agents := []agentRow{{TabID: "tab-1", Agent: "claude"}}
	if got := duplicateAgent(origin, agents); got != "claude" {
		t.Fatalf("duplicateAgent() = %q, want claude", got)
	}
}
