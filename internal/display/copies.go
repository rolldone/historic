package display

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
)

// WarningIcon prefixes the two-copy notice in the TUI, where a coloured row is
// easy to miss while scrolling.
const WarningIcon = "⚠"

// NewerCopyNotice renders the bracketed freshness notice appended to the newer
// copy of a duplicated topic. The caption text is always present so no-TTY
// output (pipes, logs, tests) still carries the message; the warning colour is
// added only when w accepts ANSI styling.
func NewerCopyNotice(w io.Writer, caption string, icon bool) string {
	if caption == "" {
		return ""
	}
	label := caption
	if icon {
		label = WarningIcon + " " + caption
	}
	return "[" + styleWarning(w, label) + "]"
}

// ColorEnabled reports whether ANSI styling should be written to w. Non-file
// writers (buffers, pipes) and NO_COLOR environments stay unstyled.
func ColorEnabled(w io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(file.Fd())
}

// StyleWarning wraps text in the terminal warning colour when possible.
func StyleWarning(w io.Writer, text string) string {
	return styleWarning(w, text)
}

func styleWarning(w io.Writer, text string) string {
	if !ColorEnabled(w) {
		return text
	}
	return "\x1b[33m" + text + "\x1b[0m"
}

// Divider returns a thin horizontal rule spanning width columns. It is drawn
// above the two mirrored copies of one topic so the repeated row reads as an
// intentional pair instead of a rendering glitch.
func Divider(width int) string {
	if width <= 0 {
		width = TerminalWidth()
	}
	return strings.Repeat("─", width)
}

// CopyRow is one row of a listing surface whose entity may exist in both storage
// regions at once: the open workdir and the `.database` mirror. Listing surfaces
// build one row per storage copy, then use CollapseIdenticalCopies to decide
// which duplicates carry information.
type CopyRow struct {
	// Key is the identity the mirrored rows share. Rows with different keys are
	// never treated as copies of one another.
	Key string
	// Stamp is the effective freshness value compared across the copies of one
	// key: the high-resolution manifest stamp when present, otherwise the
	// calendar date fallback. Two copies share a stamp when that value is equal,
	// including when both are empty (pre-v9 rows).
	Stamp string
	// Open marks the row that lives in the open workdir — the copy that holds
	// the active work.
	Open bool
}

// CollapsedCopy points at one surviving row of the input slice.
type CollapsedCopy struct {
	// Index addresses the row in the slice handed to CollapseIdenticalCopies.
	Index int
	// Mirrored is true when this single row stands in for an identical mirrored
	// pair, so a JSON consumer can tell a second copy exists even though only
	// one row is listed.
	Mirrored bool
}

// CollapseIdenticalCopies drops duplicates that carry no information. Rows
// sharing a Key collapse to their open row when every copy carries the same
// Stamp — an identical mirror is not worth a second row. When the stamps differ
// every row survives, because then the duplicate is the signal that one side
// moved ahead. Lone rows and rows with a unique Key are returned untouched.
// Order follows the input slice.
func CollapseIdenticalCopies(rows []CopyRow) []CollapsedCopy {
	groups := make(map[string][]int, len(rows))
	order := make([]string, 0, len(rows))
	for index, row := range rows {
		if _, seen := groups[row.Key]; !seen {
			order = append(order, row.Key)
		}
		groups[row.Key] = append(groups[row.Key], index)
	}
	collapsed := make([]CollapsedCopy, 0, len(rows))
	for _, key := range order {
		group := groups[key]
		if len(group) < 2 || !sameCopyStamp(rows, group) {
			for _, index := range group {
				collapsed = append(collapsed, CollapsedCopy{Index: index})
			}
			continue
		}
		collapsed = append(collapsed, CollapsedCopy{Index: openCopyIndex(rows, group), Mirrored: true})
	}
	return collapsed
}

// sameCopyStamp reports whether every row of one group carries the same
// effective stamp.
func sameCopyStamp(rows []CopyRow, group []int) bool {
	for _, index := range group[1:] {
		if rows[index].Stamp != rows[group[0]].Stamp {
			return false
		}
	}
	return true
}

// openCopyIndex prefers the open row of a group, falling back to the first row
// when the group somehow holds no open copy.
func openCopyIndex(rows []CopyRow, group []int) int {
	for _, index := range group {
		if rows[index].Open {
			return index
		}
	}
	return group[0]
}
