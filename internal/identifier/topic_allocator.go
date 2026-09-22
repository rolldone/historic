package identifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"historic/internal/domain"
)

var ErrLockHeld = errors.New("topic allocator lock is held")

// Lock serializes operations which allocate topic identities.
type Lock interface {
	Acquire(context.Context) error
	Release() error
}

type lockRecord struct {
	PID       int    `json:"pid"`
	Timestamp string `json:"timestamp"`
}

type FileLock struct {
	path string
	file *os.File
}

func NewFileLock(path string) Lock { return &FileLock{path: path} }

func (lock *FileLock) Acquire(ctx context.Context) error {
	if lock.file != nil {
		return nil
	}
	for {
		file, err := os.OpenFile(lock.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			record := lockRecord{PID: os.Getpid(), Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
			if encodeErr := json.NewEncoder(file).Encode(record); encodeErr != nil {
				_ = file.Close()
				_ = os.Remove(lock.path)
				return fmt.Errorf("write topic lock: %w", encodeErr)
			}
			if err := file.Sync(); err != nil {
				_ = file.Close()
				_ = os.Remove(lock.path)
				return fmt.Errorf("sync topic lock: %w", err)
			}
			lock.file = file
			return nil
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create topic lock: %w", err)
		}
		if stale, inspectErr := staleLock(lock.path); inspectErr == nil && stale {
			// Never remove a lock whose owner process is alive. A malformed lock
			// is intentionally treated as live to avoid unsafe recovery.
			if removeErr := os.Remove(lock.path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				return fmt.Errorf("remove stale topic lock: %w", removeErr)
			}
			continue
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", ErrLockHeld, ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func (lock *FileLock) Release() error {
	if lock.file == nil {
		return nil
	}
	closeErr := lock.file.Close()
	removeErr := os.Remove(lock.path)
	lock.file = nil
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}
	return nil
}

func staleLock(path string) (bool, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var record lockRecord
	if err := json.Unmarshal(input, &record); err != nil || record.PID <= 0 {
		return false, errors.New("invalid lock record")
	}
	if err := syscall.Kill(record.PID, 0); err == nil || errors.Is(err, syscall.EPERM) {
		return false, nil
	}
	return errors.Is(err, syscall.ESRCH), nil
}

// MillisecondClock supplies time to the topic allocator.
type MillisecondClock interface{ Now() time.Time }

type TopicIDAllocator interface {
	Allocate(context.Context) (domain.TopicID, error)
}

type topicIDAllocator struct {
	root  string
	clock MillisecondClock
	lock  Lock
}

func NewTopicIDAllocator(root string, clock MillisecondClock, lock Lock) TopicIDAllocator {
	if clock == nil {
		clock = allocClock{}
	}
	if lock == nil {
		lock = NewFileLock(filepath.Join(root, ".topic-id.lock"))
	}
	return &topicIDAllocator{root: root, clock: clock, lock: lock}
}

func (allocator *topicIDAllocator) Allocate(ctx context.Context) (domain.TopicID, error) {
	if err := allocator.lock.Acquire(ctx); err != nil {
		return "", err
	}
	defer allocator.lock.Release()
	return allocator.allocateUnlocked()
}

func (allocator *topicIDAllocator) allocateUnlocked() (domain.TopicID, error) {
	maxID := int64(0)
	lastPath := filepath.Join(allocator.root, ".topic-id.last")
	if input, err := os.ReadFile(lastPath); err == nil {
		if stored, parseErr := strconv.ParseInt(string(input), 10, 64); parseErr == nil && stored > maxID {
			maxID = stored
		}
	}
	for _, root := range []string{allocator.root, filepath.Join(allocator.root, ".database")} {
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("scan topics: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if id, ok := domain.TopicIDFromFolder(entry.Name()); ok {
				n, parseErr := strconv.ParseInt(id.String(), 10, 64)
				if parseErr == nil && n > maxID {
					maxID = n
				}
			}
		}
	}
	now := allocator.clock.Now().UTC().UnixMilli()
	if now <= 0 {
		now = 1
	}
	if now <= maxID {
		now = maxID + 1
	}
	return domain.ParseTopicID(strconv.FormatInt(now, 10))
}

// AllocateUnlocked is used by repository transactions that already hold the
// workspace lock. It deliberately performs a fresh filesystem scan.
func AllocateTopicIDUnlocked(root string, clock MillisecondClock) (domain.TopicID, error) {
	if clock == nil {
		clock = allocClock{}
	}
	return (&topicIDAllocator{root: root, clock: clock}).allocateUnlocked()
}

type allocClock struct{}

func (allocClock) Now() time.Time { return time.Now() }
