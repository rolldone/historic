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

// Sort modes cycled by the `s` keybinding in the results view. Relevance is
// the default and maps to an empty search sort option, which preserves the
// pre-existing behaviour (FTS rank for queries, activity order for recent
// topics).
const (
	SortRelevance = "relevance"
	SortUpdated   = "updated"
	SortCreated   = "created"
	SortTitle     = "title"
)

var sortModes = []string{SortRelevance, SortUpdated, SortCreated, SortTitle}

type Options struct {
	Input  io.Reader
	Output io.Writer
}

type model struct {
	workspace  config.Workspace
	input      textinput.Model
	results    []search.Result
	selected   int
	page       int
	limit      int
	pagination search.Pagination
	status     domain.Status
	scope      string
	typeName   string
	sortMode   string
	focus      focusMode
	errorText  string
	// previewOffset scrolls the structured preview when it exceeds the
	// viewport. It resets whenever the selection changes.
	previewOffset int
	width         int
	height        int
}

type focusMode int

const (
	focusResults focusMode = iota
	focusQuery
	focusFilter
)

type refreshMsg struct{}
type refreshResultMsg struct {
	page search.SearchPage
	err  error
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
	return model{workspace: workspace, input: input, limit: DefaultLimit, scope: "all", sortMode: SortRelevance, focus: focusResults}
}

func (m model) Init() tea.Cmd { return m.refresh() }

func (m model) refresh() tea.Cmd {
	keyword := strings.TrimSpace(m.input.Value())
	options := search.Options{Keyword: keyword, Page: m.page + 1, PageSize: m.limit, OpenOnly: m.scope == "active", ClosedOnly: m.scope == "archived", Status: m.status, Sort: m.sortOption()}
	if m.typeName == "work-order" {
		options.Type = "task"
	}
	if m.typeName == "topic" {
		options.Type = "meta"
	}
	if keyword == "" {
		return func() tea.Msg {
			page, err := recentTopics(m.workspace, options)
			return refreshResultMsg{page: page, err: err}
		}
	}
	return func() tea.Msg {
		page, err := search.FindPage(m.workspace, options)
		return refreshResultMsg{page: page, err: err}
	}
}

func recentTopics(workspace config.Workspace, options search.Options) (search.SearchPage, error) {
	return search.RecentTopicsPage(workspace, options)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(message)
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case refreshResultMsg:
		m.results, m.errorText = message.page.Items, ""
		if message.err != nil {
			m.errorText = message.err.Error()
		} else {
			m.page = message.page.Pagination.Page - 1
			m.pagination = message.page.Pagination
		}
		m.selected = 0
		m.previewOffset = 0
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
	case "s":
		m.cycleSort()
		m.selected = 0
		m.previewOffset = 0
		return m, m.refresh()
	case "r":
		return m, m.refresh()
	case "n", "right":
		if m.pageData().Pagination.HasMore {
			m.page++
			m.selected = 0
			return m, m.refresh()
		}
	case "p", "left":
		if m.page > 0 {
			m.page--
			m.selected = 0
			return m, m.refresh()
		}
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.previewOffset = 0
		}
	case "down", "j":
		if m.selected+1 < len(m.currentPage()) {
			m.selected++
			m.previewOffset = 0
		}
	case "ctrl+d", "pgdown":
		m.previewOffset++
	case "ctrl+u", "pgup":
		if m.previewOffset > 0 {
			m.previewOffset--
		}
	case "enter":
		m.previewOffset = 0
	case "esc":
		m.focus = focusResults
	}
	return m, nil
}

func (m model) pageData() search.SearchPage {
	pagination := m.pagination
	if pagination.Page == 0 {
		pagination = search.Pagination{Page: m.page + 1, PageSize: m.limit}
	}
	return search.SearchPage{Items: m.results, Pagination: pagination}
}

func (m model) currentPage() []search.Result {
	if m.pagination.Page > 0 {
		return m.results
	}
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
	pageData := m.pageData()
	paginationText := "No more results"
	if pageData.Pagination.HasMore {
		paginationText = "More available"
	}
	header := accent.Render("Historic Search") + "\n" + truncate(fmt.Sprintf("Query: %s | status=%s scope=%s type=%s sort=%s page-size=%d", m.input.View(), statusName(m.status), m.scope, m.typeNameOrAll(), m.sortName(), m.limit), width)
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
	previewHeight := visible - 1
	if previewHeight < 1 {
		previewHeight = 1
	}
	previewWidth := rightWidth
	if width < 72 {
		previewWidth = width
	}
	previewBlock := m.visiblePreview(previewWidth, previewHeight)
	left := "Results\n"
	for index := start; index < len(rows) && index < start+visible; index++ {
		result := rows[index]
		marker := "  "
		if index == m.selected {
			marker = "> "
		}
		storage := strings.ToUpper(result.Storage)
		updated := result.UpdatedAt
		if updated == "" {
			updated = result.Mtime
		}
		left += truncate(fmt.Sprintf("%s[%s] %s %s · %s", marker, storage, result.ID, result.Title, updated), leftWidth) + "\n"
	}
	if len(rows) == 0 {
		left += muted.Render("No matches") + "\n"
	}
	right := "Preview\n" + strings.Join(previewBlock, "\n")
	var body string
	if width < 72 {
		body = truncateLines(left, width, contentHeight) + "\n" + truncateLines(right, width, contentHeight)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftWidth).Render(left), lipgloss.NewStyle().Width(rightWidth).Render(right))
	}
	footerText := fmt.Sprintf("Page %d · showing %d results · %s | sort=%s | ↑↓ nav  Ctrl+D/U scroll  / query  f filter  s sort  n/→ next  p/← prev  r refresh  q quit", m.page+1, len(m.currentPage()), paginationText, m.sortName())
	if width < 72 {
		footerText = fmt.Sprintf("↑↓ navigate · Page %d · %d results · %s · sort=%s", m.page+1, len(m.currentPage()), paginationText, m.sortName())
	}
	footer := muted.Render(truncate(footerText, width))
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
	case "all":
		m.scope = "active"
	case "active":
		m.scope = "archived"
	default:
		m.scope = "all"
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
	statuses := []domain.Status{"", domain.StatusCreate, domain.StatusDraft, domain.StatusPending, domain.StatusPlanned, domain.StatusProgress, domain.StatusReview, domain.StatusBlocked, domain.StatusComplete, domain.StatusFailed, domain.StatusCancelled, domain.StatusArchived, domain.StatusForwarded}
	for index, status := range statuses {
		if m.status == status {
			m.status = statuses[(index+1)%len(statuses)]
			return
		}
	}
	m.status = ""
}

// cycleSort advances the TUI sort mode: relevance → updated → created → title.
func (m *model) cycleSort() {
	current := m.sortName()
	for index, mode := range sortModes {
		if current == mode {
			m.sortMode = sortModes[(index+1)%len(sortModes)]
			return
		}
	}
	m.sortMode = SortRelevance
}

// sortName normalizes the stored mode so an empty value reads as the default.
func (m *model) sortName() string {
	if m.sortMode == "" {
		return SortRelevance
	}
	return m.sortMode
}

// sortOption maps a TUI sort mode onto the search layer's sort option.
// Relevance is expressed as an empty option, which keeps the default FTS rank
// ordering for queries and the activity ordering for recent topics.
func (m *model) sortOption() string {
	switch m.sortName() {
	case SortUpdated, SortCreated, SortTitle:
		return m.sortName()
	default:
		return ""
	}
}

// CycleSortForTest exposes the sort cycle to unit tests.
func CycleSortForTest(current string) string {
	model := model{sortMode: current}
	model.cycleSort()
	return model.sortName()
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
