package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

const agentSidebarStart = "# >>> sunznx.herdr-agent-sidebar"
const agentSidebarEnd = "# <<< sunznx.herdr-agent-sidebar"

const agentSidebarBlock = agentSidebarStart + `
[ui.sidebar.agents]
row_gap = 0
rows = [
  ["$group"],
  [
    "$agent_index",
    { token = "$agent_logo_claude", fg = "#d97757", bold = true },
    { token = "$agent_logo_codex", fg = "#5c8dff", bold = true },
    { token = "$agent_logo_gemini", fg = "#4285f4", bold = true },
    { token = "$agent_logo_opencode", fg = "#9a9808", bold = true },
    { token = "$agent_logo_other", fg = "#c78a1f", bold = true },
    "state_text",
  ],
  ["pane"],
  ["$gap"],
]
` + agentSidebarEnd

const agentSidebarSource = "plugin:sunznx.herdr-agent-sidebar"

var agentSidebarLogoTokens = map[string]string{
	"claude":   "agent_logo_claude",
	"codex":    "agent_logo_codex",
	"gemini":   "agent_logo_gemini",
	"opencode": "agent_logo_opencode",
	"other":    "agent_logo_other",
}

type sidebarAgent struct {
	PaneID      string `json:"pane_id"`
	TabID       string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	AgentStatus string `json:"agent_status"`
	Agent       string `json:"agent"`
}

type sidebarWorkspace struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

type sidebarLayout struct {
	SortKey    string
	WSKey      string
	TabKey     string
	Group      string
	Gap        string
	Indent     string
	AgentIndex string
}

func agentSidebarDaemon(ctx context.Context, c herdr.Client) error {
	claim, owner, err := claimAgentSidebarDaemon()
	if err != nil {
		return err
	}
	activity := loadAgentSidebarActivity()
	lastActivityWrite := time.Time{}
	last := map[string]string{}
	viewSet := false
	for {
		if !agentSidebarClaimedBy(claim, owner) {
			return nil
		}
		var response struct {
			Result struct {
				Agents []sidebarAgent `json:"agents"`
			} `json:"result"`
		}
		if err := c.JSON(ctx, &response, "agent", "list"); err == nil {
			if !viewSet {
				if err := setAgentSidebarView(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "agent-sidebar: set workspace view: %v\n", err)
				} else {
					viewSet = true
				}
			}
			now := time.Now()
			changed := false
			for _, agent := range response.Result.Agents {
				if agent.PaneID != "" && agent.AgentStatus == "working" {
					activity[agent.PaneID] = now.UnixMilli()
					changed = true
				}
			}
			if changed && (lastActivityWrite.IsZero() || now.Sub(lastActivityWrite) >= 5*time.Second) {
				if err := saveAgentSidebarActivity(activity); err != nil {
					fmt.Fprintf(os.Stderr, "agent-sidebar: save activity: %v\n", err)
				} else {
					lastActivityWrite = now
				}
			}
			labels := agentSidebarWorkspaceLabels(ctx, c)
			layouts := agentSidebarLayouts(response.Result.Agents, activity, labels)
			current := make(map[string]string, len(response.Result.Agents))
			for _, agent := range response.Result.Agents {
				if agent.PaneID == "" {
					continue
				}
				vendor := strings.ToLower(strings.TrimSpace(agent.Agent))
				if _, ok := agentSidebarLogoTokens[vendor]; !ok {
					vendor = "other"
				}
				layout := layouts[agent.PaneID]
				current[agent.PaneID] = vendor + "\x00" + layout.SortKey + "\x00" + layout.WSKey + "\x00" + layout.TabKey + "\x00" + layout.Group + "\x00" + layout.Gap + "\x00" + layout.Indent + "\x00" + layout.AgentIndex
			}
			next := make(map[string]string, len(current))
			for _, agent := range response.Result.Agents {
				vendor := strings.ToLower(strings.TrimSpace(agent.Agent))
				if _, ok := agentSidebarLogoTokens[vendor]; !ok {
					vendor = "other"
				}
				layout := layouts[agent.PaneID]
				if last[agent.PaneID] == current[agent.PaneID] {
					next[agent.PaneID] = current[agent.PaneID]
					continue
				}
				if err := reportAgentSidebarMetadata(ctx, c, agent.PaneID, vendor, layout); err != nil {
					fmt.Fprintf(os.Stderr, "agent-sidebar: report metadata for %s: %v\n", agent.PaneID, err)
					continue
				}
				next[agent.PaneID] = current[agent.PaneID]
			}
			for pane := range last {
				if _, ok := current[pane]; !ok {
					if err := reportAgentSidebarMetadata(ctx, c, pane, "", sidebarLayout{}); err != nil {
						if strings.Contains(err.Error(), "pane_not_found") {
							continue
						}
						fmt.Fprintf(os.Stderr, "agent-sidebar: clear metadata for %s: %v\n", pane, err)
						next[pane] = last[pane]
					}
				}
			}
			last = next
		} else {
			viewSet = false
		}
		select {
		case <-ctx.Done():
			_ = saveAgentSidebarActivity(activity)
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func claimAgentSidebarDaemon() (string, string, error) {
	activity, err := agentSidebarActivityPath()
	if err != nil {
		return "", "", err
	}
	claim := filepath.Join(filepath.Dir(activity), "daemon.pid")
	if err := os.MkdirAll(filepath.Dir(claim), 0o700); err != nil {
		return "", "", err
	}
	owner := strconv.Itoa(os.Getpid())
	if err := os.WriteFile(claim, []byte(owner), 0o600); err != nil {
		return "", "", err
	}
	return claim, owner, nil
}

func agentSidebarClaimedBy(claim, owner string) bool {
	current, err := os.ReadFile(claim)
	return err == nil && string(current) == owner
}

func loadAgentSidebarActivity() map[string]int64 {
	activity := map[string]int64{}
	file, err := agentSidebarActivityPath()
	if err != nil {
		return activity
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return activity
	}
	_ = json.Unmarshal(data, &activity)
	return activity
}

func saveAgentSidebarActivity(activity map[string]int64) error {
	file, err := agentSidebarActivityPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o600)
}

func agentSidebarActivityPath() (string, error) {
	if value := os.Getenv("HERDR_PLUGIN_STATE_DIR"); value != "" {
		return filepath.Join(value, "activity.json"), nil
	}
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		root = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(root, "herdr", "plugins", "sunznx.herdr-agent-sidebar", "activity.json"), nil
}

func agentSidebarWorkspaceLabels(ctx context.Context, c herdr.Client) map[string]string {
	var response struct {
		Result struct {
			Workspaces []sidebarWorkspace `json:"workspaces"`
		} `json:"result"`
	}
	if err := c.JSON(ctx, &response, "workspace", "list"); err != nil {
		return nil
	}
	labels := make(map[string]string, len(response.Result.Workspaces))
	for _, workspace := range response.Result.Workspaces {
		if workspace.WorkspaceID != "" {
			labels[workspace.WorkspaceID] = workspace.Label
		}
	}
	return labels
}

func agentSidebarLayouts(agents []sidebarAgent, activity map[string]int64, labels map[string]string) map[string]sidebarLayout {
	workspaceMax := map[string]int64{}
	tabMax := map[string]int64{}
	for _, agent := range agents {
		lastActive := activity[agent.PaneID]
		if agent.WorkspaceID != "" && lastActive > workspaceMax[agent.WorkspaceID] {
			workspaceMax[agent.WorkspaceID] = lastActive
		}
		if agent.TabID != "" && lastActive > tabMax[agent.TabID] {
			tabMax[agent.TabID] = lastActive
		}
	}
	layouts := make(map[string]sidebarLayout, len(agents))
	for _, agent := range agents {
		layout := sidebarLayout{
			SortKey: sidebarActivityKey(activity[agent.PaneID], agent.PaneID),
		}
		if agent.WorkspaceID != "" {
			layout.WSKey = sidebarActivityKey(workspaceMax[agent.WorkspaceID], agent.WorkspaceID)
		}
		if agent.TabID != "" {
			layout.TabKey = sidebarActivityKey(tabMax[agent.TabID], agent.TabID)
		}
		layouts[agent.PaneID] = layout
	}
	ordered := append([]sidebarAgent(nil), agents...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := layouts[ordered[i].PaneID], layouts[ordered[j].PaneID]
		if a.WSKey != b.WSKey {
			return a.WSKey > b.WSKey
		}
		if a.TabKey != b.TabKey {
			return a.TabKey > b.TabKey
		}
		if a.SortKey != b.SortKey {
			return a.SortKey > b.SortKey
		}
		return ordered[i].PaneID < ordered[j].PaneID
	})
	seen := map[string]bool{}
	last := map[string]string{}
	agentIndex := 0
	for _, agent := range ordered {
		layout := layouts[agent.PaneID]
		agentIndex++
		layout.AgentIndex = strconv.Itoa(agentIndex)
		if agent.WorkspaceID == "" {
			layouts[agent.PaneID] = layout
			continue
		}
		if !seen[agent.WorkspaceID] {
			layout.Group = labels[agent.WorkspaceID]
			if layout.Group == "" {
				layout.Group = agent.WorkspaceID
			}
			seen[agent.WorkspaceID] = true
		} else {
			layout.Indent = "\u200b  "
		}
		layouts[agent.PaneID] = layout
		last[agent.WorkspaceID] = agent.PaneID
	}
	for _, pane := range last {
		layout := layouts[pane]
		layout.Gap = "\u200b"
		layouts[pane] = layout
	}
	return layouts
}

func sidebarActivityKey(at int64, id string) string {
	return fmt.Sprintf("%012d-%s", at/60000, id)
}

func reportAgentSidebarMetadata(ctx context.Context, c herdr.Client, pane, vendor string, layout sidebarLayout) error {
	args := []string{"pane", "report-metadata", pane, "--source", agentSidebarSource}
	for _, token := range []string{"agent_logo_claude", "agent_logo_codex", "agent_logo_gemini", "agent_logo_opencode", "agent_logo_other", "group", "gap", "sort_key", "ws_key", "tab_key", "agent_index"} {
		args = append(args, "--clear-token", token)
	}
	if vendor != "" {
		args = append(args, "--token", agentSidebarLogoTokens[vendor]+"="+agentSidebarGlyph(vendor))
	}
	for token, value := range map[string]string{"group": layout.Group, "gap": layout.Gap, "sort_key": layout.SortKey, "ws_key": layout.WSKey, "tab_key": layout.TabKey, "agent_index": layout.AgentIndex} {
		if value != "" {
			if token == "agent_index" {
				value = layout.Indent + value
			}
			args = append(args, "--token", token+"="+value)
		}
	}
	_, err := c.Run(ctx, args...)
	return err
}

func agentSidebarGlyph(vendor string) string {
	switch vendor {
	case "claude":
		return "C"
	case "codex":
		return "X"
	case "gemini":
		return "G"
	case "opencode":
		return "O"
	default:
		return "•"
	}
}

func agentSidebar(ctx context.Context, c herdr.Client, enabled bool, reload bool) error {
	file, err := agentSidebarConfigPath()
	if err != nil {
		return err
	}
	current, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read Herdr config: %w", err)
	}
	next, err := updateAgentSidebar(string(current), enabled)
	if err != nil {
		return err
	}
	if next == string(current) {
		if !enabled {
			_ = clearAgentSidebarView(ctx)
		}
		return nil
	}
	if err := os.WriteFile(file+".herdr-agent-sidebar.bak", current, 0o600); err != nil {
		return fmt.Errorf("backup Herdr config: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(file), ".herdr-agent-sidebar-*")
	if err != nil {
		return fmt.Errorf("create temporary Herdr config: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set temporary Herdr config permissions: %w", err)
	}
	if _, err := temp.WriteString(next); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write Herdr config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary Herdr config: %w", err)
	}
	if err := os.Rename(tempName, file); err != nil {
		return fmt.Errorf("replace Herdr config: %w", err)
	}
	if reload {
		if !enabled {
			_ = clearAgentSidebarView(ctx)
		}
		if !herdrExecutableAvailable(c.Bin) {
			fmt.Fprintln(os.Stderr, "agent-sidebar: herdr not found; configuration will be applied when herdr is installed")
			return nil
		}
		return reloadHerdrConfig(ctx, c)
	}
	return nil
}

func herdrExecutableAvailable(bin string) bool {
	if strings.ContainsRune(bin, filepath.Separator) {
		info, err := os.Stat(bin)
		return err == nil && info.Mode().IsRegular()
	}
	_, err := exec.LookPath(bin)
	return err == nil
}

func setAgentSidebarView(ctx context.Context) error {
	return agentSidebarViewRequest(ctx, "agent.view.set", map[string]any{
		"source": agentSidebarSource,
		"label":  "active",
		"sort": []map[string]any{
			{"field": map[string]string{"token": "ws_key"}, "order": "desc"},
			{"field": map[string]string{"token": "tab_key"}, "order": "desc"},
			{"field": map[string]string{"token": "sort_key"}, "order": "desc"},
		},
	})
}

func clearAgentSidebarView(ctx context.Context) error {
	return agentSidebarViewRequest(ctx, "agent.view.clear", map[string]any{
		"source": agentSidebarSource,
	})
}

func agentSidebarViewRequest(ctx context.Context, method string, params any) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("agent view socket is not supported on Windows")
	}
	socketPath, err := agentSidebarSocketPath()
	if err != nil {
		return err
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("connect Herdr socket: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	request := map[string]any{
		"id":     fmt.Sprintf("agent-sidebar-%d", time.Now().UnixNano()),
		"method": method,
		"params": params,
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return fmt.Errorf("send %s: %w", method, err)
	}
	var response struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&response); err != nil {
		return fmt.Errorf("read %s response: %w", method, err)
	}
	if len(response.Error) > 0 && string(response.Error) != "null" {
		return fmt.Errorf("Herdr %s failed: %s", method, response.Error)
	}
	return nil
}

func agentSidebarSocketPath() (string, error) {
	if value := os.Getenv("HERDR_SOCKET_PATH"); value != "" {
		return value, nil
	}
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "herdr", "herdr.sock"), nil
}

func agentSidebarConfigPath() (string, error) {
	if value := os.Getenv("HERDR_CONFIG_PATH"); value != "" {
		return value, nil
	}
	if runtime.GOOS == "windows" {
		root := os.Getenv("APPDATA")
		if root == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve home directory: %w", err)
			}
			root = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(root, "herdr", "config.toml"), nil
	}
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "herdr", "config.toml"), nil
}

func updateAgentSidebar(text string, enabled bool) (string, error) {
	managed := regexp.MustCompile(regexp.QuoteMeta(agentSidebarStart) + `[\s\S]*?` + regexp.QuoteMeta(agentSidebarEnd) + `\n?`)
	if !enabled {
		return strings.TrimSuffix(managed.ReplaceAllString(text, ""), "\n"), nil
	}
	if managed.MatchString(text) {
		return managed.ReplaceAllStringFunc(text, func(string) string { return agentSidebarBlock + "\n" }), nil
	}
	if regexp.MustCompile(`(?m)^\[ui\.sidebar\.agents(?:\.rows_by_agent)?\]\s*$`).MatchString(text) {
		return "", fmt.Errorf("config.toml already contains [ui.sidebar.agents]; refusing to overwrite it")
	}
	return strings.TrimRight(text, "\r\n") + "\n\n" + agentSidebarBlock + "\n", nil
}

func reloadHerdrConfig(ctx context.Context, c herdr.Client) error {
	if _, err := c.Run(ctx, "server", "reload-config"); err != nil {
		return err
	}
	return nil
}

func agentSidebarSelfTest() error {
	base := "onboarding = false\n"
	installed, err := updateAgentSidebar(base, true)
	if err != nil || !strings.Contains(installed, "[ui.sidebar.agents]") {
		return fmt.Errorf("install self-test failed")
	}
	if !strings.Contains(installed, `["pane"]`) || strings.Contains(installed, "terminal_title_stripped") {
		return fmt.Errorf("Auto Title pane label is not configured")
	}
	if again, _ := updateAgentSidebar(installed, true); again != installed {
		return fmt.Errorf("install is not idempotent")
	}
	removed, err := updateAgentSidebar(installed, false)
	if err != nil || removed != base {
		return fmt.Errorf("remove self-test failed")
	}
	if _, err := updateAgentSidebar("[ui.sidebar.agents]\nrows = []\n", true); err == nil {
		return fmt.Errorf("conflict self-test failed")
	}
	return nil
}
