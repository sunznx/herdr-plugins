package app

import "testing"

func TestIsModelQuotaError(t *testing.T) {
	for _, message := range []string{
		"You've hit your usage limit for GPT-5.3-Codex-Spark",
		"quota exceeded",
		"request exceeded your limit",
	} {
		if !isModelQuotaError(message) {
			t.Errorf("expected quota error for %q", message)
		}
	}
	if isModelQuotaError("invalid API key") {
		t.Fatal("authentication errors must not trigger model fallback")
	}
}
