package identifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
)

type allocatorClock struct{ now time.Time }

func (clock allocatorClock) Now() time.Time { return clock.now }

func TestTopicIDAllocatorAdvancesWithinSameMillisecond(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clock := allocatorClock{now: time.UnixMilli(1758537600123)}
	allocator := NewTopicIDAllocator(workspace.Histories, clock, NewFileLock(filepath.Join(workspace.Histories, ".topic-id.lock")))
	first, err := allocator.Allocate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first != "1758537600123" {
		t.Fatalf("first = %s", first)
	}
	if err := os.Mkdir(filepath.Join(workspace.Histories, first.String()+"-first"), 0o755); err != nil {
		t.Fatal(err)
	}
	second, err := allocator.Allocate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second != "1758537600124" {
		t.Fatalf("second = %s", second)
	}
}

func TestTopicIDAllocatorClampsClockRollback(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace.Histories, "1758537600123-existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	allocator := NewTopicIDAllocator(workspace.Histories, allocatorClock{now: time.UnixMilli(1758537600000)}, NewFileLock(filepath.Join(workspace.Histories, ".topic-id.lock")))
	id, err := allocator.Allocate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if id != "1758537600124" {
		t.Fatalf("rollback allocation = %s", id)
	}
}

func TestTopicIDAllocatorScansOpenAndClosedRoots(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace.Database, "1758537600124-closed"), 0o755); err != nil {
		t.Fatal(err)
	}
	allocator := NewTopicIDAllocator(workspace.Histories, allocatorClock{now: time.UnixMilli(1758537600123)}, NewFileLock(filepath.Join(workspace.Histories, ".topic-id.lock")))
	id, err := allocator.Allocate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if id != "1758537600125" {
		t.Fatalf("closed-root allocation = %s", id)
	}
}

func TestFileLockRelease(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace.Histories, ".topic-id.lock")
	lock := NewFileLock(path)
	if err := lock.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("lock remains: %v", err)
	}
}

func TestTopicIDAllocatorProducesValidDomainID(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	allocator := NewTopicIDAllocator(workspace.Histories, allocatorClock{now: time.UnixMilli(1758537600123)}, nil)
	id, err := allocator.Allocate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.ParseTopicID(id.String()); err != nil {
		t.Fatal(err)
	}
}
