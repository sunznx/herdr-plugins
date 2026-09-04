package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Copy(ctx context.Context, text string) error {
	bin := os.Getenv("HERDR_COPY_CLIPBOARD_BIN")
	if bin == "" {
		bin = "/usr/bin/pbcopy"
	}
	cmd := exec.CommandContext(ctx, bin)
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not write to clipboard: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}
