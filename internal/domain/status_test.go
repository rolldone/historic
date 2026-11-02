package domain

import "testing"

func TestAggregateStatusTable(t *testing.T) {
	cases := []struct {
		name     string
		statuses []Status
		want     Status
	}{
		{"three complete", []Status{StatusComplete, StatusComplete, StatusComplete}, StatusComplete},
		{"complete and cancelled", []Status{StatusComplete, StatusCancelled}, StatusComplete},
		{"all cancelled", []Status{StatusCancelled, StatusCancelled}, StatusCancelled},
		{"complete and forwarded", []Status{StatusComplete, StatusForwarded}, StatusComplete},
		{"all forwarded", []Status{StatusForwarded, StatusForwarded}, StatusComplete},
		{"cancelled and forwarded", []Status{StatusCancelled, StatusForwarded}, StatusComplete},
		{"archived and cancelled", []Status{StatusArchived, StatusCancelled}, StatusComplete},
		{"complete and failed", []Status{StatusComplete, StatusFailed}, StatusFailed},
		{"cancelled and failed", []Status{StatusCancelled, StatusFailed}, StatusFailed},
		{"blocked and complete", []Status{StatusBlocked, StatusComplete}, StatusBlocked},
		{"progress and forwarded", []Status{StatusProgress, StatusForwarded}, StatusProgress},
		{"draft and cancelled", []Status{StatusDraft, StatusCancelled}, StatusDraft},
		{"review and blocked", []Status{StatusReview, StatusBlocked}, StatusBlocked},
		{"pending and progress", []Status{StatusPending, StatusProgress}, StatusProgress},
		{"asset only (empty input)", nil, ""},
		{"topic without files (empty input)", []Status{}, ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := AggregateStatus(testCase.statuses); got != testCase.want {
				t.Fatalf("AggregateStatus(%v) = %q, want %q", testCase.statuses, got, testCase.want)
			}
		})
	}
}

func TestAggregateStatusOpenPriority(t *testing.T) {
	ordered := []Status{
		StatusBlocked,
		StatusReview,
		StatusProgress,
		StatusPlanned,
		StatusPending,
		StatusDraft,
		StatusCreate,
	}
	for index := 0; index < len(ordered)-1; index++ {
		got := AggregateStatus([]Status{ordered[index+1], ordered[index]})
		if got != ordered[index] {
			t.Fatalf("AggregateStatus(%q, %q) = %q, want %q", ordered[index+1], ordered[index], got, ordered[index])
		}
	}
}

func TestAggregateStatusIgnoresInputOrder(t *testing.T) {
	sets := [][]Status{
		{StatusComplete, StatusCancelled, StatusFailed},
		{StatusProgress, StatusBlocked, StatusReview, StatusComplete},
		{StatusForwarded, StatusArchived, StatusCancelled},
		{StatusDraft, StatusPending, StatusPlanned},
	}
	for _, set := range sets {
		want := AggregateStatus(set)
		permutations := permuteStatuses(set)
		for _, permutation := range permutations {
			if got := AggregateStatus(permutation); got != want {
				t.Fatalf("AggregateStatus(%v) = %q, want %q", permutation, got, want)
			}
		}
	}
}

func TestAggregateStatusAccessorsAreSorted(t *testing.T) {
	for _, list := range [][]Status{OpenStatuses(), CloseStatuses(), ValidStatuses()} {
		for index := 1; index < len(list); index++ {
			if list[index-1] >= list[index] {
				t.Fatalf("status list not sorted: %v", list)
			}
		}
	}
	if len(OpenStatuses()) != len(openStatuses) || len(CloseStatuses()) != len(closeStatuses) {
		t.Fatal("status accessors do not match registered status sets")
	}
}

func permuteStatuses(values []Status) [][]Status {
	if len(values) <= 1 {
		return [][]Status{append([]Status(nil), values...)}
	}
	results := make([][]Status, 0)
	for index := range values {
		rest := make([]Status, 0, len(values)-1)
		rest = append(rest, values[:index]...)
		rest = append(rest, values[index+1:]...)
		for _, permutation := range permuteStatuses(rest) {
			results = append(results, append([]Status{values[index]}, permutation...))
		}
	}
	return results
}
