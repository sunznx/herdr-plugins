package herdr

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ConfigPath returns the active Herdr config path, honoring explicit and XDG overrides.
func ConfigPath() string {
	if path := os.Getenv("HERDR_CONFIG_PATH"); path != "" {
		return path
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "herdr", "config.toml")
}

// ConfigBool reads a boolean from a TOML section. An empty section means the
// top-level table. It intentionally handles only the scalar needed by plugins.
func ConfigBool(path, section, key string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	current := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.Trim(line, "[]")
			continue
		}
		if current != section {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
			return strings.TrimSpace(parts[1]) == "true"
		}
	}
	return false
}
