package repository

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"

	_ "modernc.org/sqlite"
)

const (
	extendA = "1791851149892"
	extendB = "1792889892200"
	extendC = "1792969715909"
)

func createOpenTopic(t *testing.T, store TopicStore, id, title string) string {
	t.Helper()
	topic, err := store.CreateTopic(title, id, "")
	if err != nil {
		t.Fatal(err)
	}
	return topic.Path
}

func closeTopicDir(t *testing.T, store TopicStore, openPath string) string {
	t.Helper()
	closedPath := filepath.Join(store.Workspace.Database, filepath.Base(openPath))
	if err := os.Rename(openPath, closedPath); err != nil {
		t.Fatal(err)
	}
	return closedPath
}

func readMeta(t *testing.T, topicPath string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(topicPath, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func extendsIDInIndex(t *testing.T, store TopicStore, id, storage string) sql.NullString {
	t.Helper()
	db, err := sql.Open("sqlite", store.Workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var value sql.NullString
	if err := db.QueryRow("SELECT extends_id FROM topics WHERE id = ? AND storage = ?", id, storage).Scan(&value); err != nil {
		t.Fatalf("query extends_id for %s/%s: %v", id, storage, err)
	}
	return value
}

func TestExtendTopicOpenParentUpdatesMetadataAndIndex(t *testing.T) {
	store := newTestStore(t)
	createOpenTopic(t, store, extendA, "Parent")
	childPath := createOpenTopic(t, store, extendB, "Child")

	result, err := store.ExtendTopic(domain.TopicID(extendB), extendA)
	if err != nil {
		t.Fatalf("ExtendTopic: %v", err)
	}
	if result.ID != domain.TopicID(extendB) || result.Extends != domain.TopicID(extendA) {
		t.Fatalf("result = %+v", result)
	}

	metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(childPath, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Extends != domain.TopicID(extendA) {
		t.Fatalf("metadata extends = %q, want %q", metadata.Extends, extendA)
	}
	value := extendsIDInIndex(t, store, extendB, "open")
	if !value.Valid || value.String != extendA {
		t.Fatalf("index extends_id = %#v, want %s", value, extendA)
	}
}

func TestExtendTopicClosedTargetAndClosedParent(t *testing.T) {
	store := newTestStore(t)
	parentOpen := createOpenTopic(t, store, extendA, "Parent")
	childOpen := createOpenTopic(t, store, extendB, "Child")
	closeTopicDir(t, store, parentOpen)
	closeTopicDir(t, store, childOpen)

	result, err := store.ExtendTopic(domain.TopicID(extendB), extendA)
	if err != nil {
		t.Fatalf("ExtendTopic closed: %v", err)
	}
	if result.Storage != domain.StorageClosed {
		t.Fatalf("storage = %q, want closed", result.Storage)
	}
	value := extendsIDInIndex(t, store, extendB, "closed")
	if !value.Valid || value.String != extendA {
		t.Fatalf("closed index extends_id = %#v, want %s", value, extendA)
	}
}

func TestExtendTopicChain(t *testing.T) {
	store := newTestStore(t)
	createOpenTopic(t, store, extendA, "A")
	createOpenTopic(t, store, extendB, "B")
	createOpenTopic(t, store, extendC, "C")

	if _, err := store.ExtendTopic(domain.TopicID(extendB), extendA); err != nil {
		t.Fatalf("extend B: %v", err)
	}
	if _, err := store.ExtendTopic(domain.TopicID(extendC), extendB); err != nil {
		t.Fatalf("extend C: %v", err)
	}
	for id, want := range map[string]string{extendB: extendA, extendC: extendB} {
		value := extendsIDInIndex(t, store, id, "open")
		if !value.Valid || value.String != want {
			t.Fatalf("%s extends_id = %#v, want %s", id, value, want)
		}
	}
}

func TestExtendTopicImmutableByteIdentical(t *testing.T) {
	store := newTestStore(t)
	createOpenTopic(t, store, extendA, "A")
	childPath := createOpenTopic(t, store, extendB, "B")
	createOpenTopic(t, store, extendC, "C")
	if _, err := store.ExtendTopic(domain.TopicID(extendB), extendA); err != nil {
		t.Fatalf("first extend: %v", err)
	}
	before := readMeta(t, childPath)
	if _, err := store.ExtendTopic(domain.TopicID(extendB), extendC); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second extend error = %v, want ErrConflict", err)
	}
	after := readMeta(t, childPath)
	if string(before) != string(after) {
		t.Fatal("_meta.yaml changed after rejected double extend")
	}
}

func TestExtendTopicFailuresLeaveMetadataUntouched(t *testing.T) {
	tests := []struct {
		name    string
		parent  string
		wantErr error
	}{
		{name: "self-extend", parent: extendB, wantErr: domain.ErrConflict},
		{name: "parent-missing", parent: "1799999999999", wantErr: domain.ErrTopicMissing},
		{name: "parent-invalid", parent: "not-a-topic-id", wantErr: domain.ErrInvalidID},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t)
			childPath := createOpenTopic(t, store, extendB, "B")
			before := readMeta(t, childPath)
			if _, err := store.ExtendTopic(domain.TopicID(extendB), tc.parent); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			after := readMeta(t, childPath)
			if string(before) != string(after) {
				t.Fatal("_meta.yaml changed after rejected extend")
			}
		})
	}
}

func TestExtendTopicTargetMissing(t *testing.T) {
	store := newTestStore(t)
	createOpenTopic(t, store, extendA, "A")
	if _, err := store.ExtendTopic(domain.TopicID(extendB), extendA); !errors.Is(err, domain.ErrTopicMissing) {
		t.Fatalf("error = %v, want ErrTopicMissing", err)
	}
}

func TestExtendTopicDoesNotModifyParent(t *testing.T) {
	store := newTestStore(t)
	parentPath := createOpenTopic(t, store, extendA, "Parent")
	createOpenTopic(t, store, extendB, "Child")
	before := readMeta(t, parentPath)
	if _, err := store.ExtendTopic(domain.TopicID(extendB), extendA); err != nil {
		t.Fatalf("ExtendTopic: %v", err)
	}
	after := readMeta(t, parentPath)
	if string(before) != string(after) {
		t.Fatal("parent metadata modified by extend")
	}
}
