package display

import "testing"

func TestCollapseIdenticalCopiesKeepsTheOpenRowForEqualStamps(t *testing.T) {
	rows := []CopyRow{
		{Key: "topic-1", Stamp: "2026-11-13T16:02:11Z", Open: false},
		{Key: "topic-1", Stamp: "2026-11-13T16:02:11Z", Open: true},
	}
	collapsed := CollapseIdenticalCopies(rows)
	if len(collapsed) != 1 {
		t.Fatalf("collapsed = %#v, want a single row", collapsed)
	}
	if collapsed[0].Index != 1 || !collapsed[0].Mirrored {
		t.Fatalf("collapsed row = %#v, want the open row flagged mirrored", collapsed[0])
	}
}

func TestCollapseIdenticalCopiesKeepsBothRowsWhenStampsDiffer(t *testing.T) {
	rows := []CopyRow{
		{Key: "topic-1", Stamp: "2026-11-13T16:02:11Z", Open: false},
		{Key: "topic-1", Stamp: "2026-11-13T10:45:30Z", Open: true},
	}
	collapsed := CollapseIdenticalCopies(rows)
	if len(collapsed) != 2 {
		t.Fatalf("collapsed = %#v, want both rows preserved", collapsed)
	}
	for _, row := range collapsed {
		if row.Mirrored {
			t.Fatalf("diverged rows must not be flagged mirrored: %#v", row)
		}
	}
	if collapsed[0].Index != 0 || collapsed[1].Index != 1 {
		t.Fatalf("collapsed order = %#v, want input order", collapsed)
	}
}

func TestCollapseIdenticalCopiesCollapsesEqualEmptyStamps(t *testing.T) {
	// Pre-v9 rows carry no stamp: both sides fall back to nothing, which still
	// counts as identical.
	rows := []CopyRow{
		{Key: "topic-1", Stamp: "", Open: true},
		{Key: "topic-1", Stamp: "", Open: false},
	}
	collapsed := CollapseIdenticalCopies(rows)
	if len(collapsed) != 1 || collapsed[0].Index != 0 || !collapsed[0].Mirrored {
		t.Fatalf("collapsed = %#v, want the open row flagged mirrored", collapsed)
	}
}

func TestCollapseIdenticalCopiesLeavesSingleRowsUntouched(t *testing.T) {
	rows := []CopyRow{
		{Key: "topic-1", Stamp: "2026-11-13T16:02:11Z", Open: true},
		{Key: "topic-2", Stamp: "2026-11-13T16:02:11Z", Open: false},
	}
	collapsed := CollapseIdenticalCopies(rows)
	if len(collapsed) != 2 {
		t.Fatalf("collapsed = %#v, want both lone rows kept", collapsed)
	}
	for index, row := range collapsed {
		if row.Index != index || row.Mirrored {
			t.Fatalf("collapsed[%d] = %#v, want a plain pass-through", index, row)
		}
	}
}

func TestCollapseIdenticalCopiesFallsBackToTheFirstRowWithoutAnOpenCopy(t *testing.T) {
	rows := []CopyRow{
		{Key: "topic-1", Stamp: "2026-11-13T16:02:11Z", Open: false},
		{Key: "topic-1", Stamp: "2026-11-13T16:02:11Z", Open: false},
	}
	collapsed := CollapseIdenticalCopies(rows)
	if len(collapsed) != 1 || collapsed[0].Index != 0 || !collapsed[0].Mirrored {
		t.Fatalf("collapsed = %#v, want the first row as the survivor", collapsed)
	}
}
