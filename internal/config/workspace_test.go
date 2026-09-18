package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeCreatesWorkspaceAndIndex(t *testing.T) {
	root := t.TempDir()
	workspace, err := Initialize(root)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	for _, path := range []string{workspace.Histories, workspace.Database, workspace.Index} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("stat %s: %v", path, err)
		}
	}
}

func TestInitializeIsIdempotent(t *testing.T) {
	root := t.TempDir()
	workspace, err := Initialize(root)
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00014-topic", "note.md")
	if err := os.MkdirAll(filepath.Dir(topic), 0o755); err != nil {
		t.Fatal(err)
	}
	const content = "preserve me\n"
	if err := os.WriteFile(topic, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(root); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	got, err := os.ReadFile(topic)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("topic changed: %q", got)
	}
}

func TestInitializeRejectsHistoriesFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, HistoriesDirName), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Initialize(root)
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("error = %v, want ErrNotDirectory", err)
	}
}

func TestInitializeRejectsLegacyWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, LegacyHistoriesDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Initialize(root)
	if !errors.Is(err, ErrLegacyWorkspace) {
		t.Fatalf("error = %v, want ErrLegacyWorkspace", err)
	}
}

func TestInitializeRejectsCanonicalAndLegacyWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, HistoricDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, LegacyHistoriesDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Initialize(root)
	if !errors.Is(err, ErrWorkspaceConflict) {
		t.Fatalf("error = %v, want ErrWorkspaceConflict", err)
	}
}

func TestDiscoverRootFindsInitializedAncestor(t *testing.T) {
	root := t.TempDir()
	if _, err := Initialize(root); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested", "child")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("root = %q, want %q", got, root)
	}
}
