package markdown

import (
	"testing"
	"time"
)

func TestFormatTimestampTruncatesToUTCSeconds(t *testing.T) {
	// 10:45:30 WIB is 03:45:30 UTC; sub-second precision must be dropped.
	value := time.Date(2026, 11, 13, 10, 45, 30, 987654321, time.FixedZone("WIB", 7*3600))
	if got := FormatTimestamp(value); got != "2026-11-13T03:45:30Z" {
		t.Fatalf("FormatTimestamp = %q", got)
	}
}

func TestParseTimestampAcceptsCanonicalStampsOnly(t *testing.T) {
	valid := []string{"2026-11-13T10:45:30Z", "1970-01-01T00:00:00Z"}
	for _, value := range valid {
		parsed, err := ParseTimestamp(value)
		if err != nil {
			t.Fatalf("ParseTimestamp(%q) = %v", value, err)
		}
		if FormatTimestamp(parsed) != value {
			t.Fatalf("round trip mismatch for %q", value)
		}
	}

	invalid := []string{
		"2026-11-13",                // date only, no time
		"2026-11-13T10:45:30",       // missing zone
		"2026-11-13T10:45:30+07:00", // not UTC
		"2026-11-13T10:45:30.5Z",    // sub-second precision
		"2026-11-13 10:45:30Z",      // space instead of T
		"not-a-timestamp",
	}
	for _, value := range invalid {
		if _, err := ParseTimestamp(value); err == nil {
			t.Fatalf("ParseTimestamp(%q) accepted an invalid stamp", value)
		}
	}
}

func TestValidTimestampAllowsEmptyForLegacyTopics(t *testing.T) {
	if !ValidTimestamp("") {
		t.Fatal("empty stamp must stay valid for legacy topics")
	}
	if !ValidTimestamp("2026-11-13T10:45:30Z") {
		t.Fatal("canonical stamp must be valid")
	}
	if ValidTimestamp("2026-11-13T10:45:30") {
		t.Fatal("zone-less stamp must be rejected")
	}
}
