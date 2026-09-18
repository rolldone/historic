package lifecycle

import (
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
)

func TestValidateRestorePath(t *testing.T) {
	for _, path := range []string{"", "../topic", "/tmp/topic", "topic/file"} {
		if err := validateRestorePath(path); err == nil {
			t.Errorf("validateRestorePath(%q) accepted", path)
		}
	}
	if err := validateRestorePath("00001-topic"); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreRejectsEmptySnapshotWithoutChangingWorkspace(t *testing.T) {
	workspace := config.NewWorkspace(t.TempDir())
	if err := Restore(workspace, domain.ID("00001"), "", false); err == nil {
		t.Fatal("empty snapshot accepted")
	}
}
