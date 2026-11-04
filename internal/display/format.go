// Package display contains terminal presentation helpers shared by CLI
// commands. It is deliberately independent of any specific command so the
// same wrapping, truncation, and width rules can be reused by the TUI.
package display

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

const (
	// DefaultWidth is the fallback terminal width when neither the real
	// terminal size nor COLUMNS is available.
	DefaultWidth = 80
	// MinWidth keeps very narrow terminals usable without negative widths.
	MinWidth = 24
)

// TerminalWidth resolves the usable output width. It prefers the real stdout
// size, then COLUMNS, and finally DefaultWidth.
func TerminalWidth() int {
	if width, _, err := term.GetSize(os.Stdout.Fd()); err == nil && width > 0 {
		return clampWidth(width)
	}
	if raw := strings.TrimSpace(os.Getenv("COLUMNS")); raw != "" {
		if width, err := strconv.Atoi(raw); err == nil && width > 0 {
			return clampWidth(width)
		}
	}
	return DefaultWidth
}

func clampWidth(width int) int {
	if width < MinWidth {
		return MinWidth
	}
	return width
}

// Width returns the printable cell width of value, ignoring ANSI escapes.
func Width(value string) int {
	return ansi.StringWidth(value)
}

// Truncate shortens value to at most width cells, appending an ellipsis when
// truncation happens. ANSI escapes are preserved.
func Truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if Width(value) <= width {
		return value
	}
	if width <= 3 {
		return ansi.Truncate(value, width, "")
	}
	return ansi.Truncate(value, width, "...")
}

// TruncateMiddle shortens a path-like value in the middle so the leading
// workspace prefix and the file name remain readable: a/very/.../file.md.
func TruncateMiddle(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if Width(value) <= width {
		return value
	}
	if width <= 3 {
		return ansi.Truncate(value, width, "")
	}
	runes := []rune(value)
	keep := width - 3
	head := keep / 2
	tail := keep - head
	if head < 0 || tail < 0 || head+tail > len(runes) {
		return ansi.Truncate(value, width, "")
	}
	return string(runes[:head]) + "..." + string(runes[len(runes)-tail:])
}

// WrapWords wraps plain text on word boundaries to fit width cells. Words
// longer than width are hard-split so output never exceeds the viewport.
func WrapWords(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if width < 1 {
		width = 1
	}
	words := strings.Fields(text)
	lines := make([]string, 0, len(words))
	current := ""
	for _, word := range words {
		if Width(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, hardSplit(word, width)...)
			continue
		}
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if Width(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func hardSplit(word string, width int) []string {
	runes := []rune(word)
	lines := make([]string, 0, len(runes)/width+1)
	for len(runes) > 0 {
		take := width
		if take > len(runes) {
			take = len(runes)
		}
		lines = append(lines, string(runes[:take]))
		runes = runes[take:]
	}
	return lines
}
