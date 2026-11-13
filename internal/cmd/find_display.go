package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/display"
	"historic/internal/search"
)

const findLabelWidth = 7

// writeFindHuman renders one search page as labeled, multi-line entries. The
// JSON contract is untouched; this function only owns human presentation.
func writeFindHuman(out io.Writer, page search.SearchPage, keyword string, width int) error {
	if width <= 0 {
		width = display.DefaultWidth
	}
	// Keep one spare cell so terminals with autowrap never fold the last
	// character of an exactly-full line onto the next visual row.
	if width > 2 {
		width--
	}
	results := page.Items
	if len(results) == 0 {
		_, err := fmt.Fprintln(out, "No matches found.")
		return err
	}
	blocks := make([][]string, 0, len(results))
	for _, result := range results {
		blocks = append(blocks, renderFindResult(result, keyword, width))
	}
	for index, block := range blocks {
		if index > 0 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
		for _, line := range block {
			if _, err := fmt.Fprintln(out, line); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	pagination := page.Pagination
	if pagination.HasMore {
		_, err := fmt.Fprintf(out, "Page %d · showing %d results · more results available\nUse --page %d --page-size %d\n", pagination.Page, len(results), pagination.Page+1, pagination.PageSize)
		return err
	}
	_, err := fmt.Fprintf(out, "Page %d · showing %d results · no more results\n", pagination.Page, len(results))
	return err
}

func renderFindResult(result search.Result, keyword string, width int) []string {
	lines := make([]string, 0, 8)
	lines = append(lines, findHeader(result, width))

	lines = append(lines, findLabeled("path", []string{findPath(result.Path, width)}, width)...)
	if timeLine := findTimeLine(result); timeLine != "" {
		lines = append(lines, findLabeled("time", []string{timeLine}, width)...)
	}
	if reason := strings.TrimSpace(result.Description); reason != "" {
		lines = append(lines, findLabeled("reason", display.WrapWords(reason, findValueWidth(width)), width)...)
	}
	lines = append(lines, findMatchLines(result, keyword, width)...)
	if len(result.Tags) > 0 {
		lines = append(lines, findLabeled("tag", []string{strings.Join(result.Tags, ", ")}, width)...)
	}
	if result.Extends != "" {
		lines = append(lines, "  extends → "+result.Extends)
	}
	if result.ForwardedTo != "" {
		lines = append(lines, "  forwarded → "+result.ForwardedTo)
	}
	return lines
}

func findHeader(result search.Result, width int) string {
	title := strings.TrimSpace(result.Title)
	if title == "" {
		title = filepath.Base(result.Path)
	}
	right := fmt.Sprintf("[%s] · %s", strings.ToUpper(result.Storage), result.Status)
	right = display.Truncate(right, width)
	left := result.TopicID + " · " + title
	available := width - display.Width(right) - 1
	if available < 1 {
		return right
	}
	left = display.Truncate(left, available)
	gap := width - display.Width(left) - display.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func findPath(path string, width int) string {
	return display.TruncateMiddle(strings.TrimSpace(path), findValueWidth(width))
}

func findTimeLine(result search.Result) string {
	date := display.FormatUpdatedStamp(result.UpdatedAt)
	if date == "" {
		date = display.FormatUpdatedStamp(result.CreatedAt)
	}
	parts := make([]string, 0, 2)
	if date != "" {
		parts = append(parts, "updated "+date)
	}
	if mtime := shortMtime(result.Mtime); mtime != "" {
		parts = append(parts, "mtime "+mtime)
	}
	return strings.Join(parts, " · ")
}

func shortMtime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if timestamp, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return timestamp.UTC().Format(time.RFC3339)
	}
	return value
}

func findValueWidth(width int) int {
	valueWidth := width - findValueIndent()
	if valueWidth < 1 {
		return 1
	}
	return valueWidth
}

func findValueIndent() int {
	return 2 + findLabelWidth + 1
}

func findLabelPrefix(label string) string {
	return "  " + fmt.Sprintf("%-*s", findLabelWidth, label) + " "
}

// findLabeled renders already-wrapped values under a label with a hanging
// indent, so continuation lines stay aligned with the first value.
func findLabeled(label string, valueLines []string, width int) []string {
	if len(valueLines) == 0 {
		return nil
	}
	prefix := findLabelPrefix(label)
	indent := strings.Repeat(" ", findValueIndent())
	lines := make([]string, 0, len(valueLines))
	for index, value := range valueLines {
		value = display.Truncate(value, findValueWidth(width))
		if index == 0 {
			lines = append(lines, prefix+value)
			continue
		}
		lines = append(lines, indent+value)
	}
	return lines
}

func findMatchLines(result search.Result, keyword string, width int) []string {
	if strings.TrimSpace(keyword) == "" {
		return nil
	}
	snippet := strings.TrimSpace(result.Snippet)
	if snippet == "" {
		return nil
	}
	decorated := "▸" + snippet + "◂"
	available := findValueWidth(width)
	prefix := findLabelPrefix("match")
	indent := strings.Repeat(" ", findValueIndent())
	wrapped := highlightWrapped(decorated, keyword, available)
	if result.MatchCount > 1 {
		meta := fmt.Sprintf("%d lokasi", result.MatchCount)
		if result.MatchLine > 0 {
			meta += fmt.Sprintf(" · terbaik di sekitar baris %d", result.MatchLine)
		}
		lines := []string{prefix + display.Truncate(meta, available)}
		for _, line := range wrapped {
			lines = append(lines, indent+line)
		}
		return lines
	}
	lines := make([]string, 0, len(wrapped))
	for index, line := range wrapped {
		if index == 0 {
			lines = append(lines, prefix+line)
			continue
		}
		lines = append(lines, indent+line)
	}
	return lines
}

func highlightWrapped(value, keyword string, width int) []string {
	wrapped := display.WrapWords(value, width)
	for index, line := range wrapped {
		wrapped[index] = search.HighlightHuman(line, keyword)
	}
	return wrapped
}
