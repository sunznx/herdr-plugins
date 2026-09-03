package app

import "testing"

func TestLastCommandOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		command bool
		want    string
	}{
		{"plain", "$ printf hello\nhello\n$ \n", false, "hello"},
		{"with command", "$ printf hello\nhello\n$ \n", true, "printf hello\nhello"},
		{"robbyrussell", "➜ repo git:(main) printf hello\nhello\n➜ repo git:(main)\n", true, "printf hello\nhello"},
		{"dirty robbyrussell", "➜ repo git:(main) touch x\ndirty\n➜ repo git:(main) ✗\n", false, "dirty"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := lastCommandOutput(test.input, test.command)
			if err != nil || got != test.want {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
		})
	}
}
