package main

// A simple example that shows how to send activity to Bubble Tea in real-time
// through a channel.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/zalando/go-keyring"

	dbmodel "aicommit/.gen/model"
)

var (
	mainContentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
)

type ScreenView int

const (
	SetupView         ScreenView = 0 // 0
	CommitMessageView ScreenView = 1 // null, \0
)

type model struct {
	quitting     bool
	openAISecret string

	view ScreenView

	genMessageState struct {
		sub           chan string // where we'll receive activity notifications
		responses     int         // how many responses we've received
		loading       bool
		spinner       spinner.Model
		commitMessage *strings.Builder
	}

	terminalWidth  int
	terminalHeight int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForActivity(m.genMessageState.sub), // wait for activity
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.view == CommitMessageView {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.terminalWidth = msg.Width
			m.terminalHeight = msg.Height
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "enter":
				m.genMessageState.responses = 0
				m.genMessageState.loading = true
				m.genMessageState.commitMessage.Reset()
				return m, tea.Batch(runAI2(&m), m.genMessageState.spinner.Tick, tea.ClearScreen)
			default:
				return m, nil
			}
		case responseMsg:
			m.genMessageState.responses++                                   // record external activity
			m.genMessageState.commitMessage.WriteString(msg.messageContent) // update the model
			return m, waitForActivity(m.genMessageState.sub)                // wait for next event
		case spinner.TickMsg:
			var genMessageSpinnerCmd tea.Cmd
			if m.genMessageState.loading {
				m.genMessageState.spinner, genMessageSpinnerCmd = m.genMessageState.spinner.Update(msg)
			}
			return m, tea.Batch(genMessageSpinnerCmd)
		case genMsg:
			if msg.msgType == "Done" {
				m.genMessageState.loading = false
				m.genMessageState.commitMessage.Reset()
				m.genMessageState.commitMessage.WriteString(msg.Content)
				return m, nil
			}
			return m, nil
		default:
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() string {
	commitMessage := m.genMessageState.commitMessage.String()
	s := mainContentStyle.Width(m.terminalWidth).Render(fmt.Sprintf("\n %s Events received: %d\n\n Press any key to exit\n %s", m.genMessageState.spinner.View(), m.genMessageState.responses, commitMessage))
	if m.quitting {
		s += "\n"
	}
	return s
}

func main() {
	secret, err := keyring.Get("crowdlog-aicommit", "anon")
	if err != nil {
		println("Error getting OpenAI key:", err)
	}
	p := tea.NewProgram(model{

		genMessageState: struct {
			sub           chan string
			responses     int
			loading       bool
			spinner       spinner.Model
			commitMessage *strings.Builder
		}{
			sub:           make(chan string),
			responses:     0,
			loading:       false,
			spinner:       spinner.New(),
			commitMessage: &strings.Builder{},
		},
		view:         CommitMessageView,
		openAISecret: secret,
	})

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}

// ---------------- Commit Message Generation ----------------

// A message used to indicate that activity has occurred. In the real world (for
// example, chat) this would contain actual data.
type responseMsg struct {
	messageContent string
}

// A command that waits for the activity on a channel.
func waitForActivity(stream chan string) tea.Cmd {
	return func() tea.Msg {
		content := <-stream
		return responseMsg{messageContent: content}
	}
}

type genMsg struct {
	Content string
	msgType string
}

func runAI2(m *model) tea.Cmd {

	return func() tea.Msg {
		gitDiff, err := getGitDiff()
		if err != nil {
			println("Error getting git diff:", err.Error())
			return nil
		}

		dateCreated := time.Now()
		diffStructuredJson := ""
		model := "gpt-4-1106-preview"
		aiProvider := "openai"
		promptBytes, err := json.Marshal([]string{"Generate a short commit message."})
		if err != nil {
			println("Error marshalling prompts:", err.Error())
			return nil
		}
		prompts := string(promptBytes)

		var gitDiffRow = &dbmodel.Diff{
			Diff:               &gitDiff,
			DateCreated:        &dateCreated,
			DiffStructuredJSON: &diffStructuredJson,
			Model:              &model,
			AiProvider:         &aiProvider,
			Prompts:            &prompts,
		}
		initialize := false
		cdb, err := getCommitDBFactory(initialize)
		if err != nil {
			println("Error downloading and installing SQLite:", err.Error())
			return nil
		}
		_, err = cdb.InsertDiff(*gitDiffRow)
		if err != nil {
			println("Error inserting diff:", err.Error())
			return nil
		}

		main2()

		structuredDiff, err := cdb.GetDiff()
		if err != nil {
			println("Error getting git diff:", err.Error())
			return nil
		}

		if structuredDiff.DiffStructuredJSON == nil {
			println("Error getting structured diff:", err.Error())
		}

		content, err := genMessage(gitDiff, m)
		if err != nil {
			println("Error generating commit message using AI:", err.Error())
			return nil
		}
		return genMsg{
			Content: content.Content,
			msgType: "Done",
		}
	}
}

func genMessage(gitDiff string, m *model) (*schema.AIChatMessage, error) {

	llm, err := openai.NewChat(openai.WithModel("gpt-3.5-turbo-1106"), openai.WithToken(m.openAISecret))
	if err != nil {
		return nil, err
	}

	var chats = []schema.ChatMessage{
		schema.SystemChatMessage{Content: "Generate a short commit message. "},
		schema.HumanChatMessage{Content: gitDiff},
	}
	completion, err := llm.Call(context.Background(), chats, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		m.genMessageState.sub <- string(chunk)
		return nil
	}))

	if err != nil {
		println("Error calling AI:", err.Error())
		return nil, err
	}

	return completion, nil

}
