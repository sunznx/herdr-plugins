package fzf

import "testing"

func TestHasArg(t *testing.T) {
	if !hasArg([]string{"--print-query", "--phony"}, "--print-query") {
		t.Fatal("expected argument to be found")
	}
	if hasArg([]string{"--phony"}, "--print-query") {
		t.Fatal("did not expect argument to be found")
	}
}
