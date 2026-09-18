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
	for _, status := range []Status{StatusCreate, StatusPending, StatusProgress, StatusReview, StatusBlocked} {
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
	valid := []Status{StatusCreate, StatusPending, StatusProgress, StatusReview, StatusBlocked, StatusComplete, StatusFailed, StatusCancelled, StatusArchived}
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
