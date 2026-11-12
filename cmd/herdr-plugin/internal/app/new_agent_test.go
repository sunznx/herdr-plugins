package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAgentPickerListsCodexFirst(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_NEW_AGENT_KIND", "")
	bin := t.TempDir()
	for _, kind := range []string{"claude", "codex"} {
		if err := os.WriteFile(filepath.Join(bin, kind), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	rows := filepath.Join(t.TempDir(), "rows")
	t.Setenv("TEST_AGENT_ROWS", rows)
	if err := os.WriteFile(filepath.Join(bin, "fzf"), []byte("#!/bin/sh\n/bin/cat > \"$TEST_AGENT_ROWS\"\nexit 130\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	root := NewCommand()
	root.SetArgs([]string{"new", "picker", "agent"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rows)
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)[0]
	if got := strings.Fields(first); len(got) == 0 || got[0] != "codex" {
		t.Fatalf("Codex must be the first agent candidate; rows:\n%s", data)
	}
}

func TestNewAgentPickerStartsDoubaoInSelectedWorkspace(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "doubao"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "fzf"), []byte("#!/bin/sh\n/usr/bin/awk -F '\\t' '$1 == \"doubao\" { print }'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("HERDR_TEST_REPO", t.TempDir())
	t.Setenv("HERDR_NEW_AGENT_KIND", "")
	t.Setenv("HERDR_NEW_AGENT_CHOICE", "__workspace__:w1")
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	root := NewCommand()
	root.SetArgs([]string{"new", "picker", "agent"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "tab create --workspace w1 ") || !strings.Contains(string(data), "pane run w1:p2 exec doubao\n") {
		t.Fatalf("Doubao must start in a new tab in the selected workspace; calls:\n%s", data)
	}
}

func TestNewAgentPickerInstallsMissingAgentBeforeOpeningTab(t *testing.T) {
	for kind, installer := range map[string]struct{ bin, args string }{
		"codex":    {"npm", "install -g @openai/codex"},
		"claude":   {"brew", "install --cask claude-code"},
		"gemini":   {"npm", "install -g @google/gemini-cli"},
		"opencode": {"npm", "install -g @opencode/cli"},
		"doubao":   {"npm", "install -g @bytedance/doubao-cli --registry=https://registry.npmjs.org/"},
	} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("HERDR_GO_TEST_HELPER", "1")
			t.Setenv("HERDR_BIN_PATH", os.Args[0])
			t.Setenv("HERDR_NEW_AGENT_KIND", "")
			t.Setenv("HERDR_NEW_AGENT_CHOICE", "__workspace__:w1")
			t.Setenv("HERDR_TEST_REPO", t.TempDir())
			bin := t.TempDir()
			t.Setenv("PATH", bin)
			t.Setenv("TEST_AGENT_BIN", bin)
			t.Setenv("TEST_INSTALL_KIND", kind)
			calls := filepath.Join(t.TempDir(), "calls")
			t.Setenv("HERDR_GO_TEST_CALLS", calls)
			installCall := filepath.Join(t.TempDir(), "install")
			t.Setenv("TEST_INSTALL_CALL", installCall)
			script := `#!/bin/sh
printf '%s\n' "$*" > "$TEST_INSTALL_CALL"
printf '#!/bin/sh\n' > "$TEST_AGENT_BIN/$TEST_INSTALL_KIND"
/bin/chmod +x "$TEST_AGENT_BIN/$TEST_INSTALL_KIND"
`
			if err := os.WriteFile(filepath.Join(bin, installer.bin), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			fzf := "#!/bin/sh\n/usr/bin/awk -F '\\t' -v kind=\"$TEST_INSTALL_KIND\" '$1 == kind { print }'\n"
			if err := os.WriteFile(filepath.Join(bin, "fzf"), []byte(fzf), 0o755); err != nil {
				t.Fatal(err)
			}
			root := NewCommand()
			root.SetArgs([]string{"new", "picker", "agent"})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(installCall)
			if err != nil || strings.TrimSpace(string(data)) != installer.args {
				t.Fatalf("missing %s must run its official installer %q; call=%q err=%v", kind, installer.args, data, err)
			}
			data, err = os.ReadFile(calls)
			if err != nil || !strings.Contains(string(data), "tab create --workspace w1 ") || !strings.Contains(string(data), "pane run w1:p2 exec "+kind+"\n") {
				t.Fatalf("successful installation must continue to the selected workspace and agent; calls=%s err=%v", data, err)
			}
		})
	}
}

func TestNewAgentPickerListsAgentsWithoutInstructions(t *testing.T) {
	t.Setenv("HERDR_GO_TEST_HELPER", "1")
	t.Setenv("HERDR_BIN_PATH", os.Args[0])
	t.Setenv("HERDR_NEW_AGENT_KIND", "")
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	rows := filepath.Join(t.TempDir(), "rows")
	args := filepath.Join(t.TempDir(), "args")
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("TEST_AGENT_ROWS", rows)
	t.Setenv("TEST_AGENT_ARGS", args)
	t.Setenv("HERDR_GO_TEST_CALLS", calls)
	fzf := `#!/bin/sh
printf '%s\n' "$@" > "$TEST_AGENT_ARGS"
/bin/cat > "$TEST_AGENT_ROWS"
exit 130
`
	if err := os.WriteFile(filepath.Join(bin, "fzf"), []byte(fzf), 0o755); err != nil {
		t.Fatal(err)
	}
	root := NewCommand()
	root.SetArgs([]string{"new", "picker", "agent"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rows)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 5 || !strings.HasPrefix(lines[0], "codex\t") {
		t.Fatalf("the list must start with Codex and offer multiple agents:\n%s", data)
	}
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 2 || fields[1] != fields[0]+" [未安装]" {
			t.Fatalf("agent rows must contain only the name and installation status; got %q", line)
		}
	}
	data, err = os.ReadFile(args)
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.HasPrefix(arg, "--preview=") && arg != "--preview=" || strings.HasPrefix(arg, "--header=") && arg != "--header=" {
			t.Fatalf("the small popup must devote its space to the list, without preview or instructions; got %q", arg)
		}
	}
	if data, _ := os.ReadFile(calls); strings.Contains(string(data), "tab create") || strings.Contains(string(data), "workspace list") || strings.Contains(string(data), "pane run") {
		t.Fatalf("cancelling the agent list must leave workspaces untouched:\n%s", data)
	}
}

func TestNewAgentPickerDoesNotOpenWorkspaceWhenInstallationFails(t *testing.T) {
	for name, result := range map[string]string{"installer fails": "exit 1", "executable missing": "exit 0"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("HERDR_GO_TEST_HELPER", "1")
			t.Setenv("HERDR_BIN_PATH", os.Args[0])
			t.Setenv("HERDR_NEW_AGENT_KIND", "doubao")
			bin := t.TempDir()
			t.Setenv("PATH", bin)
			calls := filepath.Join(t.TempDir(), "calls")
			t.Setenv("HERDR_GO_TEST_CALLS", calls)
			installCall := filepath.Join(t.TempDir(), "install")
			t.Setenv("TEST_INSTALL_CALL", installCall)
			script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$TEST_INSTALL_CALL\"\n" + result + "\n"
			if err := os.WriteFile(filepath.Join(bin, "npm"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			root := NewCommand()
			root.SetArgs([]string{"new", "picker", "agent"})
			if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "doubao") {
				t.Fatalf("an unsuccessful installation must stop with the agent name; got %v", err)
			}
			if _, err := os.Stat(installCall); err != nil {
				t.Fatalf("the selected missing agent must attempt installation: %v", err)
			}
			if data, _ := os.ReadFile(calls); strings.Contains(string(data), "workspace list") || strings.Contains(string(data), "tab create") || strings.Contains(string(data), "pane run") {
				t.Fatalf("failed installation must stop before workspace selection:\n%s", data)
			}
		})
	}
}
