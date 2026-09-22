package identifier

import (
	"bytes"
	"crypto/rand"
	"sync"
	"testing"
	"time"
)

type fixedClock struct{ value time.Time }

func (clock fixedClock) Now() time.Time { return clock.value }

func TestUUIDv7GeneratorFormatAndConcurrency(t *testing.T) {
	generator := NewUUIDv7Generator(fixedClock{value: time.UnixMilli(1_700_000_000_000)}, rand.Reader)
	const workers = 100
	const perWorker = 100
	ids := make(chan string, workers*perWorker)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := 0; index < perWorker; index++ {
				id, err := generator.New()
				if err != nil {
					t.Errorf("New: %v", err)
					return
				}
				ids <- id.String()
			}
		}()
	}
	group.Wait()
	close(ids)
	seen := make(map[string]struct{}, workers*perWorker)
	for id := range ids {
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("duplicate ID %q", id)
		}
		seen[id] = struct{}{}
		if len(id) != 36 || id[14] != '7' || (id[19] != '8' && id[19] != '9' && id[19] != 'a' && id[19] != 'b') {
			t.Fatalf("invalid UUIDv7 %q", id)
		}
	}
	if len(seen) != workers*perWorker {
		t.Fatalf("generated %d IDs, want %d", len(seen), workers*perWorker)
	}
}

func TestUUIDv7GeneratorRejectsCollision(t *testing.T) {
	generator := NewUUIDv7Generator(fixedClock{value: time.UnixMilli(1)}, bytes.NewReader(bytes.Repeat([]byte{0}, 40)))
	if _, err := generator.New(); err != nil {
		t.Fatal(err)
	}
	if _, err := generator.New(); err != ErrCollision {
		t.Fatalf("collision error = %v, want %v", err, ErrCollision)
	}
}
