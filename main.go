package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/zalando/go-keyring"
)

type TerminalSize struct {
	Width  int
	Height int
}

// Define the model struct which includes the Bubbletea model elements
type model struct {
	ready          bool
	openAIKeyInput textinput.Model
	openAISecret   string
	commitMessage  strings.Builder
	errMsg         error
	hasOpenAIKey   bool
	viewport       viewport.Model
	spinner        spinner.Model
	isFetching     bool
	terminalSize   TerminalSize
}

// Implement the tea.Model interface for model
func (m *model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = false
		var prevContent = m.viewport.View()
		m.viewport.SetContent("")
		m.terminalSize.Width = msg.Width
		m.terminalSize.Height = msg.Height
		m.viewport.SetContent(prevContent)
		m.ready = true
		return m, nil
	case tea.KeyMsg:

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			println("Exiting...")
			return m, tea.Quit
		}
	}

	if !m.hasOpenAIKey {
		if m.openAIKeyInput.Value() != "" {
			switch msg := msg.(type) {

			case tea.KeyMsg:
				switch msg.Type {

				case tea.KeyEnter:
					println("Saving OpenAI key...")
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

	if m.hasOpenAIKey {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				cmd = runAI(m)
				return m, cmd
			}
		case tickMsg:
			m.viewport.SetContent(m.commitMessage.String())
			m.viewport.GotoBottom()
			return m, cmd
		}
	}
	var openAIKeyInputCmd tea.Cmd
	var spinnerCmd tea.Cmd
	m.openAIKeyInput, openAIKeyInputCmd = m.openAIKeyInput.Update(msg)
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	cmd = tea.Batch(openAIKeyInputCmd, spinnerCmd)

	return m, cmd
}

type tickMsg time.Time

func initialModel() model {
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
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	secret, err := keyring.Get(service, user)
	if err != nil {
		hasKey = false
		return model{
			openAIKeyInput: ti,
			hasOpenAIKey:   hasKey,
			spinner:        s,
			terminalSize:   TerminalSize{Width: 0, Height: 0},
			ready:          false,
		}
	}
	return model{
		openAIKeyInput: ti,
		hasOpenAIKey:   true,
		openAISecret:   secret,
		spinner:        s,
		terminalSize:   TerminalSize{Width: 0, Height: 0},
		ready:          false,
	}
}

func (m *model) View() string {

	if !m.ready {
		return "Initializing..."
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

	var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Width(m.terminalSize.Width)

	if m.isFetching {
		return helpStyle.Render(fmt.Sprintf(
			"%s | %s", m.spinner.View(), m.commitMessage.String(),
		))
	}

	if m.commitMessage.String() == "" {
		return helpStyle.Render(
			"Press enter to generate commit message",
		)
	}

	return helpStyle.Render(fmt.Sprintf("Commit Message: %s\n", wordwrap.String(m.commitMessage.String(), m.terminalSize.Width)))
}

func main() {
	err := downloadAndInstallSqlite()
	if err != nil {
		println("Error downloading and installing SQLite:", err.Error())
		return
	}
	var rootCmd = &cobra.Command{
		Use:   "myapp",
		Short: "Git Commit Message Generator",
	}

	var cmdAICommit = &cobra.Command{
		Use:   "aicommit",
		Short: "Generate commit message using AI",
		Run:   runTea,
	}

	rootCmd.AddCommand(cmdAICommit)
	rootCmd.Execute()
}

func runTea(cmd *cobra.Command, args []string) {
	m := initialModel()
	p := tea.NewProgram(&m)
	go func() {
		for c := range time.Tick(20 * time.Millisecond) {
			if m.commitMessage.String() != "" {
				return
			}
			p.Send(tickMsg(c))
		}
	}()
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		return
	}

}

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "head")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func generateCommitMessageUsingAI(gitDiff string, m *model) (*schema.AIChatMessage, error) {

	llm, err := openai.NewChat(openai.WithModel("gpt-4-1106-preview"), openai.WithToken(m.openAISecret))
	if err != nil {
		return nil, err
	}

	// Create a channel to stream the responses
	stream := make(chan []byte)

	var chats = []schema.ChatMessage{
		schema.SystemChatMessage{Content: "Generate a short commit message. Use conventional commits. "},
		schema.HumanChatMessage{Content: gitDiff},
	}
	m.isFetching = true
	completion, err := llm.Call(context.Background(), chats, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		m.commitMessage.WriteString(string(chunk))
		return nil
	}))

	if err != nil {
		println("Error calling AI:", err.Error())
		close(stream)
		return nil, err
	}
	m.isFetching = false

	return completion, nil

}

func runAI(m *model) tea.Cmd {

	return func() tea.Msg {
		m.isFetching = true
		gitDiff, err := getGitDiff()
		if err != nil {
			println("Error getting git diff:", err.Error())
			return nil
		}

		_, err = generateCommitMessageUsingAI(gitDiff, m)
		if err != nil {
			println("Error generating commit message using AI:", err.Error())
			return nil
		}

		return nil
	}
}
