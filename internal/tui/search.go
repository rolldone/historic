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
	header := accent.Render("Historic Search") + "\n" + fmt.Sprintf("Query: %s | status=%s scope=%s type=%s limit=%d | page %d/%d", m.input.View(), statusName(m.status), m.scope, m.typeNameOrAll(), m.limit, m.page+1, m.pageCount())
	if m.focus == focusFilter {
		header += "\n" + muted.Render("Filter focus: s scope; v status; t type; Esc/f closes")
	}
	if m.errorText != "" {
		header += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F56")).Render(m.errorText)
	}
	left := "Results\n"
	for index, result := range m.currentPage() {
		marker := "  "
		if index == m.selected {
			marker = "> "
		}
		left += fmt.Sprintf("%s%s %s\n", marker, result.ID, result.Title)
	}
	if len(m.currentPage()) == 0 {
		left += muted.Render("No matches") + "\n"
	}
	right := "Preview\n" + m.preview
	body := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(42).Render(left), lipgloss.NewStyle().Width(72).Render(right))
	footer := muted.Render("↑↓ navigate  Enter preview  / query  f filter(s scope/v status/t type)  n/p page  r refresh  q quit")
	return header + "\n\n" + body + "\n\n" + footer
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
