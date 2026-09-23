package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWithinResolvesExternalSymlinkIntoRepository(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "generated")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	link := filepath.Join(outside, "caller-output")
	if err := os.Symlink(inside, link); err != nil {
		t.Fatal(err)
	}
	if !isWithin(root, link) {
		t.Fatal("isWithin failed to resolve symlink target")
	}
}
