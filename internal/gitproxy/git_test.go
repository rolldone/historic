package gitproxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenIsIdempotentAndNoRemote(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".database")
	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(root)
	if err != nil || first.Root != second.Root {
		t.Fatalf("second Open = %#v %v", second, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	if _, err := second.run("remote"); err != nil {
		t.Fatal(err)
	}
}
