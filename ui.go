package main

import (
	"fmt"
	"strings"

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

	return model{
		messages:         []Message{},
		textarea:         ta,
		viewport:         vp,
		width:            80,
		height:           24,
		markdownRenderer: renderer,
	}
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+y":
			trimmedMessage := strings.TrimSpace(m.textarea.Value())
			if trimmedMessage != "" {
				m.textarea.Reset()

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
		}

	case NewCategoryMessage:
		m.textarea.Reset()
		m.messages = nil
		m.updateViewport()

	case AiMessage:
		m.messages = append(m.messages, Message{
			Content: msg.Content,
			IsUser:  false,
		})
		m.updateViewport()

	case AiThinkingMessage:
		if msg.Thinking {
			m.textarea.Reset()
			m.textarea.Blur()
		} else {
			m.textarea.Focus()
		}
	}

	// Update textarea and viewport
	var cmd tea.Cmd
	if m.textarea.Focused() {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateViewport() {
	// Update renderer width
	m.markdownRenderer, _ = glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(m.width/2-8),
	)

	// Styles
	userStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		MarginRight(1).
		MarginBottom(1).
		Align(lipgloss.Left)

	llmStyle := lipgloss.NewStyle().
		Padding(0, 1).
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

	helpView := helpStyle.Render("Enter: new line • Ctrl+Y: send • Ctrl+C: quit • ↑/↓: scroll")

	// Combine all parts
	return fmt.Sprintf("%s\n%s\n%s", m.viewport.View(), inputView, helpView)
}
