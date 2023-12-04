package main

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// Define the model struct which includes the Bubbletea model elements
type model struct {
	openAIKeyInput textinput.Model
	openAISecret   string
	gitDiff        string
	commitMessage  strings.Builder
	errMsg         error
	hasOpenAIKey   bool
	viewport       viewport.Model
}

// Implement the tea.Model interface for model
func (m *model) Init() tea.Cmd {

	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			println("Exiting...")
			return m, tea.Quit
		}
		// if msg.String() == "c" {
		// 	gitdiff, err := getGitDiff()
		// 	println("getting git diff")
		// 	if err != nil {
		// 		println(gitdiff)
		// 	}
		// }
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
				println("Generating commit message...")

				cmd = someCmd(m)
				return m, cmd
			}
		case tickMsg:
			println(m.commitMessage.String(), "angelo")
			m.viewport.SetContent(m.commitMessage.String())
			m.viewport.GotoBottom()
		}

	}
	m.openAIKeyInput, cmd = m.openAIKeyInput.Update(msg)

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

	secret, err := keyring.Get(service, user)
	if err != nil {
		hasKey = false
		return model{
			openAIKeyInput: ti,
			hasOpenAIKey:   hasKey,
		}
	}
	println("secret: ", secret)
	return model{
		openAIKeyInput: ti,
		hasOpenAIKey:   true,
		openAISecret:   secret,
	}
}

func (m *model) View() string {
	tea.LogToFile("log.txt", "hello")
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
	return fmt.Sprintf("Commit Message testing: %s\n", m.commitMessage.String())
}

func main() {
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

func generateCommitMessageUsingAI(gitDiff string, m *model) (openai.ChatCompletionStream, error) {
	println("generating commit message using AI. gitdiff:", gitDiff)

	client := openai.NewClient(m.openAISecret)
	resp, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "This is a commit message generator. Generate short commit messages. Use conventional commits to describe your changes. ",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: gitDiff,
				},
			},
		},
	)
	if err != nil {
		return openai.ChatCompletionStream{}, err
	}
	return *resp, err
}

func someCmd(m *model) tea.Cmd {

	return func() tea.Msg {
		gitDiff, err := getGitDiff()
		if err != nil {
			println("Error getting git diff:", err)
			return nil
		}

		resp, err := generateCommitMessageUsingAI(gitDiff, m)
		if err != nil {
			println("Error generating commit message using AI:", err)
			return nil
		}

		for {
			resp, recvErr := resp.Recv()
			if recvErr == io.EOF {
				// End of the stream
				break
			}
			if recvErr != nil {
				println("Error receiving response:", recvErr)
				return nil
			}

			// Assuming the response has a field that contains the message part
			m.commitMessage.Write([]byte(resp.Choices[0].Delta.Content))
			// println("commit message: ", m.commitMessage)

		}

		return nil
	}
}
