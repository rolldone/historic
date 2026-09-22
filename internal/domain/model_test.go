package domain

import (
	"errors"
	"testing"
)

func TestParseIDPreservesPaddingAndNumber(t *testing.T) {
	id, err := ParseID("00014")
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	if id.String() != "00014" {
		t.Fatalf("ID string = %q, want 00014", id)
	}
	if id.Number() != 14 {
		t.Fatalf("ID number = %d, want 14", id.Number())
	}
}

func TestParseIDRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"14", "0001a", "000001", "-0001", ""} {
		if _, err := ParseID(value); !errors.Is(err, ErrInvalidID) {
			t.Errorf("ParseID(%q) error = %v, want ErrInvalidID", value, err)
		}
	}
}

func TestParseTopicIDAcceptsValidValues(t *testing.T) {
	valid := []string{
		"1758537600123",
		"1700000000000",
		"9999999999999",
	}
	for _, value := range valid {
		id, err := ParseTopicID(value)
		if err != nil {
			t.Errorf("ParseTopicID(%q) unexpected error: %v", value, err)
		}
		if id.String() != value {
			t.Errorf("ParseTopicID(%q).String() = %q", value, id.String())
		}
		if !id.Valid() {
			t.Errorf("ParseTopicID(%q).Valid() = false", value)
		}
	}
}

func TestParseTopicIDRejectsInvalidValues(t *testing.T) {
	invalid := []string{
		"",
		"14",
		"00001",
		"175853760012",
		"17585376001234",
		"175853760012a",
		"-1758537600123",
		"0000000000000",
		"01a0c8xx-xxxx-7xxx-xxxx-xxxxxxxxxxxx",
		"search-read-model",
	}
	for _, value := range invalid {
		_, err := ParseTopicID(value)
		if err == nil {
			t.Errorf("ParseTopicID(%q) should have returned an error", value)
		}
	}
}

func TestTopicIDValid(t *testing.T) {
	tests := []struct {
		id    TopicID
		valid bool
	}{
		{"1758537600123", true},
		{"9999999999999", true},
		{"00001", false},
		{"", false},
		{"175853760012a", false},
		{"0000000000000", false},
	}
	for _, tt := range tests {
		if got := tt.id.Valid(); got != tt.valid {
			t.Errorf("TopicID(%q).Valid() = %v, want %v", tt.id, got, tt.valid)
		}
	}
}

func TestTopicIDFromFolderExtractsModernID(t *testing.T) {
	id, ok := TopicIDFromFolder("1758537600123-admin-dashboard")
	if !ok || id != "1758537600123" {
		t.Errorf("TopicIDFromFolder modern: got %q, %v", id, ok)
	}
	id, ok = TopicIDFromFolder("00001-admin-dashboard")
	if ok {
		t.Errorf("TopicIDFromFolder legacy should return false, got %q, %v", id, ok)
	}
	id, ok = TopicIDFromFolder("not-a-folder")
	if ok {
		t.Errorf("TopicIDFromFolder invalid should return false, got %q, %v", id, ok)
	}
}

func TestParseTopicIdentityAcceptsBothFormats(t *testing.T) {
	id, err := ParseTopicIdentity("00001")
	if err != nil || id != "00001" {
		t.Errorf("ParseTopicIdentity(00001) = %q, %v", id, err)
	}
	id, err = ParseTopicIdentity("1758537600123")
	if err != nil || id != "1758537600123" {
		t.Errorf("ParseTopicIdentity(1758537600123) = %q, %v", id, err)
	}
	_, err = ParseTopicIdentity("not-valid")
	if err == nil {
		t.Errorf("ParseTopicIdentity(not-valid) should have returned an error")
	}
}

func TestIDValidAcceptsBothLegacyAndModern(t *testing.T) {
	tests := []struct {
		id    ID
		valid bool
	}{
		{"00001", true},
		{"1758537600123", true},
		{"14", false},
		{"", false},
		{"00001a", false},
	}
	for _, tt := range tests {
		if got := tt.id.Valid(); got != tt.valid {
			t.Errorf("ID(%q).Valid() = %v, want %v", tt.id, got, tt.valid)
		}
	}
}

func TestTopicFolderNameAcceptsModernID(t *testing.T) {
	got, err := TopicFolderName(ID("1758537600123"), "Admin Dashboard")
	if err != nil {
		t.Fatalf("TopicFolderName: %v", err)
	}
	if got != "1758537600123-admin-dashboard" {
		t.Fatalf("folder = %q, want 1758537600123-admin-dashboard", got)
	}
}

func TestParseFileIDAcceptsOnlyCanonicalUUIDv7(t *testing.T) {
	valid, err := ParseFileID("0192f3b5-1e20-7abc-8def-0123456789ab")
	if err != nil || !valid.Valid() {
		t.Fatalf("ParseFileID(valid) = %q, %v", valid, err)
	}
	for _, value := range []string{"0192f3b5-1e20-6abc-8def-0123456789ab", "0192f3b5-1e20-7abc-0def-0123456789ab", "0192F3B5-1e20-7abc-8def-0123456789ab", "not-a-uuid"} {
		if _, err := ParseFileID(value); !errors.Is(err, ErrInvalidFileID) {
			t.Errorf("ParseFileID(%q) error = %v, want ErrInvalidFileID", value, err)
		}
	}
}

func TestSlugTitle(t *testing.T) {
	tests := map[string]string{
		"Admin Dashboard":        "admin-dashboard",
		"  Resep__Nasi Goreng! ": "resep-nasi-goreng",
		"API v2 / OAuth":         "api-v2-oauth",
	}
	for title, want := range tests {
		got, err := SlugTitle(title)
		if err != nil {
			t.Fatalf("SlugTitle(%q): %v", title, err)
		}
		if got != want {
			t.Errorf("SlugTitle(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestTopicFolderName(t *testing.T) {
	got, err := TopicFolderName(ID("00014"), "Admin Dashboard")
	if err != nil {
		t.Fatalf("TopicFolderName: %v", err)
	}
	if got != "00014-admin-dashboard" {
		t.Fatalf("folder = %q, want 00014-admin-dashboard", got)
	}
}

func TestStatusParsingIsCaseSensitive(t *testing.T) {
	if _, err := ParseStatus("Progress"); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("ParseStatus accepted case-variant: %v", err)
	}
	status, err := ParseStatus("progress")
	if err != nil || status != StatusProgress {
		t.Fatalf("ParseStatus(progress) = %q, %v", status, err)
	}
}

func TestStatusOpenAndCloseSets(t *testing.T) {
	for _, status := range []Status{StatusCreate, StatusDraft, StatusPending, StatusPlanned, StatusProgress, StatusReview, StatusBlocked} {
		if !status.IsOpen() || status.IsClose() {
			t.Errorf("%q should be open only", status)
		}
	}
	for _, status := range []Status{StatusComplete, StatusFailed, StatusCancelled, StatusArchived} {
		if !status.IsClose() || status.IsOpen() {
			t.Errorf("%q should be close only", status)
		}
	}
}

func TestTransitionMatrix(t *testing.T) {
	valid := []Status{StatusCreate, StatusDraft, StatusPending, StatusPlanned, StatusProgress, StatusReview, StatusBlocked, StatusComplete, StatusFailed, StatusCancelled, StatusArchived}
	for _, current := range valid {
		for _, next := range valid {
			want := current == next || !current.IsClose()
			if got := CanTransition(current, next); got != want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", current, next, got, want)
			}
		}
	}
}

func TestValidateTransitionErrors(t *testing.T) {
	if err := ValidateTransition(StatusProgress, StatusComplete); err != nil {
		t.Fatalf("valid transition rejected: %v", err)
	}
	if err := ValidateTransition(StatusComplete, StatusProgress); !errors.Is(err, ErrConflict) {
		t.Fatalf("closed transition error = %v, want ErrConflict", err)
	}
	if err := ValidateTransition(Status("unknown"), StatusProgress); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("invalid current status error = %v, want ErrInvalidStatus", err)
	}
}
