package zoxide

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestList(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "zoxide")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf '/tmp/one\\n/tmp/two\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"/tmp/one", "/tmp/two"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("paths: got %#v, want %#v", got, want)
	}
}
