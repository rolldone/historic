package markdown

import (
	"fmt"
	"time"
)

// TimestampLayout is the canonical high-resolution freshness stamp: RFC3339 in
// UTC with second precision (for example 2026-11-13T10:45:30Z). It is stored in
// `_meta.yaml` both at the topic level and per managed file entry so ordering by
// "when the file was last edited" survives a pull, unlike a local-only index.
const TimestampLayout = "2006-01-02T15:04:05Z"

// FormatTimestamp renders t as the canonical RFC3339 UTC second-precision stamp.
func FormatTimestamp(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format(TimestampLayout)
}

// ParseTimestamp validates a canonical freshness stamp. Only second-precision
// UTC timestamps are accepted; anything else is reported as invalid so a hand
// edited value cannot silently corrupt ordering.
func ParseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(TimestampLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("must be an RFC3339 UTC timestamp such as %s", time.Now().UTC().Format(TimestampLayout))
	}
	if FormatTimestamp(parsed) != value {
		return time.Time{}, fmt.Errorf("must be UTC with second precision such as %s", time.Now().UTC().Format(TimestampLayout))
	}
	return parsed, nil
}

// ValidTimestamp reports whether value is empty (legacy topics without a stamp
// remain valid) or a canonical RFC3339 UTC second-precision stamp.
func ValidTimestamp(value string) bool {
	if value == "" {
		return true
	}
	_, err := ParseTimestamp(value)
	return err == nil
}
