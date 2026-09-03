package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	sharedfzf "github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/fzf"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/workspacepicker"
)

type paletteContext struct {
	Pane      string `json:"pane"`
	Tab       string `json:"tab"`
	Workspace string `json:"workspace"`
	CWD       string `json:"cwd"`
}

type paletteItem struct {
	Kind, Payload, Title, Keywords, Hint, Key, Usage string
	Rank, Count                                      int
	Last                                             int64
}

type usageValue struct {
	Count int   `json:"count"`
	Last  int64 `json:"last"`
}

var staticPaletteItems = []paletteItem{
	{Payload: "new_tab", Title: "New tab", Key: "new_tab", Keywords: "create window", Hint: "herdr tab create --focus"},
	{Payload: "sunznx.herdr-ai-rename.tab", Title: "Rename tab", Keywords: "label title custom plugin", Hint: "sunznx.herdr-ai-rename.tab"},
	{Payload: "close_tab", Title: "Close tab", Key: "close_tab", Keywords: "kill remove quit delete", Hint: "herdr tab close <tab>"},
	{Payload: "split_vertical", Title: "Split pane right (vertical)", Key: "split_vertical", Keywords: "vsplit beside column new", Hint: "herdr pane split <pane> --direction right --focus"},
	{Payload: "split_horizontal", Title: "Split pane down (horizontal)", Key: "split_horizontal", Keywords: "hsplit below row new", Hint: "herdr pane split <pane> --direction down --focus"},
	{Payload: "zoom_pane", Title: "Toggle zoom (fullscreen pane)", Key: "zoom", Keywords: "maximize fullscreen big toggle", Hint: "herdr pane zoom <pane> --toggle"},
	{Payload: "close_pane", Title: "Close pane", Key: "close_pane", Keywords: "kill remove quit delete", Hint: "herdr pane close <pane>"},
	{Payload: "rename_pane", Title: "Rename pane", Key: "rename_pane", Keywords: "label title", Hint: "herdr pane rename <pane> <name>"},
	{Payload: "focus_left", Title: "Focus pane left", Key: "focus_pane_left", Keywords: "go move navigate h"},
	{Payload: "focus_right", Title: "Focus pane right", Key: "focus_pane_right", Keywords: "go move navigate l"},
	{Payload: "focus_up", Title: "Focus pane up", Key: "focus_pane_up", Keywords: "go move navigate k"},
	{Payload: "focus_down", Title: "Focus pane down", Key: "focus_pane_down", Keywords: "go move navigate j"},
	{Payload: "resize_left", Title: "Resize pane left", Key: "resize_mode", Keywords: "grow shrink wider narrower border"},
	{Payload: "resize_right", Title: "Resize pane right", Key: "resize_mode", Keywords: "grow shrink wider narrower border"},
	{Payload: "resize_up", Title: "Resize pane up", Key: "resize_mode", Keywords: "grow shrink taller shorter border"},
	{Payload: "resize_down", Title: "Resize pane down", Key: "resize_mode", Keywords: "grow shrink taller shorter border"},
	{Payload: "swap_left", Title: "Swap pane with the one left", Keywords: "exchange switch rotate reorder"},
	{Payload: "swap_right", Title: "Swap pane with the one right", Keywords: "exchange switch rotate reorder"},
	{Payload: "swap_up", Title: "Swap pane with the one above", Keywords: "exchange switch rotate reorder"},
	{Payload: "swap_down", Title: "Swap pane with the one below", Keywords: "exchange switch rotate reorder"},
	{Payload: "move_pane_tab", Title: "Move pane to tab", Keywords: "send relocate join merge existing"},
	{Payload: "move_pane_workspace", Title: "Move pane to workspace", Keywords: "send relocate project existing new scratch zoxide directory"},
	{Payload: "move_pane_new_tab", Title: "Move pane out to a new tab", Keywords: "send relocate extract break out"},
	{Payload: "start_agent", Title: "Start an agent in a new split", Keywords: "claude codex gemini ai launch spawn run new"},
	{Payload: "prompt_agent", Title: "Send a prompt to an agent", Keywords: "ask message text tell claude ai"},
	{Payload: "interrupt_agent", Title: "Interrupt an agent (esc)", Keywords: "stop cancel escape abort key"},
	{Payload: "sunznx.herdr-ai-rename.agent", Title: "Rename agent", Keywords: "label name current custom plugin"},
	{Payload: "rename_pane_agent", Title: "Rename pane and agent", Keywords: "label name title current together"},
	{Payload: "new_workspace", Title: "New workspace", Keywords: "zoxide git root recent project directory"},
	{Payload: "rename_workspace", Title: "Rename workspace", Key: "rename_workspace", Keywords: "label title project"},
	{Payload: "close_workspace", Title: "Close workspace", Key: "close_workspace", Keywords: "kill remove quit delete project"},
	{Payload: "reload_config", Title: "Reload herdr config", Key: "reload_config", Keywords: "settings keys keybindings toml refresh"},
}

func paletteOpen(ctx context.Context, c herdr.Client) error {
	pc := herdr.PluginContext()
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	cwd := pc.FocusedCWD
	if cwd == "" {
		cwd = pc.WorkspaceCWD
	}
	if cwd == "" {
		cwd = os.Getenv("HERDR_WORKSPACE_CWD")
	}
	if paneCWD := herdr.PaneCWD(pane); cwd == "" {
		cwd = paneCWD
	}
	forwarded, _ := json.Marshal(paletteContext{Pane: pane.PaneID, Tab: pane.TabID, Workspace: pane.WorkspaceID, CWD: cwd})
	return c.OpenPane(ctx, envOr("HERDR_PLUGIN_ID", "sunznx.command-palette-popup"), "palette", true, "", "CPP_CONTEXT_JSON="+string(forwarded))
}

func palette(ctx context.Context, c herdr.Client) error {
	pc := paletteContext{}
	_ = json.Unmarshal([]byte(os.Getenv("CPP_CONTEXT_JSON")), &pc)
	popup := herdr.PluginContext()
	popupPane := popup.FocusedPaneID
	if popupPane != "" {
		pc.Pane = popupPane
		if popup.TabID != "" {
			pc.Tab = popup.TabID
		}
		if popup.WorkspaceID != "" {
			pc.Workspace = popup.WorkspaceID
		}
		if popup.FocusedCWD != "" {
			pc.CWD = popup.FocusedCWD
		} else if popup.WorkspaceCWD != "" {
			pc.CWD = popup.WorkspaceCWD
		}
	}
	items, state, err := buildPalette(ctx, c, pc)
	if err != nil {
		return err
	}
	rows := renderPalette(items)
	if os.Getenv("CPP_LIST_ONLY") == "1" {
		fmt.Print(rows)
		return nil
	}
	choice := ""
	if wanted := os.Getenv("CPP_CHOICE"); wanted != "" {
		for _, item := range items {
			if item.Kind+"\t"+item.Payload == wanted {
				choice = item.Kind + "\t" + item.Payload
				break
			}
		}
		if choice == "" {
			return fmt.Errorf("CPP_CHOICE %q matched no row", wanted)
		}
	} else {
		selected, err := sharedfzf.Pick(ctx, rows, "--delimiter=\t", "--with-nth=3", "--ansi", "--prompt=herdr command ▸ ", "--header=↑↓ select · enter run · esc cancel", "--reverse", "--cycle", "--no-multi", "--tiebreak=begin,index", "--info=inline", `--preview=printf "%s\n" {5}`, "--preview-window=down,3,wrap,border-top")
		if err != nil || selected == "" {
			return err
		}
		parts := strings.SplitN(selected, "\t", 3)
		if len(parts) < 2 {
			return fmt.Errorf("invalid palette selection")
		}
		choice = parts[0] + "\t" + parts[1]
	}
	parts := strings.SplitN(choice, "\t", 2)
	return dispatchPalette(ctx, c, pc, popupPane, parts[0], parts[1], state)
}

type paletteState struct {
	Tabs       []tabRow
	Workspaces []workspaceRow
	Agents     []agentRow
}

func buildPalette(ctx context.Context, c herdr.Client, pc paletteContext) ([]paletteItem, paletteState, error) {
	usageDirOut, _ := c.Run(ctx, "plugin", "config-dir", "sunznx.command-palette-popup")
	usageDir := strings.TrimSpace(string(usageDirOut))
	usage := readUsage(filepath.Join(usageDir, "usage.json"))
	enableWorktree := configBool(envOr("CPP_CONFIG_PATH", filepath.Join(usageDir, "config.toml")), "enable_worktree")
	keys := effectiveKeys(ctx, c)
	items := append([]paletteItem(nil), staticPaletteItems...)
	if enableWorktree {
		items = append(items,
			paletteItem{Payload: "new_worktree", Title: "New worktree here", Key: "new_worktree", Keywords: "git branch checkout create"},
			paletteItem{Payload: "new_worktree_branch", Title: "New worktree on a branch", Keywords: "git checkout create base prompt"},
			paletteItem{Payload: "remove_worktree", Title: "Remove this worktree checkout", Key: "remove_worktree", Keywords: "git delete prune rm"})
	}
	for i := range items {
		items[i].Kind, items[i].Rank = "static", i
		items[i].Usage = items[i].Payload
		if strings.HasPrefix(items[i].Payload, "sunznx.herdr-ai-rename.") {
			items[i].Usage = "plugin:" + items[i].Payload
		}
		if u := usage[items[i].Usage]; u.Count > 0 {
			items[i].Count, items[i].Last = u.Count, u.Last
		}
		items[i].Key = displayKey(keys, items[i].Key, 0)
		items[i].Keywords += " " + items[i].Payload
	}
	// Decode plugin action fields explicitly because their JSON tags differ.
	var rawActions struct {
		Result struct {
			Actions []map[string]any `json:"actions"`
		} `json:"result"`
	}
	if err := c.JSON(ctx, &rawActions, "plugin", "action", "list"); err == nil {
		excluded := map[string]bool{"sunznx.herdr-move.open": true, "sunznx.herdr-move.tab": true, "sunznx.herdr-ai-rename.open": true, "sunznx.herdr-ai-rename.tab": true, "sunznx.herdr-ai-rename.agent": true}
		for i, raw := range rawActions.Result.Actions {
			plugin, _ := raw["plugin_id"].(string)
			action, _ := raw["action_id"].(string)
			title, _ := raw["title"].(string)
			qid := plugin + "." + action
			if plugin == envOr("HERDR_PLUGIN_ID", "sunznx.command-palette-popup") || excluded[qid] {
				continue
			}
			u := usage["plugin:"+qid]
			items = append(items, paletteItem{Kind: "plugin", Payload: qid, Title: title, Keywords: "plugin action " + qid, Hint: "herdr plugin action invoke " + qid, Usage: "plugin:" + qid, Rank: 1000 + i, Count: u.Count, Last: u.Last})
		}
	}
	staticEnd := len(items)
	sort.SliceStable(items[:staticEnd], func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		if items[i].Last != items[j].Last {
			return items[i].Last > items[j].Last
		}
		return items[i].Rank < items[j].Rank
	})
	state := paletteState{}
	var tabsResp struct {
		Result struct {
			Tabs []tabRow `json:"tabs"`
		} `json:"result"`
	}
	_ = c.JSON(ctx, &tabsResp, "tab", "list")
	state.Tabs = tabsResp.Result.Tabs
	var localTabs struct {
		Result struct {
			Tabs []tabRow `json:"tabs"`
		} `json:"result"`
	}
	if pc.Workspace != "" {
		_ = c.JSON(ctx, &localTabs, "tab", "list", "--workspace", pc.Workspace)
	} else {
		localTabs = tabsResp
	}
	for _, tab := range localTabs.Result.Tabs {
		if tab.TabID != pc.Tab {
			items = append(items, paletteItem{Kind: "goto_tab", Payload: tab.TabID, Title: "Go to tab: " + fallback(tab.Label, tab.TabID), Key: displayKey(keys, "switch_tab", tab.Number), Keywords: "switch jump goto tab " + tab.Label, Hint: "herdr tab focus " + tab.TabID})
		}
	}
	var wsResp struct {
		Result struct {
			Workspaces []workspaceRow `json:"workspaces"`
		} `json:"result"`
	}
	_ = c.JSON(ctx, &wsResp, "workspace", "list")
	state.Workspaces = wsResp.Result.Workspaces
	state.Agents, _ = listAgents(ctx, c)
	for i, agent := range state.Agents {
		if agent.PaneID != pc.Pane {
			title := fallback(agent.TerminalTitleStripped, fallback(agent.Agent, agent.PaneID))
			items = append(items, paletteItem{Kind: "focus_agent", Payload: agent.PaneID, Title: "Focus agent: " + truncate(title, 38) + " · " + fallback(agent.AgentStatus, "?"), Key: displayKey(keys, "focus_agent", i+1), Keywords: "agent claude codex focus jump " + agent.Agent + " " + agent.AgentStatus + " " + title, Hint: "herdr agent focus " + agent.PaneID})
		}
	}
	if enableWorktree && pc.CWD != "" {
		var wt struct {
			Result struct {
				Worktrees []struct {
					Path            string `json:"path"`
					Branch          string `json:"branch"`
					OpenWorkspaceID string `json:"open_workspace_id"`
					IsBare          bool   `json:"is_bare"`
					IsPrunable      bool   `json:"is_prunable"`
				} `json:"worktrees"`
			} `json:"result"`
		}
		if err := c.JSON(ctx, &wt, "worktree", "list", "--cwd", pc.CWD); err == nil {
			for _, w := range wt.Result.Worktrees {
				if w.OpenWorkspaceID == "" && !w.IsBare && !w.IsPrunable {
					title := w.Branch
					if title == "" {
						title = filepath.Base(w.Path)
					}
					items = append(items, paletteItem{Kind: "open_worktree", Payload: w.Path, Title: "Open worktree: " + truncate(title, 34), Keywords: "git worktree branch checkout open " + w.Branch + " " + w.Path, Hint: "herdr worktree open --path " + w.Path + " --focus"})
				}
			}
		}
	}
	return items, state, nil
}

func renderPalette(items []paletteItem) string {
	width := 24
	for _, item := range items {
		if item.Kind == "static" && len([]rune(item.Title)) > width {
			width = len([]rune(item.Title))
		}
	}
	var out strings.Builder
	for _, item := range items {
		title := pad(item.Title, width)
		key := pad(item.Key, 14)
		display := strings.TrimSpace(title + "  " + key + "  \x1b[2m" + item.Keywords + "\x1b[0m")
		fmt.Fprintf(&out, "%s\t%s\t%s\t%s\t%s\n", item.Kind, item.Payload, display, item.Keywords, item.Hint)
	}
	return out.String()
}

func dispatchPalette(ctx context.Context, c herdr.Client, pc paletteContext, popupPane, kind, payload string, state paletteState) error {
	if kind == "goto_tab" {
		_, err := c.Run(ctx, "tab", "focus", payload)
		return err
	}
	if kind == "focus_agent" {
		_, err := c.Run(ctx, "agent", "focus", payload)
		return err
	}
	if kind == "open_worktree" {
		args := []string{"worktree", "open", "--path", payload, "--focus"}
		if pc.CWD != "" {
			args = append(args, "--cwd", pc.CWD)
		}
		_, err := c.Run(ctx, args...)
		return err
	}
	if kind == "plugin" {
		recordUsage(ctx, c, "plugin:"+payload)
		return invokePlugin(ctx, c, payload)
	}
	if kind != "static" {
		return fmt.Errorf("unrecognized selection kind %q", kind)
	}
	usage := payload
	if strings.HasPrefix(payload, "sunznx.herdr-ai-rename.") {
		usage = "plugin:" + payload
	}
	recordUsage(ctx, c, usage)
	require := func(value, message string) (string, error) {
		if value == "" {
			return "", fmt.Errorf("%s", message)
		}
		return value, nil
	}
	var args []string
	switch payload {
	case "new_tab":
		args = []string{"tab", "create", "--focus"}
		if pc.Workspace != "" {
			args = append(args, "--workspace", pc.Workspace)
		}
		if pc.CWD != "" {
			args = append(args, "--cwd", pc.CWD)
		}
	case "close_tab":
		if _, err := require(pc.Tab, "no origin tab to close"); err != nil {
			return err
		}
		args = []string{"tab", "close", pc.Tab}
	case "split_vertical", "split_horizontal":
		if _, err := require(pc.Pane, "no origin pane to split"); err != nil {
			return err
		}
		dir := "right"
		if payload == "split_horizontal" {
			dir = "down"
		}
		args = []string{"pane", "split", pc.Pane, "--direction", dir, "--focus"}
		if pc.CWD != "" {
			args = append(args, "--cwd", pc.CWD)
		}
	case "close_pane":
		if _, err := require(pc.Pane, "no origin pane to close"); err != nil {
			return err
		}
		args = []string{"pane", "close", pc.Pane}
	case "zoom_pane":
		if _, err := require(pc.Pane, "no origin pane to zoom"); err != nil {
			return err
		}
		args = []string{"pane", "zoom", pc.Pane, "--toggle"}
	case "rename_pane":
		if _, err := require(pc.Pane, "no origin pane to rename"); err != nil {
			return err
		}
		name, _ := ask("New pane name: ")
		if name == "" {
			return nil
		}
		args = []string{"pane", "rename", pc.Pane, name}
	case "focus_left", "focus_right", "focus_up", "focus_down":
		if _, err := require(pc.Pane, "no origin pane to focus from"); err != nil {
			return err
		}
		args = []string{"pane", "focus", "--direction", strings.TrimPrefix(payload, "focus_"), "--pane", pc.Pane}
	case "resize_left", "resize_right", "resize_up", "resize_down":
		if _, err := require(pc.Pane, "no origin pane to resize"); err != nil {
			return err
		}
		args = []string{"pane", "resize", "--direction", strings.TrimPrefix(payload, "resize_"), "--pane", pc.Pane}
	case "swap_left", "swap_right", "swap_up", "swap_down":
		if _, err := require(pc.Pane, "no origin pane to swap"); err != nil {
			return err
		}
		args = []string{"pane", "swap", "--direction", strings.TrimPrefix(payload, "swap_"), "--pane", pc.Pane}
	case "move_pane_new_tab":
		if _, err := require(pc.Pane, "no origin pane to move"); err != nil {
			return err
		}
		args = []string{"pane", "move", pc.Pane, "--new-tab", "--focus"}
	case "move_pane_tab":
		return paletteMoveTab(ctx, c, popupPane, state)
	case "move_pane_workspace":
		return paletteMoveWorkspace(ctx, c, popupPane)
	case "sunznx.herdr-ai-rename.tab":
		return paletteRenameTab(ctx, c, popupPane)
	case "sunznx.herdr-ai-rename.agent":
		return paletteRenameAgent(ctx, c, popupPane, false, state.Agents)
	case "rename_pane_agent":
		return paletteRenameAgent(ctx, c, popupPane, true, state.Agents)
	case "new_workspace":
		return paletteNewWorkspace(ctx, c)
	case "rename_workspace":
		if _, err := require(pc.Workspace, "no origin workspace to rename"); err != nil {
			return err
		}
		name, _ := ask("New workspace name: ")
		if name == "" {
			return nil
		}
		args = []string{"workspace", "rename", pc.Workspace, name}
	case "close_workspace":
		if _, err := require(pc.Workspace, "no origin workspace to close"); err != nil {
			return err
		}
		args = []string{"workspace", "close", pc.Workspace}
	case "start_agent":
		if _, err := require(pc.Pane, "no origin pane to split for the agent"); err != nil {
			return err
		}
		return paletteStartAgent(ctx, c, pc)
	case "prompt_agent", "interrupt_agent":
		target, err := pickAgent(ctx, state.Agents)
		if err != nil || target == "" {
			return err
		}
		if payload == "prompt_agent" {
			text, _ := ask("Prompt: ")
			if text == "" {
				return nil
			}
			args = []string{"agent", "prompt", target, text}
		} else {
			args = []string{"agent", "send-keys", target, "esc"}
		}
	case "new_worktree":
		if _, err := require(pc.Workspace, "no origin workspace to create a worktree in"); err != nil {
			return err
		}
		args = []string{"worktree", "create", "--workspace", pc.Workspace, "--focus"}
	case "new_worktree_branch":
		if _, err := require(pc.Workspace, "no origin workspace to create a worktree in"); err != nil {
			return err
		}
		branch, _ := ask("Branch name: ")
		if branch == "" {
			return nil
		}
		args = []string{"worktree", "create", "--workspace", pc.Workspace, "--branch", branch, "--focus"}
		base, _ := ask("Base ref (empty = default): ")
		if base != "" {
			args = append(args, "--base", base)
		}
	case "remove_worktree":
		if _, err := require(pc.Workspace, "no origin workspace to remove"); err != nil {
			return err
		}
		confirm, _ := ask("Remove the worktree checkout for " + pc.Workspace + "? [y/N] ")
		if confirm != "y" && confirm != "Y" && confirm != "yes" && confirm != "YES" {
			return nil
		}
		args = []string{"worktree", "remove", "--workspace", pc.Workspace}
	case "reload_config":
		args = []string{"server", "reload-config"}
	default:
		return fmt.Errorf("unknown action %q", payload)
	}
	_, err := c.Run(ctx, args...)
	return err
}

func paletteMoveWorkspace(ctx context.Context, c herdr.Client, popupPane string) error {
	pane, err := livePopupPane(ctx, c, popupPane)
	if err != nil {
		return err
	}
	if pane.PaneID == os.Getenv("HERDR_PANE_ID") {
		return fmt.Errorf("refused to move the palette pane")
	}
	choice, err := pickWorkspace(ctx, c, "workspace ▸ ", pane.WorkspaceID, "CPP_PICK_VALUE", "CPP_WORKSPACE_CANDIDATES_FILE")
	if err != nil || choice == nil {
		return err
	}
	target, bootstrap, _, err := createWorkspace(ctx, c, *choice, false)
	if err != nil {
		return err
	}
	if _, err = c.Run(ctx, "pane", "move", pane.PaneID, "--new-tab", "--workspace", target, "--focus"); err != nil {
		return err
	}
	if bootstrap != "" {
		_, err = c.Run(ctx, "pane", "close", bootstrap)
	}
	return err
}
func paletteNewWorkspace(ctx context.Context, c herdr.Client) error {
	choice, err := pickWorkspace(ctx, c, "workspace ▸ ", "", "CPP_PICK_VALUE", "CPP_WORKSPACE_CANDIDATES_FILE")
	if err != nil || choice == nil {
		return err
	}
	if choice.Kind == workspacepicker.Workspace || (choice.Kind == workspacepicker.Scratch && choice.WorkspaceID != "") {
		_, err = c.Run(ctx, "workspace", "focus", choice.WorkspaceID)
		return err
	}
	_, _, _, err = createWorkspace(ctx, c, *choice, true)
	return err
}
func paletteMoveTab(ctx context.Context, c herdr.Client, popupPane string, state paletteState) error {
	pane, err := livePopupPane(ctx, c, popupPane)
	if err != nil {
		return err
	}
	labels := map[string]string{}
	for _, w := range state.Workspaces {
		labels[w.WorkspaceID] = w.Label
	}
	var rows strings.Builder
	valid := map[string]bool{}
	for _, t := range state.Tabs {
		if t.TabID == pane.TabID {
			continue
		}
		valid[t.TabID] = true
		fmt.Fprintf(&rows, "%s\t%s / #%d %s\n", t.TabID, fallback(labels[t.WorkspaceID], t.WorkspaceID), t.Number, t.Label)
	}
	target, err := nestedPick(ctx, rows.String(), "move to tab ▸ ")
	if err != nil || target == "" {
		return err
	}
	if !valid[target] {
		return fmt.Errorf("destination tab %q is unavailable", target)
	}
	var response struct {
		Result struct {
			Tab tabRow `json:"tab"`
		} `json:"result"`
	}
	if err = c.JSON(ctx, &response, "tab", "get", target); err != nil || response.Result.Tab.TabID != target {
		return fmt.Errorf("destination tab %q is no longer available", target)
	}
	_, err = c.Run(ctx, "pane", "move", pane.PaneID, "--tab", target, "--split", "right", "--focus")
	return err
}
func paletteRenameTab(ctx context.Context, c herdr.Client, popupPane string) error {
	pane, err := livePopupPane(ctx, c, popupPane)
	if err != nil {
		return err
	}
	name, _ := ask("New tab name: ")
	if name == "" {
		return nil
	}
	var response struct {
		Result struct {
			Tab tabRow `json:"tab"`
		} `json:"result"`
	}
	if err = c.JSON(ctx, &response, "tab", "get", pane.TabID); err != nil || response.Result.Tab.TabID != pane.TabID {
		return fmt.Errorf("origin tab is no longer available")
	}
	_, err = c.Run(ctx, "tab", "rename", pane.TabID, name)
	return err
}
func paletteRenameAgent(ctx context.Context, c herdr.Client, popupPane string, renamePane bool, agents []agentRow) error {
	pane, err := livePopupPane(ctx, c, popupPane)
	if err != nil {
		return err
	}
	prompt := "New agent name (a-z0-9_-): "
	if renamePane {
		prompt = "New pane and agent name: "
	}
	name, _ := ask(prompt)
	if name == "" {
		return nil
	}
	if liveAgents, listErr := listAgents(ctx, c); listErr == nil {
		agents = liveAgents
	} else {
		return listErr
	}
	present := hasAgent(agents, pane.PaneID)
	if !renamePane && !present {
		return fmt.Errorf("the origin pane does not have a detected agent")
	}
	if present && !agentName.MatchString(name) {
		return fmt.Errorf("agent names must match [a-z][a-z0-9_-]{0,31}")
	}
	if present {
		if _, err = c.Run(ctx, "agent", "rename", pane.PaneID, name); err != nil {
			return err
		}
	}
	if renamePane {
		_, err = c.Run(ctx, "pane", "rename", pane.PaneID, name)
	}
	return err
}
func livePopupPane(ctx context.Context, c herdr.Client, id string) (herdr.Pane, error) {
	if id == "" {
		return herdr.Pane{}, fmt.Errorf("popup origin pane is unavailable")
	}
	pane, err := c.GetPane(ctx, id)
	if err != nil {
		return herdr.Pane{}, fmt.Errorf("origin pane %q is no longer available", id)
	}
	return pane, nil
}

func nestedPick(ctx context.Context, rows, prompt string) (string, error) {
	if value := os.Getenv("CPP_PICK_VALUE"); value != "" {
		return value, nil
	}
	selected, err := sharedfzf.Pick(ctx, rows, "--delimiter=\t", "--with-nth=2", "--prompt="+prompt, "--reverse", "--cycle", "--no-multi", "--tiebreak=begin,index")
	if err != nil || selected == "" {
		return "", err
	}
	return strings.SplitN(selected, "\t", 2)[0], nil
}
func pickAgent(ctx context.Context, agents []agentRow) (string, error) {
	var rows strings.Builder
	for _, a := range agents {
		fmt.Fprintf(&rows, "%s\t%s · %s · %s\n", a.PaneID, fallback(a.TerminalTitleStripped, fallback(a.Agent, a.PaneID)), fallback(a.AgentStatus, "?"), a.PaneID)
	}
	if rows.Len() == 0 {
		return "", fmt.Errorf("no live agents")
	}
	return nestedPick(ctx, rows.String(), "agent ▸ ")
}
func paletteStartAgent(ctx context.Context, c herdr.Client, pc paletteContext) error {
	out, _ := c.Run(ctx, "completion", "zsh")
	re := regexp.MustCompile(`--kind\[Supported agent kind[^()]*\(([^)]*)\)`)
	kinds := []string{"claude", "codex", "gemini", "cursor", "opencode", "copilot", "amp", "droid"}
	if match := re.FindStringSubmatch(string(out)); len(match) > 1 {
		kinds = strings.Fields(match[1])
	}
	var rows strings.Builder
	for _, kind := range kinds {
		fmt.Fprintf(&rows, "%s\t%s\n", kind, kind)
	}
	kind, err := nestedPick(ctx, rows.String(), "agent kind ▸ ")
	if err != nil || kind == "" {
		return err
	}
	name, _ := ask("Agent name (a-z0-9_-): ")
	if !agentName.MatchString(name) {
		return fmt.Errorf("agent names must match [a-z][a-z0-9_-]{0,31}")
	}
	args := []string{"pane", "split", pc.Pane, "--direction", "right", "--focus"}
	if pc.CWD != "" {
		args = append(args, "--cwd", pc.CWD)
	}
	out, err = c.Run(ctx, args...)
	if err != nil {
		return err
	}
	var response struct {
		Result struct {
			Pane herdr.Pane `json:"pane"`
		} `json:"result"`
	}
	if c.DryRun {
		response.Result.Pane.PaneID = "<new-pane>"
	} else if err = decode(out, &response); err != nil {
		return err
	}
	_, err = c.Run(ctx, "agent", "start", name, "--kind", kind, "--pane", response.Result.Pane.PaneID)
	return err
}

func invokePlugin(ctx context.Context, c herdr.Client, id string) error {
	out, err := c.Run(ctx, "plugin", "action", "invoke", id)
	if err != nil || c.DryRun {
		return err
	}
	var started struct {
		Result struct {
			Log struct {
				LogID    string `json:"log_id"`
				PluginID string `json:"plugin_id"`
			} `json:"log"`
		} `json:"result"`
	}
	_ = decode(out, &started)
	if started.Result.Log.LogID == "" || started.Result.Log.PluginID == "" {
		return nil
	}
	for range 25 {
		var logs struct {
			Result struct {
				Logs []map[string]any `json:"logs"`
			} `json:"result"`
		}
		if c.JSON(ctx, &logs, "plugin", "log", "list", "--plugin", started.Result.Log.PluginID, "--limit", "20") == nil {
			for _, entry := range logs.Result.Logs {
				if entry["log_id"] == started.Result.Log.LogID {
					status, _ := entry["status"].(string)
					if status == "succeeded" {
						return nil
					}
					if status == "failed" {
						return fmt.Errorf("%s failed: %v", id, entry["stderr"])
					}
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func readUsage(path string) map[string]usageValue {
	result := map[string]usageValue{}
	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return result
	}
	for key, value := range raw {
		var item usageValue
		if json.Unmarshal(value, &item) == nil && item.Count > 0 {
			result[key] = item
			continue
		}
		var count int
		if json.Unmarshal(value, &count) == nil {
			result[key] = usageValue{Count: count}
		}
	}
	return result
}
func recordUsage(ctx context.Context, c herdr.Client, id string) {
	if c.DryRun {
		return
	}
	out, err := c.Run(ctx, "plugin", "config-dir", "sunznx.command-palette-popup")
	if err != nil {
		return
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "usage.json")
	usage := readUsage(path)
	item := usage[id]
	item.Count++
	item.Last = time.Now().Unix()
	usage[id] = item
	data, _ := json.Marshal(usage)
	tmp, err := os.CreateTemp(dir, "usage-*.json")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Chmod(0600)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		_ = os.Rename(tmpPath, path)
	}
}
func configBool(path, key string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.SplitN(scanner.Text(), "#", 2)[0]
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
			return strings.TrimSpace(parts[1]) == "true"
		}
	}
	return false
}
func effectiveKeys(ctx context.Context, c herdr.Client) map[string]string {
	keys := map[string]string{}
	out, _ := c.Run(ctx, "--default-config")
	parseKeyTable(string(out), true, keys)
	path := envOr("HERDR_CONFIG_PATH", filepath.Join(envOr("HOME", ""), ".config", "herdr", "config.toml"))
	data, _ := os.ReadFile(path)
	parseKeyTable(string(data), false, keys)
	shadowed := commandKeys(string(data))
	for name, key := range keys {
		if shadowed[key] {
			delete(keys, name)
		}
	}
	return keys
}

func commandKeys(data string) map[string]bool {
	result := map[string]bool{}
	inCommand := false
	re := regexp.MustCompile(`^\s*key\s*=\s*"([^"]*)"`)
	for _, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inCommand = trimmed == "[[keys.command]]"
			continue
		}
		if inCommand {
			if match := re.FindStringSubmatch(line); len(match) == 2 {
				result[match[1]] = true
			}
		}
	}
	return result
}
func parseKeyTable(data string, uncomment bool, dst map[string]string) {
	inKeys := false
	re := regexp.MustCompile(`^\s*([a-z_][a-z0-9_]*)\s*=\s*"([^"]*)"`)
	for _, line := range strings.Split(data, "\n") {
		if uncomment {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "#")
			line = strings.TrimSpace(line)
		}
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			inKeys = strings.TrimSpace(line) == "[keys]"
			continue
		}
		if !inKeys {
			continue
		}
		if m := re.FindStringSubmatch(line); len(m) == 3 {
			dst[m[1]] = m[2]
		}
	}
}
func displayKey(keys map[string]string, name string, index int) string {
	if name == "" {
		return ""
	}
	key := keys[name]
	if index > 0 {
		if index > 9 {
			return ""
		}
		key = strings.Replace(key, "1..9", strconv.Itoa(index), 1)
	}
	if strings.Contains(key, "1..9") {
		return ""
	}
	if strings.HasPrefix(key, "prefix+") {
		return fallback(keys["prefix"], "ctrl+b") + " " + strings.TrimPrefix(key, "prefix+")
	}
	return key
}
func fallback(value, other string) string {
	if value != "" {
		return value
	}
	return other
}
func truncate(value string, n int) string {
	r := []rune(value)
	if len(r) <= n {
		return value
	}
	return string(r[:n-1]) + "…"
}
func pad(value string, n int) string {
	r := []rune(value)
	if len(r) >= n {
		return value
	}
	return value + strings.Repeat(" ", n-len(r))
}
