package identifier

import (
	"encoding/hex"
	"strconv"
	"time"

	"historic/internal/domain"
)

// TopicIDTime recovers the creation instant encoded in a modern epoch-millisecond
// topic identity. Legacy five-digit identities carry no timestamp, so they report
// ok=false and callers must fall back to a calendar date.
func TopicIDTime(id domain.ID) (time.Time, bool) {
	value := id.String()
	if len(value) != 13 {
		return time.Time{}, false
	}
	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil || millis <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(millis).UTC(), true
}

// FileIDTime recovers the creation instant encoded in the first 48 bits of a
// UUIDv7 managed-file identity. Non-UUIDv7 identities report ok=false.
func FileIDTime(id domain.FileID) (time.Time, bool) {
	if !id.Valid() {
		return time.Time{}, false
	}
	compact := make([]byte, 0, 12)
	for _, r := range id.String() {
		if r == '-' {
			continue
		}
		compact = append(compact, byte(r))
	}
	head := make([]byte, 6)
	if _, err := hex.Decode(head, compact[:12]); err != nil {
		return time.Time{}, false
	}
	var millis int64
	for _, b := range head {
		millis = millis<<8 | int64(b)
	}
	if millis <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(millis).UTC(), true
}
