package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateAgentSidebar(t *testing.T) {
	base := "onboarding = false\n"
	installed, err := updateAgentSidebar(base, true)
	if err != nil || !strings.Contains(installed, "[ui.sidebar.agents]") {
		t.Fatalf("install: %v", err)
	}
	if again, _ := updateAgentSidebar(installed, true); again != installed {
		t.Fatal("install should be idempotent")
	}
	removed, err := updateAgentSidebar(installed, false)
	if err != nil || removed != base {
		t.Fatalf("remove: %v", err)
	}
	if _, err := updateAgentSidebar("[ui.sidebar.agents]\nrows = []\n", true); err == nil {
		t.Fatal("expected existing sidebar config to be protected")
	}
}

func TestAgentSidebarViewParams(t *testing.T) {
	params := map[string]any{
		"source": agentSidebarSource,
		"sort": []map[string]any{
			{"field": map[string]string{"token": "ws_key"}, "order": "desc"},
			{"field": map[string]string{"token": "tab_key"}, "order": "desc"},
			{"field": map[string]string{"token": "sort_key"}, "order": "desc"},
		},
	}
	if params["source"] != agentSidebarSource {
		t.Fatal("view source must be owned by the plugin")
	}
	if len(params["sort"].([]map[string]any)) != 3 {
		t.Fatal("workspace view must have workspace, tab, and activity sort keys")
	}
}

func TestAgentSidebarUsesAutoTitlePaneLabel(t *testing.T) {
	if !strings.Contains(agentSidebarBlock, `["pane"]`) || strings.Contains(agentSidebarBlock, "terminal_title_stripped") {
		t.Fatal("sidebar must display Auto Title's pane label")
	}
}

func TestAgentSidebarNewClaimDisplacesOldDaemon(t *testing.T) {
	state := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", state)
	claim, old, err := claimAgentSidebarDaemon()
	if err != nil {
		t.Fatal(err)
	}
	if claim != filepath.Join(state, "daemon.pid") || !agentSidebarClaimedBy(claim, old) {
		t.Fatal("old daemon should initially own the claim")
	}
	if err := os.WriteFile(claim, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if agentSidebarClaimedBy(claim, old) || !agentSidebarClaimedBy(claim, "new") {
		t.Fatal("new daemon must displace the old daemon")
	}
}

func TestAgentSidebarLayoutsGroupByWorkspace(t *testing.T) {
	agents := []sidebarAgent{
		{PaneID: "w1:p1", TabID: "w1:t1", WorkspaceID: "w1"},
		{PaneID: "w1:p2", TabID: "w1:t1", WorkspaceID: "w1"},
		{PaneID: "w2:p1", TabID: "w2:t1", WorkspaceID: "w2"},
	}
	activity := map[string]int64{"w1:p1": 10 * 60000, "w1:p2": 12 * 60000, "w2:p1": 20 * 60000}
	layouts := agentSidebarLayouts(agents, activity, map[string]string{"w1": "one", "w2": "two"})
	if layouts["w2:p1"].Group != "two" || layouts["w1:p2"].Group != "one" || layouts["w1:p1"].Group != "" {
		t.Fatal("group header must be written only to the first pane of each sorted workspace")
	}
	if layouts["w1:p1"].Gap != "\u200b" || layouts["w1:p2"].Gap != "" {
		t.Fatal("group gap must be written only to the last pane")
	}
	if layouts["w1:p2"].Indent != "" || layouts["w1:p1"].Indent != "\u200b  " {
		t.Fatal("workspace members must carry Radar's zero-width indentation prefix")
	}
	if layouts["w2:p1"].WSKey <= layouts["w1:p1"].WSKey {
		t.Fatal("workspace activity key must sort newer workspaces first")
	}
	if layouts["w2:p1"].AgentIndex != "1" || layouts["w1:p2"].AgentIndex != "2" || layouts["w1:p1"].AgentIndex != "3" {
		t.Fatal("agent index must follow recent panel order")
	}
}
