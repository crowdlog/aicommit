package main

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
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
	commitMessage  string
	errMsg         error
	hasOpenAIKey   bool
}

// Implement the tea.Model interface for model
func (m model) Init() tea.Cmd {

	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

				commitMsg, err := runAICommit(m)
				if err != nil {
					println("Error generating commit message:", err)
				}
				m.commitMessage = commitMsg
				return m, cmd
			}
		}
	}
	m.openAIKeyInput, cmd = m.openAIKeyInput.Update(msg)

	return m, cmd
}

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

func (m model) View() string {
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
	return fmt.Sprintf("Commit Message testing: %s\n", m.commitMessage)
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
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		return
	}
}

func runAICommit(m model) (string, error) {
	gitDiff, err := getGitDiff()
	println("getting git diff")
	if err != nil {
		fmt.Println("Error fetching git diff:", err)
		return "", err
	}

	commitMsg, err := generateCommitMessageUsingAI(gitDiff, m.openAISecret)
	if err != nil {
		return "", err
	}
	return commitMsg, nil
}

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "head")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func generateCommitMessageUsingAI(gitDiff string, key string) (string, error) {
	println("generating commit message using AI")

	client := openai.NewClient(key)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "This is a commit message generator. Use conventional commits to describe your changes. ",
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
