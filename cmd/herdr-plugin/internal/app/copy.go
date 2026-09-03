package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

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
	clipboard := envOr("HERDR_COPY_CLIPBOARD_BIN", "/usr/bin/pbcopy")
	cmd := exec.CommandContext(ctx, clipboard)
	cmd.Stdin = strings.NewReader(content)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not write to clipboard: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
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
