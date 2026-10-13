package tui

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/repository"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TopicPickerOptions configures the one-shot topic selection UI.
type TopicPickerOptions struct {
	Input  io.Reader
	Output io.Writer
	Closed bool
}

// PickTopic chooses exactly one topic or returns (nil, nil) when cancelled.
// Topic views are read-only and sorted newest-first by frontmatter updated date,
// falling back to metadata mtime, then ID/path.
func PickTopic(workspace config.Workspace, options TopicPickerOptions) (*repository.TopicView, error) {
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if !IsInteractive(options.Input, options.Output) {
		return nil, fmt.Errorf("interactive topic selection requires a terminal; provide an ID or --all")
	}
	store := repository.NewTopicStore(workspace)
	views, err := store.ListTopics(options.Closed)
	if err != nil {
		return nil, err
	}
	if len(views) == 0 {
		return nil, nil
	}
	for index := range views {
		views[index].Body = strings.ToLower(strings.Join([]string{
			views[index].ID, views[index].Title, views[index].Description, views[index].Path,
			readTopicSearchText(workspace, views[index]),
		}, " "))
	}
	sort.SliceStable(views, func(i, j int) bool {
		left, right := pickerTimestamp(views[i]), pickerTimestamp(views[j])
		if !left.Equal(right) {
			return left.After(right)
		}
		if views[i].ID != views[j].ID {
			return views[i].ID < views[j].ID
		}
		return views[i].Path < views[j].Path
	})
	model := topicPickerModel{views: views, input: options.Input, selected: 0, width: 100, height: 30}
	program := tea.NewProgram(model, tea.WithInput(options.Input), tea.WithOutput(options.Output), tea.WithAltScreen())
	final, err := program.Run()
	if err != nil {
		return nil, err
	}
	result, ok := final.(topicPickerModel)
	if !ok || result.cancelled || result.selected < 0 || result.selected >= len(result.filtered()) {
		return nil, nil
	}
	view := result.filtered()[result.selected]
	view.Body = ""
	return &view, nil
}

func readTopicSearchText(workspace config.Workspace, view repository.TopicView) string {
	data, err := os.ReadFile(workspace.Root + string(os.PathSeparator) + view.Path + string(os.PathSeparator) + "_meta.yaml")
	if err != nil {
		return ""
	}
	return string(data)
}

func pickerTimestamp(view repository.TopicView) time.Time {
	if view.Updated != "" {
		if parsed, err := time.Parse("2006-01-02", view.Updated); err == nil {
			return parsed
		}
		if parsed, err := time.Parse(time.RFC3339Nano, view.Updated); err == nil {
			return parsed
		}
	}
	parsed, _ := time.Parse(time.RFC3339Nano, view.Mtime)
	return parsed
}

type topicPickerModel struct {
	views     []repository.TopicView
	input     io.Reader
	query     string
	selected  int
	width     int
	height    int
	cancelled bool
}

func (m topicPickerModel) Init() tea.Cmd { return nil }

func (m topicPickerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyRunes:
			if len(msg.Runes) == 1 && msg.Runes[0] == 'q' && m.query == "" {
				m.cancelled = true
				return m, tea.Quit
			}
			m.query += string(msg.Runes)
			m.selected = 0
		case tea.KeyEnter:
			if len(m.filtered()) > 0 {
				return m, tea.Quit
			}
		case tea.KeyUp:
			if m.selected > 0 {
				m.selected--
			}
		case tea.KeyDown:
			if m.selected+1 < len(m.filtered()) {
				m.selected++
			}
		case tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.selected = 0
			}
		}
	}
	return m, nil
}

func (m topicPickerModel) filtered() []repository.TopicView {
	query := strings.ToLower(strings.TrimSpace(m.query))
	result := make([]repository.TopicView, 0, len(m.views))
	for _, view := range m.views {
		if query == "" || strings.Contains(view.Body, query) {
			result = append(result, view)
		}
	}
	return result
}

func (m topicPickerModel) View() string {
	width, height := m.width, m.height
	if width < 20 {
		width = 20
	}
	if height < 5 {
		height = 5
	}
	rows := m.filtered()
	visible := height - 5
	if visible < 1 {
		visible = 1
	}
	start := 0
	if m.selected >= visible {
		start = m.selected - visible + 1
	}
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Select topic (↑/↓ move · Enter select · Esc cancel)"))
	lines = append(lines, "Search: "+m.query)
	for index := start; index < len(rows) && index < start+visible; index++ {
		marker := "  "
		if index == m.selected {
			marker = "> "
		}
		view := rows[index]
		date := view.Updated
		if date == "" {
			date = view.Mtime
		}
		lines = append(lines, truncate(fmt.Sprintf("%s[%s] %s %s · %s", marker, strings.ToUpper(view.Storage), view.ID, view.Title, date), width))
	}
	if len(rows) == 0 {
		lines = append(lines, "No topics match the search.")
	}
	lines = append(lines, fmt.Sprintf("%d topics · type to search · Backspace deletes", len(rows)))
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
