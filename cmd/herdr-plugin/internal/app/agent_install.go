package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Only offer missing CLIs whose official installation command is known.
func agentInstallCommand(kind string) []string {
	switch kind {
	case "codex":
		return []string{"npm", "install", "-g", "@openai/codex"}
	case "claude":
		return []string{"brew", "install", "--cask", "claude-code"}
	case "gemini":
		return []string{"npm", "install", "-g", "@google/gemini-cli"}
	case "opencode":
		return []string{"npm", "install", "-g", "@opencode/cli"}
	case "doubao":
		return []string{"npm", "install", "-g", "@bytedance/doubao-cli", "--registry=https://registry.npmjs.org/"}
	}
	return nil
}

func installAgent(ctx context.Context, kind string) error {
	command := agentInstallCommand(kind)
	if len(command) == 0 {
		return fmt.Errorf("no installer is available for %s", kind)
	}
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install %s: %w", kind, err)
	}
	if _, err := exec.LookPath(kind); err != nil {
		return fmt.Errorf("%s is unavailable on PATH after installation: %w", kind, err)
	}
	return nil
}
