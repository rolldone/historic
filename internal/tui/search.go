package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/search"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const DefaultLimit = 20

type Options struct {
	Input  io.Reader
	Output io.Writer
}

type model struct {
	workspace config.Workspace
	input     textinput.Model
	results   []search.Result
	selected  int
	page      int
	limit     int
	status    domain.Status
	scope     string
	typeName  string
	focus     focusMode
	errorText string
	preview   string
	width     int
	height    int
}

type focusMode int

const (
	focusResults focusMode = iota
	focusQuery
	focusFilter
)

type refreshMsg struct{}
type refreshResultMsg struct {
	results []search.Result
	err     error
}

func Run(workspace config.Workspace, options Options) error {
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if !IsInteractive(options.Input, options.Output) {
		return fmt.Errorf("historic search requires an interactive terminal")
	}
	program := tea.NewProgram(newModel(workspace), tea.WithInput(options.Input), tea.WithOutput(options.Output), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(workspace config.Workspace) model {
	input := textinput.New()
	input.Prompt = "Query: "
	input.Placeholder = "type a query, or leave empty for recent topics"
	input.CharLimit = 200
	input.Width = 48
	return model{workspace: workspace, input: input, limit: DefaultLimit, scope: "active", focus: focusResults}
}

func (m model) Init() tea.Cmd { return m.refresh() }

func (m model) refresh() tea.Cmd {
	keyword := strings.TrimSpace(m.input.Value())
	options := search.Options{Keyword: keyword, ActiveOnly: m.scope == "active", ArchivedOnly: m.scope == "archived", Status: m.status}
	if m.typeName == "work-order" {
		options.Type = "task"
	}
	if m.typeName == "topic" {
		options.Type = "meta"
	}
	if keyword == "" {
		return func() tea.Msg {
			results, err := recentTopics(m.workspace, options)
			return refreshResultMsg{results: results, err: err}
		}
	}
	return func() tea.Msg {
		results, err := search.Find(m.workspace, options)
		return refreshResultMsg{results: results, err: err}
	}
}

func recentTopics(workspace config.Workspace, options search.Options) ([]search.Result, error) {
	return search.RecentTopics(workspace, options)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(message)
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case refreshResultMsg:
		m.results, m.errorText = message.results, ""
		if message.err != nil {
			m.errorText = message.err.Error()
		}
		m.page = 0
		m.selected = 0
		m.updatePreview()
	case refreshMsg:
		return m, m.refresh()
	}
	return m, nil
}

func (m model) handleKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusQuery {
		if message.Type == tea.KeyEsc {
			m.input.Blur()
			m.focus = focusResults
			return m, nil
		}
		var command tea.Cmd
		m.input, command = m.input.Update(message)
		if message.Type == tea.KeyEnter {
			m.input.Blur()
			m.focus = focusResults
			return m, m.refresh()
		}
		return m, command
	}
	if m.focus == focusFilter {
		switch message.String() {
		case "esc", "f", "enter":
			m.focus = focusResults
		case "s":
			m.cycleScope()
			return m, m.refresh()
		case "t":
			m.cycleType()
			return m, m.refresh()
		case "v":
			m.cycleStatus()
			return m, m.refresh()
		}
		return m, nil
	}
	if message.Type == tea.KeyCtrlC || message.String() == "q" {
		return m, tea.Quit
	}
	switch message.String() {
	case "/":
		m.focus = focusQuery
		return m, m.input.Focus()
	case "f":
		m.focus = focusFilter
		return m, nil
	case "r":
		return m, m.refresh()
	case "n":
		if m.page+1 < m.pageCount() {
			m.page++
			m.selected = 0
			m.updatePreview()
		}
	case "p":
		if m.page > 0 {
			m.page--
			m.selected = 0
			m.updatePreview()
		}
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.updatePreview()
		}
	case "down", "j":
		if m.selected+1 < len(m.currentPage()) {
			m.selected++
			m.updatePreview()
		}
	case "enter":
		m.updatePreview()
	case "esc":
		m.focus = focusResults
	}
	return m, nil
}

func (m *model) updatePreview() {
	page := m.currentPage()
	if len(page) == 0 || m.selected >= len(page) {
		m.preview = ""
		return
	}
	result := page[m.selected]
	m.preview = fmt.Sprintf("%s\nstatus: %s\npath: %s\n\n%s", result.Title, result.Status, result.Path, result.Snippet)
}

func (m model) currentPage() []search.Result {
	start := m.page * m.limit
	if start >= len(m.results) {
		return nil
	}
	end := start + m.limit
	if end > len(m.results) {
		end = len(m.results)
	}
	return m.results[start:end]
}

func (m model) pageCount() int {
	if len(m.results) == 0 {
		return 1
	}
	return (len(m.results) + m.limit - 1) / m.limit
}

func (m model) View() string {
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	width := m.width
	if width <= 0 {
		width = 120
	}
	height := m.height
	if height <= 0 {
		height = 30
	}
	headerLines := 2
	if m.focus == focusFilter || m.errorText != "" {
		headerLines++
	}
	footerLines := 2
	contentHeight := height - headerLines - footerLines
	if contentHeight < 3 {
		contentHeight = 3
	}
	header := accent.Render("Historic Search") + "\n" + truncate(fmt.Sprintf("Query: %s | status=%s scope=%s type=%s limit=%d | page %d/%d", m.input.View(), statusName(m.status), m.scope, m.typeNameOrAll(), m.limit, m.page+1, m.pageCount()), width)
	if m.focus == focusFilter {
		header += "\n" + muted.Render("Filter focus: s scope; v status; t type; Esc/f closes")
	}
	if m.errorText != "" {
		header += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F56")).Render(truncate(m.errorText, width))
	}
	leftWidth := width / 2
	if leftWidth < 24 {
		leftWidth = width - 2
	}
	rightWidth := width - leftWidth - 2
	if rightWidth < 24 {
		rightWidth = 24
	}
	rows := m.currentPage()
	visible := contentHeight - 2
	if visible < 1 {
		visible = 1
	}
	start := 0
	if m.selected >= visible {
		start = m.selected - visible + 1
	}
	if start+visible > len(rows) {
		start = len(rows) - visible
		if start < 0 {
			start = 0
		}
	}
	left := "Results\n"
	for index := start; index < len(rows) && index < start+visible; index++ {
		result := rows[index]
		marker := "  "
		if index == m.selected {
			marker = "> "
		}
		left += truncate(fmt.Sprintf("%s%s %s", marker, result.ID, result.Title), leftWidth) + "\n"
	}
	if len(rows) == 0 {
		left += muted.Render("No matches") + "\n"
	}
	right := "Preview\n" + truncateLines(m.preview, rightWidth, visible)
	var body string
	if width < 72 {
		body = truncateLines(left, width, contentHeight) + "\n" + truncateLines(right, width, contentHeight)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftWidth).Render(left), lipgloss.NewStyle().Width(rightWidth).Render(right))
	}
	footer := muted.Render(truncate("↑↓ navigate  Enter preview  / query  f filter(s scope/v status/t type)  n/p page  r refresh  q quit", width))
	return truncateLines(header, width, headerLines) + "\n" + truncateLines(body, width, contentHeight) + "\n" + footer
}

func truncate(value string, width int) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\n", " "), "\r", " ")
	if width < 1 {
		return ""
	}
	if len([]rune(value)) <= width {
		return value
	}
	runes := []rune(value)
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func truncateLines(value string, width, lines int) string {
	if lines < 1 {
		return ""
	}
	parts := strings.Split(value, "\n")
	if len(parts) > lines {
		parts = parts[:lines]
	}
	for index := range parts {
		parts[index] = truncate(parts[index], width)
	}
	return strings.Join(parts, "\n")
}

func (m *model) typeNameOrAll() string {
	if m.typeName == "" {
		return "all"
	}
	return m.typeName
}

func statusName(status domain.Status) string {
	if status == "" {
		return "all"
	}
	return status.String()
}

func (m *model) CycleScope() {
	m.cycleScope()
}

func (m *model) cycleScope() {
	switch m.scope {
	case "active":
		m.scope = "archived"
	case "archived":
		m.scope = "all"
	default:
		m.scope = "active"
	}
}

func CycleScopeForTest(current string) string {
	model := model{scope: current}
	model.CycleScope()
	return model.scope
}

func (m *model) cycleType() {
	switch m.typeName {
	case "":
		m.typeName = "topic"
	case "topic":
		m.typeName = "work-order"
	default:
		m.typeName = ""
	}
}

func (m *model) cycleStatus() {
	statuses := []domain.Status{"", domain.StatusCreate, domain.StatusPending, domain.StatusProgress, domain.StatusReview, domain.StatusBlocked, domain.StatusComplete, domain.StatusFailed, domain.StatusCancelled, domain.StatusArchived}
	for index, status := range statuses {
		if m.status == status {
			m.status = statuses[(index+1)%len(statuses)]
			return
		}
	}
	m.status = ""
}

func isTerminal(value io.Reader) bool {
	file, ok := value.(*os.File)
	return ok && file != nil && isTerminalFile(file)
}

func IsInteractive(input io.Reader, output io.Writer) bool {
	return isTerminal(input) && isTerminalWriter(output)
}

func isTerminalWriter(value io.Writer) bool {
	file, ok := value.(*os.File)
	return ok && file != nil && isTerminalFile(file)
}

var isTerminalFile = func(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func previewAsset(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "asset metadata unavailable"
	}
	return fmt.Sprintf("asset\npath: %s\nextension: %s\nsize: %d bytes", filepath.Base(path), filepath.Ext(path), info.Size())
}
