package display

import "testing"

func TestTruncateMiddlePreservesBothEnds(t *testing.T) {
	got := TruncateMiddle("abcdefghij", 7)
	if got != "ab...ij" {
		t.Fatalf("TruncateMiddle = %q, want %q", got, "ab...ij")
	}
	if got := TruncateMiddle("short", 10); got != "short" {
		t.Fatalf("TruncateMiddle short = %q", got)
	}
}

func TestWrapWordsNeverExceedsWidth(t *testing.T) {
	lines := WrapWords("alpha beta gamma delta", 10)
	for _, line := range lines {
		if Width(line) > 10 {
			t.Fatalf("line %q exceeds width: %d", line, Width(line))
		}
	}
	if len(lines) < 2 {
		t.Fatalf("expected wrapping, got %#v", lines)
	}
}
