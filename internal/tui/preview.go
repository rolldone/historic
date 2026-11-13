package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/display"
	"historic/internal/domain"
	"historic/internal/search"

	"github.com/charmbracelet/lipgloss"
)

const previewLabelWidth = 7

const previewIndent = 2 + previewLabelWidth + 1

var (
	previewTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	previewMutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	previewWarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D7AF00"))
)

// previewLines builds the structured preview block shared by every terminal
// width. Lines are already truncated/wrapped for width so rendering only has
// to window them vertically.
func (m model) previewLines(width int) []string {
	page := m.currentPage()
	if len(page) == 0 || m.selected >= len(page) || width < 1 {
		return nil
	}
	result := page[m.selected]
	lines := make([]string, 0, 12)

	title := strings.TrimSpace(result.Title)
	if title == "" {
		title = filepath.Base(result.Path)
	}
	lines = append(lines, previewTitleStyle.Render(display.Truncate(title, width)))

	statusLine := fmt.Sprintf("[%s] · %s", strings.ToUpper(result.Storage), resultStatus(result))
	lines = append(lines, previewMutedStyle.Render(display.Truncate(statusLine, width)))
	if caption := domain.CopyRegionCaption(result.NewerCopy); caption != "" {
		lines = append(lines, previewWarningStyle.Render(display.Truncate(display.WarningIcon+" "+caption, width)))
	}

	lines = previewLabeled(lines, "time", previewTimeLine(result), width)
	lines = previewLabeled(lines, "path", display.TruncateMiddle(result.Path, width-previewIndent), width)

	if reason := strings.TrimSpace(result.Description); reason != "" {
		lines = append(lines, "")
		lines = previewLabeled(lines, "reason", reason, width)
	}
	lines = append(lines, previewMatchLines(result, strings.TrimSpace(m.input.Value()), width)...)
	if len(result.Tags) > 0 {
		lines = previewLabeled(lines, "tag", strings.Join(result.Tags, ", "), width)
	}
	if result.Extends != "" {
		lines = append(lines, "  extends → "+result.Extends)
	}
	if result.ForwardedTo != "" {
		lines = append(lines, "  forwarded → "+result.ForwardedTo)
	}
	return lines
}

func resultStatus(result search.Result) string {
	if result.Status == "" {
		return "unknown"
	}
	return result.Status
}

func previewTimeLine(result search.Result) string {
	parts := make([]string, 0, 2)
	if date := display.FormatUpdatedStamp(result.UpdatedAt); date != "" {
		parts = append(parts, "updated "+date)
	} else if date := display.FormatUpdatedStamp(result.CreatedAt); date != "" {
		parts = append(parts, "updated "+date)
	}
	if mtime := previewShortTime(result.Mtime); mtime != "" {
		parts = append(parts, "mtime "+mtime)
	}
	return strings.Join(parts, " · ")
}

func previewShortTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if timestamp, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return timestamp.UTC().Format(time.RFC3339)
	}
	return value
}

func previewLabeled(lines []string, label, value string, width int) []string {
	if strings.TrimSpace(value) == "" {
		return lines
	}
	valueWidth := width - previewIndent
	if valueWidth < 1 {
		valueWidth = 1
	}
	wrapped := previewWrap(value, valueWidth)
	prefix := "  " + fmt.Sprintf("%-*s", previewLabelWidth, label) + " "
	indent := strings.Repeat(" ", previewIndent)
	for index, line := range wrapped {
		if index == 0 {
			lines = append(lines, prefix+line)
			continue
		}
		lines = append(lines, indent+line)
	}
	return lines
}

// previewWrap preserves explicit line breaks while wrapping long lines.
func previewWrap(value string, width int) []string {
	lines := make([]string, 0, 4)
	for _, paragraph := range strings.Split(value, "\n") {
		paragraph = strings.TrimRight(paragraph, "\r")
		if strings.TrimSpace(paragraph) == "" {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			continue
		}
		lines = append(lines, display.WrapWords(paragraph, width)...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func previewMatchLines(result search.Result, keyword string, width int) []string {
	if keyword == "" || strings.TrimSpace(result.Snippet) == "" {
		return nil
	}
	available := width - previewIndent
	if available < 1 {
		available = 1
	}
	snippet := "▸" + result.Snippet + "◂"
	wrapped := previewWrap(snippet, available)
	indent := strings.Repeat(" ", previewIndent)
	lines := []string{""}
	if result.MatchCount > 1 {
		meta := fmt.Sprintf("%d lokasi", result.MatchCount)
		if result.MatchLine > 0 {
			meta += fmt.Sprintf(" · terbaik di sekitar baris %d", result.MatchLine)
		}
		lines = previewLabeled(lines, "match", meta, width)
		for _, line := range wrapped {
			lines = append(lines, indent+search.HighlightHuman(line, keyword))
		}
		return lines
	}
	prefixSnippet := "  " + fmt.Sprintf("%-*s", previewLabelWidth, "match") + " "
	for index, line := range wrapped {
		highlighted := search.HighlightHuman(line, keyword)
		if index == 0 {
			lines = append(lines, prefixSnippet+highlighted)
			continue
		}
		lines = append(lines, indent+highlighted)
	}
	return lines
}

// visiblePreview slices the preview window for the given viewport height,
// clamping the scroll offset so the last line can always be reached.
func (m *model) visiblePreview(width, height int) []string {
	lines := m.previewLines(width)
	if len(lines) == 0 || height < 1 {
		return nil
	}
	offset := m.previewOffset
	maxOffset := len(lines) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}
	return lines[offset:end]
}
