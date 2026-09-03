package fzf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Pick(ctx context.Context, rows string, args ...string) (string, error) {
	bin, err := exec.LookPath("fzf")
	if err != nil {
		return "", errors.New("fzf is not installed or not on PATH")
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(rows)
	cmd.Stderr = os.Stderr
	cmd.Env = Environment()
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 1 || exitErr.ExitCode() == 130) {
			return "", nil
		}
		return "", fmt.Errorf("fzf: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

func Run(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = Environment()
	return cmd.Run()
}

func Environment() []string {
	env := os.Environ()
	configured := os.Getenv("FZF_DEFAULT_OPTS_FILE")
	if configured == "" {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}
		configured = filepath.Join(base, "fzf", "fzfrc")
	}
	if info, err := os.Stat(configured); err != nil || info.IsDir() {
		configured = ""
	}
	return SetEnv(env, "FZF_DEFAULT_OPTS_FILE", configured)
}

func SetEnv(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	if value != "" {
		result = append(result, prefix+value)
	}
	return result
}
