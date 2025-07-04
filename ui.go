package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
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
	messages []Message
	textarea textarea.Model
	width    int
	height   int
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

	return model{
		messages: []Message{},
		textarea: ta,
		width:    80,
		height:   24,
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+y":
			trimmedMessage := strings.TrimSpace(m.textarea.Value())
			if trimmedMessage != "" {
				// Clear input
				m.textarea.Reset()

				// Add user message
				userMsg := Message{
					Content: trimmedMessage,
					IsUser:  true,
				}
				m.messages = append(m.messages, userMsg)

				go Continue(trimmedMessage)
			}
			return m, nil
		}

	case NewCategoryMessage:
		m.textarea.Reset()
		m.messages = nil

	case AiMessage:
		m.messages = append(m.messages, Message{
			Content: msg.Content,
			IsUser:  false,
		})

	case AiThinkingMessage:
		if msg.Thinking {
			m.textarea.Reset()
			m.textarea.Blur()
		} else {
			m.textarea.Focus()
		}
	}

	// Update textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	// Styles
	userStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		MarginRight(1).
		MarginBottom(1).
		Align(lipgloss.Right)

	llmStyle := lipgloss.NewStyle().
		Padding(0, 1).
		MarginLeft(1).
		MarginBottom(1).
		Align(lipgloss.Left)

	// Calculate available height for messages
	inputHeight := m.textarea.Height() + 2        // +2 for borders
	availableHeight := m.height - inputHeight - 4 // -4 for padding/margins

	// Render messages
	var messageViews []string

	// Calculate how many messages we can show
	totalMessageHeight := 0
	visibleMessages := []Message{}

	// Start from the end and work backwards to show newest messages
	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := m.messages[i]
		// Calculate actual rendered height considering soft wraps
		messageWidth := m.width/2 - 4
		if messageWidth < 10 {
			messageWidth = 10 // minimum width
		}

		lines := strings.Split(msg.Content, "\n")
		totalLines := 0
		for _, line := range lines {
			if len(line) == 0 {
				totalLines++
			} else {
				totalLines += (len(line) + messageWidth - 1) / messageWidth // ceiling division
			}
		}
		msgHeight := totalLines + 1 // +1 for margin

		if totalMessageHeight+msgHeight > availableHeight {
			break
		}

		totalMessageHeight += msgHeight
		visibleMessages = append([]Message{msg}, visibleMessages...)
	}

	// Render visible messages
	for _, msg := range visibleMessages {
		if msg.IsUser {
			// Right-aligned user message
			content := userStyle.Width(m.width/2 - 4).Render(msg.Content)
			messageViews = append(messageViews,
				lipgloss.NewStyle().Width(m.width).Align(lipgloss.Right).Render(content))
		} else {
			// Left-aligned LLM message
			content := llmStyle.Width(m.width/2 - 4).Render(msg.Content)
			messageViews = append(messageViews,
				lipgloss.NewStyle().Width(m.width).Align(lipgloss.Left).Render(content))
		}
	}

	// Join messages
	messagesView := strings.Join(messageViews, "\n")

	// Add padding to push messages up and input down
	remainingHeight := availableHeight - totalMessageHeight
	if remainingHeight > 0 {
		messagesView += strings.Repeat("\n", remainingHeight)
	}

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

	helpView := helpStyle.Render("Enter: new line • Ctrl+Y: send • Ctrl+C: quit")

	// Combine all parts
	return fmt.Sprintf("%s\n%s\n%s", messagesView, inputView, helpView)
}
