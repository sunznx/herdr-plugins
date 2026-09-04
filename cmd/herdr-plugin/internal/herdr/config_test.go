package herdr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigBoolReadsSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("enable_worktree = true\n[experimental]\nswitch_ascii_input_source_in_prefix = true\nother = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !ConfigBool(path, "", "enable_worktree") {
		t.Fatal("top-level boolean was not read")
	}
	if !ConfigBool(path, "experimental", "switch_ascii_input_source_in_prefix") {
		t.Fatal("section boolean was not read")
	}
	if ConfigBool(path, "", "switch_ascii_input_source_in_prefix") {
		t.Fatal("section key leaked into top-level lookup")
	}
}

func TestConfigPathHonorsOverride(t *testing.T) {
	t.Setenv("HERDR_CONFIG_PATH", "/tmp/herdr-test.toml")
	if got := ConfigPath(); got != "/tmp/herdr-test.toml" {
		t.Fatalf("ConfigPath() = %q", got)
	}
}
