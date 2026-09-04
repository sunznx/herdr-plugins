package zoxide

import (
	"bufio"
	"context"
	"errors"
	"os/exec"
	"strings"
)

func List(ctx context.Context) ([]string, error) {
	bin, err := exec.LookPath("zoxide")
	if err != nil {
		return nil, errors.New("zoxide is not installed or not on PATH")
	}
	out, err := exec.CommandContext(ctx, bin, "query", "-l").Output()
	if err != nil {
		return nil, err
	}
	var paths []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		if path := strings.TrimSpace(scanner.Text()); path != "" {
			paths = append(paths, path)
		}
	}
	return paths, scanner.Err()
}
