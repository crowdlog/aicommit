package main

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// Define the model struct which includes the Bubbletea model elements
type model struct {
	openAIKeyInput textinput.Model
	gitDiff        string
	commitMessage  string
	errMsg         error
	hasKey         bool
}

// Implement the tea.Model interface for model
func (m model) Init() tea.Cmd {
	service := "crowdlog-aicommit"
	user := "anon"

	secret, err := keyring.Get(service, user)
	if err != nil {

	}

	println(secret, "angelo")

	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			println("Exiting...")
			return m, tea.Quit
		}
		if msg.String() == "c" {
			gitdiff, err := getGitDiff()
			println("getting git diff")
			if err != nil {
				println(gitdiff)
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

	return model{
		openAIKeyInput: ti,
		hasKey:         false,
	}
}

func (m model) View() string {
	tea.LogToFile("log.txt", "hello")
	// Implement the logic to render the view based on the model state
	if m.errMsg != nil {
		return fmt.Sprintf("Error: %s", m.errMsg.Error())
	}
	if !m.hasKey {
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
		Run:   runAICommit,
	}

	rootCmd.AddCommand(cmdAICommit)
	rootCmd.Execute()
}

func runAICommit(cmd *cobra.Command, args []string) {

	gitDiff, err := getGitDiff()
	println("getting git diff")
	if err != nil {
		fmt.Println("Error fetching git diff:", err)
		return
	}

	// Placeholder for OpenAI API call
	commitMsg, err := generateCommitMessageUsingAI(gitDiff)
	fmt.Printf("Commit Message: %s\n", commitMsg)
	if err != nil {
		fmt.Println("Error generating commit message:", err)
		return
	}

	p := tea.NewProgram(initialModel())
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

func generateCommitMessageUsingAI(gitDiff string) (string, error) {
	// client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	// resp, err := client.CreateChatCompletion(
	// 	context.Background(),
	// 	openai.ChatCompletionRequest{
	// 		Model: openai.GPT3Dot5Turbo,
	// 		Messages: []openai.ChatCompletionMessage{
	// 			{
	// 				Role:    openai.ChatMessageRoleSystem,
	// 				Content: "This is a commit message generator. Use conventional commits to describe your changes. ",
	// 			},
	// 		},
	// 	},
	// )
	// if err != nil {
	// 	fmt.Printf("Error: %s\n", err.Error())
	// 	return "", err
	// }

	// return resp.Choices[0].Message.Content, nil
	return "", nil
}
