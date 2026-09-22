package identifier

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"historic/internal/domain"
)

var ErrCollision = errors.New("generated file ID collision")

// Clock supplies time to the UUIDv7 generator.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type uuidv7Generator struct {
	mu      sync.Mutex
	clock   Clock
	entropy io.Reader
	lastMS  int64
	seen    map[domain.FileID]struct{}
}

// Generator creates managed-file identifiers.
type Generator interface{ New() (domain.FileID, error) }

// NewUUIDv7Generator creates a concurrency-safe UUIDv7 generator. A nil clock
// or entropy reader uses the system clock or crypto/rand.Reader respectively.
func NewUUIDv7Generator(clock Clock, entropy io.Reader) Generator {
	if clock == nil {
		clock = systemClock{}
	}
	if entropy == nil {
		entropy = rand.Reader
	}
	return &uuidv7Generator{clock: clock, entropy: entropy, seen: make(map[domain.FileID]struct{})}
}

func (generator *uuidv7Generator) New() (domain.FileID, error) {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	for attempt := 0; attempt < 3; attempt++ {
		now := generator.clock.Now().UTC().UnixMilli()
		if now < generator.lastMS {
			now = generator.lastMS
		}
		var random [10]byte
		if _, err := io.ReadFull(generator.entropy, random[:]); err != nil {
			return "", fmt.Errorf("read UUIDv7 entropy: %w", err)
		}
		var raw [16]byte
		for shift := uint(0); shift < 6; shift++ {
			raw[5-shift] = byte(uint64(now) >> (shift * 8))
		}
		copy(raw[6:], random[:])
		raw[6] = (raw[6] & 0x0f) | 0x70
		raw[8] = (raw[8] & 0x3f) | 0x80
		value := domain.FileID(fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(raw[0:4]), hex.EncodeToString(raw[4:6]), hex.EncodeToString(raw[6:8]), hex.EncodeToString(raw[8:10]), hex.EncodeToString(raw[10:16])))
		if !value.Valid() {
			return "", fmt.Errorf("generated invalid UUIDv7 %q", value)
		}
		generator.lastMS = now
		if _, exists := generator.seen[value]; exists {
			continue
		}
		generator.seen[value] = struct{}{}
		return value, nil
	}
	return "", ErrCollision
}

// Default is the process-wide generator used by commands that create files.
var Default Generator = NewUUIDv7Generator(nil, nil)
