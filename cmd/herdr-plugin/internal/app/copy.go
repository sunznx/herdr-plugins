package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/clipboard"
	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func copyCurrentDir(ctx context.Context, c herdr.Client) error {
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	cwd := herdr.PaneCWD(pane)
	if cwd == "" {
		return fmt.Errorf("current pane has no directory")
	}
	return clipboard.Copy(ctx, cwd)
}

func copyCurrentAgentSession(ctx context.Context, c herdr.Client) error {
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	if pane.AgentSession == nil || pane.AgentSession.Value == "" {
		return fmt.Errorf("current pane has no agent session")
	}
	return clipboard.Copy(ctx, pane.AgentSession.Value)
}

func openCurrentAgentSession(ctx context.Context, c herdr.Client, fork bool) error {
	pane, err := herdr.OriginPane(ctx, c)
	if err != nil {
		return err
	}
	return openAgentSession(ctx, c, pane, fork)
}

func openAgentSession(ctx context.Context, c herdr.Client, pane herdr.Pane, fork bool) error {
	if pane.AgentSession == nil || pane.AgentSession.Agent == "" || pane.AgentSession.Value == "" {
		return fmt.Errorf("current pane has no agent session")
	}
	argv, err := agentSessionArgv(*pane.AgentSession, fork)
	if err != nil {
		return err
	}
	create := []string{"tab", "create"}
	if pane.WorkspaceID != "" {
		create = append(create, "--workspace", pane.WorkspaceID)
	}
	if cwd := herdr.PaneCWD(pane); cwd != "" {
		create = append(create, "--cwd", cwd)
	}
	create = append(create, "--focus")
	out, err := c.Run(ctx, create...)
	if err != nil {
		return err
	}
	var response struct {
		Result struct {
			RootPane herdr.Pane `json:"root_pane"`
		} `json:"result"`
	}
	if err := decode(out, &response); err != nil {
		return err
	}
	if response.Result.RootPane.PaneID == "" {
		return fmt.Errorf("Herdr did not return a root pane")
	}
	command := append([]string{"pane", "run", response.Result.RootPane.PaneID}, argv...)
	_, err = c.Run(ctx, command...)
	return err
}

func agentSessionArgv(session herdr.AgentSession, fork bool) ([]string, error) {
	if session.Value == "" {
		return nil, fmt.Errorf("agent session value is empty")
	}
	if fork {
		switch {
		case session.Source == "herdr:codex" && session.Agent == "codex" && session.Kind == "id":
			return []string{"codex", "fork", session.Value}, nil
		case session.Source == "herdr:claude" && session.Agent == "claude" && session.Kind == "id":
			return []string{"claude", "--resume", session.Value, "--fork-session"}, nil
		default:
			return nil, fmt.Errorf("fork is unsupported for agent session %s/%s", session.Source, session.Agent)
		}
	}
	switch {
	case session.Source == "herdr:claude" && session.Agent == "claude" && session.Kind == "id":
		return []string{"claude", "--resume", session.Value}, nil
	case session.Source == "herdr:codex" && session.Agent == "codex" && session.Kind == "id":
		return []string{"codex", "resume", session.Value}, nil
	case session.Source == "herdr:copilot" && session.Agent == "copilot" && session.Kind == "id":
		return []string{"copilot", "--resume=" + session.Value}, nil
	case session.Source == "herdr:devin" && session.Agent == "devin" && session.Kind == "id":
		return []string{"devin", "--resume", session.Value}, nil
	case session.Source == "herdr:droid" && session.Agent == "droid" && session.Kind == "id":
		return []string{"droid", "--resume", session.Value}, nil
	case session.Source == "herdr:kimi" && session.Agent == "kimi" && session.Kind == "id":
		return []string{"kimi", "--session", session.Value}, nil
	case session.Source == "herdr:mastracode" && session.Agent == "mastracode" && session.Kind == "id":
		return []string{"mastracode", "--thread", session.Value}, nil
	case session.Source == "herdr:pi" && session.Agent == "pi" && (session.Kind == "id" || session.Kind == "path"):
		return []string{"pi", "--session", session.Value}, nil
	case session.Source == "herdr:omp" && session.Agent == "omp" && (session.Kind == "id" || session.Kind == "path"):
		return []string{"omp", "--resume=" + session.Value}, nil
	case session.Source == "herdr:hermes" && session.Agent == "hermes" && session.Kind == "id":
		return []string{"hermes", "--resume", session.Value}, nil
	case session.Source == "herdr:opencode" && session.Agent == "opencode" && session.Kind == "id":
		return []string{"opencode", "--session", session.Value}, nil
	case session.Source == "herdr:qodercli" && session.Agent == "qodercli" && session.Kind == "id":
		return []string{"qodercli", "--resume", session.Value}, nil
	case session.Source == "herdr:kilo" && session.Agent == "kilo" && session.Kind == "id":
		return []string{"kilo", "--session", session.Value}, nil
	case session.Source == "herdr:cursor" && session.Agent == "cursor" && session.Kind == "id":
		return []string{"cursor-agent", "--resume", session.Value}, nil
	case session.Source == "herdr:antigravity_cli" && session.Agent == "agy" && session.Kind == "id":
		return []string{"agy", "--conversation", session.Value}, nil
	case session.Source == "herdr:grok" && session.Agent == "grok" && session.Kind == "id":
		return []string{"grok", "--resume", session.Value}, nil
	default:
		return nil, fmt.Errorf("resume is unsupported for agent session %s/%s", session.Source, session.Agent)
	}
}

func copyLast(ctx context.Context, c herdr.Client, includeCommand bool) error {
	paneID := herdr.PluginContext().FocusedPaneID
	if paneID == "" {
		paneID = os.Getenv("HERDR_PANE_ID")
	}
	if paneID == "" {
		paneID = os.Getenv("HERDR_ACTIVE_PANE_ID")
	}
	if paneID == "" {
		return fmt.Errorf("could not resolve the triggering pane")
	}
	scrollback, err := c.Run(ctx, "pane", "read", paneID, "--source", "recent-unwrapped", "--lines", "10000", "--format", "text")
	if err != nil {
		return err
	}
	content, err := lastCommandOutput(string(scrollback), includeCommand)
	if err != nil {
		return err
	}
	return clipboard.Copy(ctx, content)
}

func lastCommandOutput(scrollback string, includeCommand bool) (string, error) {
	lines := strings.Split(strings.ReplaceAll(scrollback, "\r\n", "\n"), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) < 2 {
		return "", fmt.Errorf("could not identify a completed command with non-empty output")
	}
	promptLine := len(lines) - 1
	prefix := strings.TrimRightFunc(lines[promptLine], unicode.IsSpace)
	commandLine, command := -1, ""
	robbyEmpty := robbyRussellCommand(prefix) == ""
	if strings.HasPrefix(prefix, "➜ ") && !robbyEmpty {
		return "", fmt.Errorf("could not identify a completed command with non-empty output")
	}
	if !strings.HasPrefix(prefix, "➜ ") {
		for i := promptLine - 1; i >= 0; i-- {
			if strings.HasPrefix(lines[i], prefix) {
				rest := strings.TrimLeftFunc(strings.TrimPrefix(lines[i], prefix), unicode.IsSpace)
				if rest != "" && len(lines[i]) > len(prefix) && unicode.IsSpace(rune(lines[i][len(prefix)])) {
					commandLine, command = i, rest
					break
				}
			}
		}
	} else if robbyEmpty {
		for i := promptLine - 1; i >= 0; i-- {
			if candidate := robbyRussellCommand(lines[i]); candidate != "" {
				commandLine, command = i, candidate
				break
			}
		}
	}
	if commandLine < 0 {
		marker := promptMarker(prefix)
		for i := promptLine - 1; marker != "" && i >= 0; i-- {
			if candidate := commandAfterMarker(lines[i], marker); candidate != "" {
				commandLine, command = i, candidate
				break
			}
		}
	}
	start, end := commandLine+1, promptLine-1
	if commandLine < 0 {
		return "", fmt.Errorf("could not identify a completed command with non-empty output")
	}
	if end >= start {
		trimmed := strings.TrimLeftFunc(lines[end], unicode.IsSpace)
		if strings.HasPrefix(trimmed, "╭") || strings.HasPrefix(trimmed, "┌") || strings.HasPrefix(trimmed, "┏") {
			end--
		}
	}
	for start <= end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end >= start && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if start > end {
		return "", fmt.Errorf("could not identify a completed command with non-empty output")
	}
	result := append([]string(nil), lines[start:end+1]...)
	if includeCommand {
		result = append([]string{command}, result...)
	}
	return strings.Join(result, "\n"), nil
}

func promptMarker(line string) string {
	line = strings.TrimSpace(line)
	for _, marker := range []string{"❯", "➜", "$", "%", "#", ">"} {
		if strings.HasSuffix(line, marker) {
			return marker
		}
	}
	return ""
}

func commandAfterMarker(line, marker string) string {
	for offset := 0; ; {
		index := strings.Index(line[offset:], marker)
		if index < 0 {
			return ""
		}
		index += offset
		rest := line[index+len(marker):]
		if len(rest) > 0 {
			first, _ := utf8.DecodeRuneInString(rest)
			if unicode.IsSpace(first) && strings.TrimSpace(rest) != "" {
				return strings.TrimLeftFunc(rest, unicode.IsSpace)
			}
		}
		offset = index + len(marker)
	}
}

func robbyRussellCommand(line string) string {
	if !strings.HasPrefix(line, "➜ ") {
		return ""
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "➜ "))
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return ""
	}
	rest = strings.TrimSpace(strings.TrimPrefix(rest, parts[0]))
	if strings.HasPrefix(rest, "git:(") {
		close := strings.Index(rest, ")")
		if close < 0 {
			return ""
		}
		rest = strings.TrimSpace(rest[close+1:])
		if strings.HasPrefix(rest, "✗") {
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "✗"))
		}
	}
	return rest
}
