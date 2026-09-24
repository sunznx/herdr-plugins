package app

import (
	"reflect"
	"testing"

	"github.com/sunznx/herdr-plugins/cmd/herdr-plugin/internal/herdr"
)

func TestAgentSessionArgv(t *testing.T) {
	tests := []struct {
		name string
		sess herdr.AgentSession
		fork bool
		want []string
	}{
		{"codex resume", herdr.AgentSession{Source: "herdr:codex", Agent: "codex", Kind: "id", Value: "s1"}, false, []string{"codex", "resume", "s1"}},
		{"codex fork", herdr.AgentSession{Source: "herdr:codex", Agent: "codex", Kind: "id", Value: "s1"}, true, []string{"codex", "fork", "s1"}},
		{"claude fork", herdr.AgentSession{Source: "herdr:claude", Agent: "claude", Kind: "id", Value: "s1"}, true, []string{"claude", "--resume", "s1", "--fork-session"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := agentSessionArgv(test.sess, test.fork)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v, %v; want %v", got, err, test.want)
			}
		})
	}
}

func TestAgentSessionArgvRejectsUnsupportedFork(t *testing.T) {
	_, err := agentSessionArgv(herdr.AgentSession{Source: "herdr:opencode", Agent: "opencode", Kind: "id", Value: "s1"}, true)
	if err == nil {
		t.Fatal("expected unsupported fork error")
	}
}
