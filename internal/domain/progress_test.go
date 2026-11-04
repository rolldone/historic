package domain

import "testing"

func TestProgressPercentTable(t *testing.T) {
	cases := []struct {
		name     string
		statuses []Status
		want     *int
	}{
		{"no files", nil, nil},
		{"empty list", []Status{}, nil},
		{"all closed", []Status{StatusComplete, StatusFailed, StatusCancelled}, intPointer(100)},
		{"all open", []Status{StatusProgress, StatusDraft, StatusBlocked}, intPointer(0)},
		{"one of three closed", []Status{StatusComplete, StatusProgress, StatusDraft}, intPointer(33)},
		{"two of three closed", []Status{StatusComplete, StatusArchived, StatusDraft}, intPointer(66)},
		{"forwarded counts as closed", []Status{StatusForwarded, StatusProgress}, intPointer(50)},
		{"archived counts as closed", []Status{StatusArchived, StatusReview, StatusBlocked, StatusCreate}, intPointer(25)},
		{"one of one closed", []Status{StatusComplete}, intPointer(100)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ProgressPercent(testCase.statuses)
			if testCase.want == nil {
				if got != nil {
					t.Fatalf("ProgressPercent(%v) = %d, want nil", testCase.statuses, *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ProgressPercent(%v) = nil, want %d", testCase.statuses, *testCase.want)
			}
			if *got != *testCase.want {
				t.Fatalf("ProgressPercent(%v) = %d, want %d", testCase.statuses, *got, *testCase.want)
			}
		})
	}
}

func TestProgressPercentFloorIsConservative(t *testing.T) {
	// 1 of 6 closed = 16.66 -> 16; 5 of 6 closed = 83.33 -> 83.
	if got := ProgressPercent([]Status{StatusComplete, StatusCreate, StatusDraft, StatusPending, StatusPlanned, StatusProgress}); got == nil || *got != 16 {
		t.Fatalf("1/6 percent = %v, want 16", got)
	}
	if got := ProgressPercent([]Status{StatusComplete, StatusComplete, StatusComplete, StatusComplete, StatusComplete, StatusCreate}); got == nil || *got != 83 {
		t.Fatalf("5/6 percent = %v, want 83", got)
	}
}

func TestProgressPercentNeverExceedsOneHundred(t *testing.T) {
	statuses := append([]Status{}, CloseStatuses()...)
	got := ProgressPercent(statuses)
	if got == nil || *got != 100 {
		t.Fatalf("all close statuses percent = %v, want 100", got)
	}
}

func intPointer(value int) *int { return &value }
