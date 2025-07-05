package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var quitErr error

func Err(err error) {
	quitErr = err
	teaProgram.Quit()
}

type Message struct {
	Content string
	IsUser  bool
}

type model struct {
	messages         []Message
	textarea         textarea.Model
	viewport         viewport.Model
	width            int
	height           int
	markdownRenderer *glamour.TermRenderer

	history      [][]byte // Compressed snapshots
	historyIndex int      // Current position in history
	historySize  int      // Current number of items in history
	maxHistory   int      // Maximum history size (50)
	lastSnapshot string   // Last saved snapshot to avoid duplicates
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("0"))
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(5)

	vp := viewport.New(80, 15)

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(40),
	)

	m := model{
		messages:         []Message{},
		textarea:         ta,
		viewport:         vp,
		width:            80,
		height:           24,
		markdownRenderer: renderer,
		history:          make([][]byte, 50),
		historyIndex:     -1,
		historySize:      0,
		maxHistory:       50,
		lastSnapshot:     "",
	}

	// Save initial empty state
	m.saveSnapshot()
	return m
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)

		// Update viewport size
		inputHeight := m.textarea.Height() + 4 // +4 for borders and help
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - inputHeight
		m.viewport.GotoBottom()
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+y":
			trimmedMessage := strings.TrimSpace(m.textarea.Value())
			if trimmedMessage != "" {
				m.textarea.Reset()
				m.clearHistory()

				// Add user message
				userMsg := Message{
					Content: trimmedMessage,
					IsUser:  true,
				}
				m.messages = append(m.messages, userMsg)
				m.updateViewport()

				go Continue(trimmedMessage)
			}
			return m, nil

		case "ctrl+z": // Undo
			m.undo()
			return m, nil

		case "ctrl+r": // Redo
			m.redo()
			return m, nil

		default:
			// Save snapshot on significant changes
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			newValue := m.textarea.Value()

			// Save snapshot if content changed significantly
			if levenshtein.ComputeDistance(m.lastSnapshot, newValue) > 5 {
				m.saveSnapshot()
			}

			cmds = append(cmds, cmd)
		}

	case NewCategoryMessage:
		m.textarea.Reset()
		m.clearHistory()
		m.messages = nil
		m.updateViewport()

	case AiMessage:
		m.messages = append(m.messages, Message{
			Content: msg.Content,
			IsUser:  false,
		})
		m.viewport.GotoBottom()
		m.updateViewport()

	case AiThinkingMessage:
		if msg.Thinking {
			m.textarea.Reset()
			m.clearHistory()
			m.textarea.Blur()
		} else {
			m.textarea.Focus()
		}
	}

	func() {
		if _, ok := msg.(tea.MouseMsg); ok {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}()

	return m, tea.Batch(cmds...)
}

func (m *model) updateViewport() {
	// Update renderer width
	m.markdownRenderer, _ = glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(m.width/2-10),
	)

	// Styles
	userStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0).
		MarginRight(1).
		MarginBottom(1).
		Align(lipgloss.Left)

	llmStyle := lipgloss.NewStyle().
		Padding(0).
		MarginLeft(1).
		MarginBottom(1).
		Align(lipgloss.Left)

	// Render all messages
	var messageViews []string

	for _, msg := range m.messages {
		var content string

		if msg.IsUser {
			rendered, err := m.markdownRenderer.Render(msg.Content)
			if err != nil {
				rendered = msg.Content
			}
			content = userStyle.Width(m.width/2 - 4).Render(strings.TrimSpace(rendered))
			messageViews = append(messageViews,
				lipgloss.NewStyle().Width(m.width).Align(lipgloss.Right).Render(content))
		} else {
			rendered, err := m.markdownRenderer.Render(msg.Content)
			if err != nil {
				rendered = msg.Content
			}
			content = llmStyle.Width(m.width/2 - 4).Render(strings.TrimSpace(rendered))
			messageViews = append(messageViews,
				lipgloss.NewStyle().Width(m.width).Align(lipgloss.Left).Render(content))
		}
	}

	// Set viewport content
	m.viewport.SetContent(strings.Join(messageViews, "\n"))

	// Auto-scroll to bottom
	m.viewport.GotoBottom()
}

func (m model) View() string {
	// Input area
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1)

	inputView := inputStyle.Render(m.textarea.View())

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginTop(1)

	helpView := helpStyle.Render("Enter: new line • Ctrl+Y: send • Ctrl+C: quit • mouse wheel: scroll")

	// Combine all parts
	return fmt.Sprintf("%s\n%s\n%s", m.viewport.View(), inputView, helpView)
}

// Compress text using gzip
func (m *model) compressText(text string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(text))
	if err != nil {
		return nil, err
	}
	gz.Close()
	return buf.Bytes(), nil
}

// Decompress text from gzip
func (m *model) decompressText(data []byte) (string, error) {
	buf := bytes.NewBuffer(data)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	result, err := io.ReadAll(gz)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// Save current textarea content as snapshot
func (m *model) saveSnapshot() {
	content := m.textarea.Value()

	// Don't save if content is the same as last snapshot
	if content == m.lastSnapshot {
		return
	}

	compressed, err := m.compressText(content)
	if err != nil {
		return
	}

	// Clear any redo history when saving new snapshot
	m.historySize = m.historyIndex + 1

	// Move to next position in circular buffer
	m.historyIndex = (m.historyIndex + 1) % m.maxHistory

	// If we're at capacity, we're overwriting the oldest entry
	if m.historySize < m.maxHistory {
		m.historySize++
	}

	m.history[m.historyIndex] = compressed
	m.lastSnapshot = content
}

// Undo to previous snapshot
func (m *model) undo() {
	if m.historyIndex <= 0 || m.historySize <= 1 {
		return
	}

	m.historyIndex--
	if m.historyIndex < 0 {
		m.historyIndex = m.maxHistory - 1
	}

	if m.history[m.historyIndex] != nil {
		content, err := m.decompressText(m.history[m.historyIndex])
		if err == nil {
			m.textarea.SetValue(content)
			m.lastSnapshot = content
		}
	}
}

// Redo to next snapshot
func (m *model) redo() {
	if m.historyIndex >= m.historySize-1 {
		return
	}

	nextIndex := (m.historyIndex + 1) % m.maxHistory
	if nextIndex < m.historySize && m.history[nextIndex] != nil {
		content, err := m.decompressText(m.history[nextIndex])
		if err == nil {
			m.textarea.SetValue(content)
			m.lastSnapshot = content
			m.historyIndex = nextIndex
		}
	}
}

// Clear history
func (m *model) clearHistory() {
	m.history = make([][]byte, m.maxHistory)
	m.historyIndex = -1
	m.historySize = 0
	m.lastSnapshot = ""
	m.saveSnapshot()
}
