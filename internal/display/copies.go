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
