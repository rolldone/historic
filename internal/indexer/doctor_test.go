package indexer

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestUpgradeMigratesLegacyIndexAndPreservesMarkdown(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-21"}, "body")
	legacyPath := filepath.Join(topic, markdown.LegacyMetaFilename)
	if err := markdown.WriteFile(legacyPath, document); err != nil {
		t.Fatal(err)
	}
	before := checksum(legacyPath)
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DROP TABLE historic_metadata"); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	count, _, err := Upgrade(workspace)
	if err != nil || count != 1 {
		t.Fatalf("upgrade count=%d err=%v", count, err)
	}
	canonicalPath := filepath.Join(topic, markdown.MetaFilename)
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy metadata remains after upgrade: %v", err)
	}
	canonical, err := markdown.ParseTopicMetadataFile(canonicalPath)
	if err != nil || canonical.ID != "00001" || canonical.Title != "Topic" || canonical.Status != domain.StatusProgress {
		t.Fatalf("migrated metadata = %#v err=%v", canonical, err)
	}
	if checksum(legacyPath) == before {
		t.Fatal("legacy checksum unexpectedly available after migration")
	}
	version, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer version.Close()
	var got int
	if err := version.QueryRow("SELECT value FROM historic_metadata WHERE key='schema_version'").Scan(&got); err != nil || got != SchemaVersion {
		t.Fatalf("schema version=%d err=%v", got, err)
	}
}

func TestUpgradeRejectsConcurrentLock(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireUpgradeLock(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, _, err := Upgrade(workspace); err != ErrUpgradeLocked {
		t.Fatalf("err=%v, want %v", err, ErrUpgradeLocked)
	}
}

func checksum(path string) string {
	data, _ := os.ReadFile(path)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
