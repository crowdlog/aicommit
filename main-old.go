package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/zalando/go-keyring"

	dbmodel "aicommit/.gen/model"
)

type TerminalSize struct {
	Width  int
	Height int
}

// Define the teamodel struct which includes the Bubbletea teamodel elements
type teamodel struct {
	openAIKeyInput textinput.Model
	openAISecret   string
	commitMessage  string
	streamMessage  strings.Builder
	errMsg         error
	hasOpenAIKey   bool
	spinner        spinner.Model
	terminalSize   TerminalSize
	loadingStyle   lipgloss.Style
	loadingView    bool
}

var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
	// helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

// Implement the tea.Model interface for model
func (m *teamodel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}

func (m *teamodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var spinnerCmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		return m, spinnerCmd
	case tea.WindowSizeMsg:
		m.terminalSize.Width = msg.Width
		m.terminalSize.Height = msg.Height
		return m, nil
	case tea.KeyMsg:

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			println("Exiting...")
			return m, tea.Quit
		}
	}

	// ------------------- OpenAI Key Input -------------------

	if !m.hasOpenAIKey {
		if m.openAIKeyInput.Value() != "" {
			switch msg := msg.(type) {

			case tea.KeyMsg:
				switch msg.Type {
				case tea.KeyEnter:
					err := keyring.Set("crowdlog-aicommit", "anon", m.openAIKeyInput.Value())
					if err != nil {
						println("Error saving OpenAI key:", err)
					}
					m.openAISecret = m.openAIKeyInput.Value()
					m.hasOpenAIKey = true
					return m, cmd
				}
			}
		}
	}

	// ------------------- Commit Message Input -------------------
	if m.hasOpenAIKey {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				m.streamMessage.Reset()
				return m, tea.Cmd(func() tea.Msg {
					return runAIMsg("Start")
				})
			}
		case runAIMsg:
			switch msg {
			case "Start":
				m.loadingView = true
				return m, runAI(m)
			case "Done":
				m.loadingView = false
				return m, nil
			}

		}
	}
	var openAIKeyInputCmd tea.Cmd

	m.openAIKeyInput, openAIKeyInputCmd = m.openAIKeyInput.Update(msg)
	cmd = tea.Batch(openAIKeyInputCmd, spinnerCmd)

	return m, cmd
}
func initialModel() teamodel {
	ti := textinput.New()
	ti.Placeholder = "sk-..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	service := "crowdlog-aicommit"
	user := "anon"

	var hasKey = false

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Spinner.FPS = 30
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	secret, err := keyring.Get(service, user)
	// var loadingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).B
	if err != nil {
		hasKey = false
		return teamodel{
			loadingView:    false,
			openAIKeyInput: ti,
			hasOpenAIKey:   hasKey,
			spinner:        s,
			terminalSize:   TerminalSize{Width: 0, Height: 0},
		}
	}
	return teamodel{
		loadingView:    false,
		openAIKeyInput: ti,
		hasOpenAIKey:   true,
		openAISecret:   secret,
		spinner:        s,
		terminalSize:   TerminalSize{Width: 0, Height: 0},
	}
}

func (m *teamodel) View() string {
	if !m.loadingView {
		if m.commitMessage == "" {
			return mainContentStyle.Render(
				"Press enter to generate commit message",
			)
		}
		return fmt.Sprintf("Commit Message: %s\n", wordwrap.String(m.commitMessage, m.terminalSize.Width))
	}

	// Implement the logic to render the view based on the model state
	if m.errMsg != nil {
		return fmt.Sprintf("Error: %s", m.errMsg.Error())
	}

	if !m.hasOpenAIKey {
		return fmt.Sprintf(
			"No OpenAI key detected. Please enter it below:\n\n%s\n\n%s",
			m.openAIKeyInput.View(),
			"(esc to quit)",
		) + "\n"
	}
	spin := m.spinner.View() + "Generating commit message..."
	msg := wordwrap.String(m.streamMessage.String(), m.terminalSize.Width)
	return spin + "\n\n" + msg
}

// func main() {
// 	initLogger()
// 	log.Debug().Msg("Starting up...")
// 	initialize := true
// 	cdb, err := getCommitDBFactory(initialize)
// 	if err != nil {
// 		println("Error downloading and installing SQLite:", err.Error())
// 		return
// 	}
// 	_, err = cdb.GetUserSettings()
// 	if err != nil {
// 		println("Error getting user settings:", err.Error())
// 		return
// 	}
// 	var rootCmd = &cobra.Command{
// 		Use:   "myapp",
// 		Short: "Git Commit Message Generator",
// 	}

// 	var cmdAICommit = &cobra.Command{
// 		Use:   "aicommit",
// 		Short: "Generate commit message using AI",
// 		Run:   runTea,
// 	}

// 	rootCmd.AddCommand(cmdAICommit)
// 	rootCmd.Execute()
// }

// func runTea(cmd *cobra.Command, args []string) {
// 	m := initialModel()
// 	p := tea.NewProgram(&m)
// 	if _, err := p.Run(); err != nil {
// 		fmt.Println("Error running program:", err)
// 		return
// 	}

// }

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "-U10", "head")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func generateCommitMessageUsingAI(gitDiff string, m *teamodel) (*schema.AIChatMessage, error) {

	llm, err := openai.NewChat(openai.WithModel("gpt-3.5-turbo-1106"), openai.WithToken(m.openAISecret))
	if err != nil {
		return nil, err
	}

	// Create a channel to stream the responses
	stream := make(chan []byte)

	var chats = []schema.ChatMessage{
		schema.SystemChatMessage{Content: "Generate a short commit message. "},
		schema.HumanChatMessage{Content: gitDiff},
	}
	completion, err := llm.Call(context.Background(), chats, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		// m.streamMessage.WriteString(string(chunk))
		return nil
	}))

	if err != nil {
		println("Error calling AI:", err.Error())
		close(stream)
		return nil, err
	}

	return completion, nil

}

type loadingMsg string

type runAIMsg string

func runAI(m *teamodel) tea.Cmd {

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
		promptBytes, err := json.Marshal([]string{"Generate a short commit message. Use conventional commits. "})
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

		_, err = generateCommitMessageUsingAI(gitDiff, m)
		if err != nil {
			println("Error generating commit message using AI:", err.Error())
			return nil
		}
		return runAIMsg("Done")
	}
}
