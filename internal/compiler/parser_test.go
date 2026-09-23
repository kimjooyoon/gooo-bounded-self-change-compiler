package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSourceRejectsDuplicateKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "self-change.gooo")
	source := "gooo bounded_self_change v1\nscenario id=first id=second\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSource(path); err == nil {
		t.Fatal("duplicate source key was accepted")
	}
}
