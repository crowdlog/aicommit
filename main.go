package main

// A simple example that shows how to send activity to Bubble Tea in real-time
// through a channel.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/phuslu/log"
	"github.com/spf13/cobra"
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
	NoView            ScreenView = -1
	SettingsView      ScreenView = 0 // 0
	CommitMessageView ScreenView = 1 // null, \0
)

type model struct {
	quitting bool
	view     ScreenView
	cdb      *CommitDB

	genMessageState struct {
		sub           chan string // where we'll receive activity notifications
		responses     int         // how many responses we've received
		loading       bool
		spinner       spinner.Model
		commitMessage *strings.Builder
	}

	settingsState struct {
		providerAPIKey    string
		hasProviderAPIKey bool
		userSettings      dbmodel.UserSettings
		form              *huh.Form
	}

	terminalWidth  int
	terminalHeight int
}

func getTeaProgram(db *CommitDB) *tea.Program {
	keyring.Delete("crowdlog-aicommit-openai", "anon")
	userSettings, err := db.GetUserSettings()
	if err != nil {
		panic(err)
	}
	hasProviderAPIKey := false
	providerAPIKey := ""

	if userSettings.AiProvider != nil {
		service := "crowdlog-aicommit-" + *userSettings.AiProvider
		user := "anon"
		key, apiKeyErr := keyring.Get(service, user)
		if apiKeyErr != nil {
			log.Info().Err(apiKeyErr)
			hasProviderAPIKey = false
		}
		if key != "" {
			hasProviderAPIKey = true
			providerAPIKey = key
		}
	}

	var view ScreenView
	if err := hasCompleteSettings(userSettings, hasProviderAPIKey); err == nil {
		log.Debug().Msg("User settings are complete")
		view = CommitMessageView
	} else {
		log.Debug().Msg("User settings are incomplete \n ")
		log.Debug().Msg(err.Error())
		view = SettingsView
	}
	form := NewSettingsForm(newSettingsFormArgs{addProviderKeyInput: !hasProviderAPIKey})

	return tea.NewProgram(model{
		cdb: db,
		settingsState: struct {
			providerAPIKey    string
			hasProviderAPIKey bool
			userSettings      dbmodel.UserSettings
			form              *huh.Form
		}{
			providerAPIKey:    providerAPIKey,
			hasProviderAPIKey: hasProviderAPIKey,
			userSettings:      userSettings,
			form:              form,
		},
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
		view: view,
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForActivity(m.genMessageState.sub), // wait for activity
		textinput.Blink,
		m.settingsState.form.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.view == SettingsView {
		var cmds []tea.Cmd
		form, cmd := m.settingsState.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.settingsState.form = f
			cmds = append(cmds, cmd)
		}

		if m.settingsState.form.State == huh.StateCompleted {
			println("Form is completed")
			err := m.SaveSettings()
			if err != nil {
				println("Error saving settings:", err.Error())
				return m, nil
			}
			m.view = CommitMessageView
			return m, tea.Batch(cmds...)
		}

		return m, tea.Batch(cmds...)
	}
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
				return m, tea.Batch(generateMessage(&m), m.genMessageState.spinner.Tick, tea.ClearScreen)
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

	if m.quitting {
		return "Quitting..."
	}

	if m.view == SettingsView {
		if m.settingsState.form.State == huh.StateCompleted {
			provider := m.settingsState.form.GetString("provider")
			model := m.settingsState.form.GetString("model")
			return fmt.Sprintf("Your AI provider is %s and your model is %s", provider, model)
		}
		return m.settingsState.form.View()
	}

	if m.view == CommitMessageView {
		commitMessage := m.genMessageState.commitMessage.String()
		s := mainContentStyle.Width(m.terminalWidth).Render(fmt.Sprintf("\n %s Events received: %d\n\n Press any key to exit\n %s", m.genMessageState.spinner.View(), m.genMessageState.responses, commitMessage))
		if m.quitting {
			s += "\n"
		}
		return s
	}
	return "Oops, something went wrong!"
}

func main() {
	initLogger()
	initialize := true
	cdb, err := getCommitDBFactory(initialize)
	if err != nil {
		println("Error downloading and installing SQLite:", err.Error())
		return
	}

	var cmdAICommit = &cobra.Command{
		Use:   "start",
		Short: "Generate commit message using AI",
		Run: func(cmd *cobra.Command, args []string) {
			p := getTeaProgram(cdb)
			if _, err := p.Run(); err != nil {
				fmt.Println("could not start program:", err)
				os.Exit(1)
			}
		},
	}
	cmdAICommit.Execute()
}

// ---------------- User Settings ----------------

type newSettingsFormArgs struct {
	addProviderKeyInput bool
}

func NewSettingsForm(args newSettingsFormArgs) *huh.Form {
	// Define the base groups
	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("provider").
				Options(huh.NewOptions("openai", "more coming soon...")...).
				Title("Choose your AI Provider").
				Validate(func(t string) error {
					if t != "openai" {
						return errors.New("only openai is supported at this time")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("model").
				Options(huh.NewOptions("gpt-4-1106-preview", "gpt-3.5-turbo-1106")...).
				Title("Choose your model"),
		),
	}

	// Conditionally add the provider key input field
	if args.addProviderKeyInput {
		providerKeyGroup := huh.NewGroup(
			huh.NewInput().Key("provider-key").Title("Please Provide your OpenAI API Key"),
		)
		groups = append(groups, providerKeyGroup)
	}

	// Create the form with the groups
	return huh.NewForm(groups...)
}

func hasCompleteSettings(userSettings dbmodel.UserSettings, hasProviderAPIKey bool) error {
	var validProviders = []string{"openai"}
	if userSettings.AiProvider == nil {
		return errors.New("no AI provider selected")
	}
	if userSettings.AiProvider != nil && !StringInSlice(*userSettings.AiProvider, validProviders) {
		return errors.New("invalid AI provider selected")
	}
	if !hasProviderAPIKey {
		return errors.New("no API key provided")
	}
	return nil
}

func (m *model) SaveSettings() error {
	if m.settingsState.form.State != huh.StateCompleted {
		return fmt.Errorf("form is not completed")
	}
	provider := m.settingsState.form.GetString("provider")
	model := m.settingsState.form.GetString("model")
	providerKey := m.settingsState.form.GetString("provider-key")
	service := "crowdlog-aicommit-" + provider
	user := "anon"
	err := keyring.Set(service, user, providerKey)
	if err != nil {
		return err
	}
	userSettings := dbmodel.UserSettings{
		AiProvider:     &provider,
		ModelSelection: &model,
	}
	m.settingsState.providerAPIKey = providerKey
	_, err = m.cdb.UpdateUserSettings(userSettings)
	if err != nil {
		return err
	}
	newUserSettings, err := m.cdb.GetUserSettings()
	if err != nil {
		return err
	}
	m.settingsState.userSettings = newUserSettings
	return nil
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

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "-U10", "head")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

type genMsg struct {
	Content string
	msgType string
}

func generateMessage(m *model) tea.Cmd {

	return func() tea.Msg {
		gitDiff, err := getGitDiff()
		if err != nil {
			println("Error getting git diff:", err.Error())
			return nil
		}

		dateCreated := time.Now()
		diffStructuredJson := ""
		model := m.settingsState.userSettings.ModelSelection
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
			Model:              model,
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
	llm, err := openai.NewChat(openai.WithModel(*m.settingsState.userSettings.ModelSelection), openai.WithToken(m.settingsState.providerAPIKey))
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
