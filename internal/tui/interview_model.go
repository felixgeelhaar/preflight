package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// interviewStep represents the current step in the interview.
type interviewStep int

const (
	interviewStepExperience interviewStep = iota
	interviewStepLanguages
	interviewStepTools
	interviewStepGoals
	interviewStepSuggestion
)

// interviewModel implements the AI interview TUI.
type interviewModel struct {
	ctx      context.Context
	step     interviewStep
	provider advisor.AIProvider
	styles   ui.Styles
	width    int
	height   int
	loading  bool
	err      error

	// Interview data
	profile advisor.InterviewProfile

	// Components
	experienceList components.List
	languageList   components.List
	toolList       components.List
	goalList       components.List

	// Results
	recommendation *advisor.AIRecommendation
	skipped        bool
	cancelled      bool
}

// interviewCompleteMsg signals that the interview is complete.
type interviewCompleteMsg struct {
	recommendation *advisor.AIRecommendation
	skipped        bool
}

// aiResponseMsg carries the AI response.
type aiResponseMsg struct {
	recommendation *advisor.AIRecommendation
	err            error
}

func newInterviewModel(ctx context.Context, provider advisor.AIProvider) interviewModel {
	styles := ui.DefaultStyles()

	// Experience level options
	experienceItems := []components.ListItem{
		{ID: "beginner", Title: "Beginner", Description: "New to development or this environment"},
		{ID: "intermediate", Title: "Intermediate", Description: "Comfortable with basics, learning advanced topics"},
		{ID: "advanced", Title: "Advanced", Description: "Experienced developer with specific preferences"},
	}
	experienceList := components.NewList(experienceItems).
		WithWidth(60).
		WithHeight(6)

	// Language options (multi-select style, but we'll handle it simply)
	languageItems := []components.ListItem{
		{ID: "go", Title: "Go", Description: "Systems programming and cloud services"},
		{ID: "python", Title: "Python", Description: "Data science, scripting, web development"},
		{ID: "javascript", Title: "JavaScript/TypeScript", Description: "Web and Node.js development"},
		{ID: "rust", Title: "Rust", Description: "Systems programming and performance"},
		{ID: "java", Title: "Java/Kotlin", Description: "Enterprise and Android development"},
		{ID: "other", Title: "Other", Description: "Ruby, C++, PHP, etc."},
	}
	languageList := components.NewList(languageItems).
		WithWidth(60).
		WithHeight(10)

	// Tool options
	toolItems := []components.ListItem{
		{ID: "neovim", Title: "Neovim/Vim", Description: "Terminal-based modal editor"},
		{ID: "vscode", Title: "VS Code", Description: "GUI editor with extensions"},
		{ID: "jetbrains", Title: "JetBrains IDEs", Description: "IntelliJ, PyCharm, GoLand, etc."},
		{ID: "terminal", Title: "Terminal-focused", Description: "CLI tools, tmux, command line"},
	}
	toolList := components.NewList(toolItems).
		WithWidth(60).
		WithHeight(8)

	// Goal options
	goalItems := []components.ListItem{
		{ID: "productivity", Title: "Productivity", Description: "Fast, efficient workflow"},
		{ID: "consistency", Title: "Consistency", Description: "Same setup across machines"},
		{ID: "learning", Title: "Learning", Description: "Explore new tools and workflows"},
		{ID: "minimal", Title: "Minimal", Description: "Simple, lightweight configuration"},
	}
	goalList := components.NewList(goalItems).
		WithWidth(60).
		WithHeight(8)

	return interviewModel{
		ctx:            ctx,
		step:           interviewStepExperience,
		provider:       provider,
		styles:         styles,
		width:          80,
		height:         24,
		experienceList: experienceList,
		languageList:   languageList,
		toolList:       toolList,
		goalList:       goalList,
	}
}

func (m interviewModel) Init() tea.Cmd {
	return nil
}

func (m interviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = m.styles.WithWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case components.ListSelectedMsg:
		return m.handleSelection(msg)

	case aiResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			// On error, skip AI suggestion
			m.skipped = true
			return m, func() tea.Msg {
				return interviewCompleteMsg{skipped: true}
			}
		}
		m.recommendation = msg.recommendation
		m.step = interviewStepSuggestion
		return m, nil
	}

	// Update active component
	var cmd tea.Cmd
	//nolint:exhaustive // Only update components in interactive steps
	switch m.step {
	case interviewStepExperience:
		m.experienceList, cmd = m.experienceList.Update(msg)
	case interviewStepLanguages:
		m.languageList, cmd = m.languageList.Update(msg)
	case interviewStepTools:
		m.toolList, cmd = m.toolList.Update(msg)
	case interviewStepGoals:
		m.goalList, cmd = m.goalList.Update(msg)
	}

	return m, cmd
}

func (m interviewModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	//nolint:exhaustive // We only handle specific key types
	switch msg.Type {
	case tea.KeyCtrlC:
		m.cancelled = true
		return m, tea.Quit

	case tea.KeyEsc:
		// Skip interview
		m.skipped = true
		return m, func() tea.Msg {
			return interviewCompleteMsg{skipped: true}
		}

	case tea.KeyEnter:
		if m.step == interviewStepSuggestion {
			// Accept suggestion
			return m, func() tea.Msg {
				return interviewCompleteMsg{recommendation: m.recommendation}
			}
		}
	}

	// Let component handle the key
	var cmd tea.Cmd
	//nolint:exhaustive // Only handle components in interactive steps
	switch m.step {
	case interviewStepExperience:
		m.experienceList, cmd = m.experienceList.Update(msg)
	case interviewStepLanguages:
		m.languageList, cmd = m.languageList.Update(msg)
	case interviewStepTools:
		m.toolList, cmd = m.toolList.Update(msg)
	case interviewStepGoals:
		m.goalList, cmd = m.goalList.Update(msg)
	}

	return m, cmd
}

func (m interviewModel) handleSelection(msg components.ListSelectedMsg) (tea.Model, tea.Cmd) {
	//nolint:exhaustive // Only handle selection in specific steps
	switch m.step {
	case interviewStepExperience:
		m.profile.ExperienceLevel = msg.Item.ID
		m.step = interviewStepLanguages
		return m, nil

	case interviewStepLanguages:
		m.profile.PrimaryLanguage = msg.Item.ID
		m.profile.Languages = []string{msg.Item.ID}
		m.step = interviewStepTools
		return m, nil

	case interviewStepTools:
		m.profile.Tools = []string{msg.Item.ID}
		m.step = interviewStepGoals
		return m, nil

	case interviewStepGoals:
		m.profile.Goals = []string{msg.Item.ID}
		// Start AI suggestion
		m.loading = true
		return m, m.requestSuggestion()
	}

	return m, nil
}

func (m interviewModel) requestSuggestion() tea.Cmd {
	return func() tea.Msg {
		if m.provider == nil || !m.provider.Available() {
			return aiResponseMsg{err: nil, recommendation: nil}
		}

		// Use parent context with 15-second timeout for AI requests
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
		defer cancel()

		prompt := advisor.BuildInterviewPrompt(m.profile)
		resp, err := m.provider.Complete(ctx, prompt)
		if err != nil {
			return aiResponseMsg{err: err}
		}

		rec, err := advisor.ParseRecommendations(resp.Content())
		if err != nil {
			return aiResponseMsg{err: err}
		}

		return aiResponseMsg{recommendation: rec}
	}
}

func (m interviewModel) View() string {
	if m.loading {
		return m.viewLoading()
	}

	switch m.step {
	case interviewStepExperience:
		return m.viewExperience()
	case interviewStepLanguages:
		return m.viewLanguages()
	case interviewStepTools:
		return m.viewTools()
	case interviewStepGoals:
		return m.viewGoals()
	case interviewStepSuggestion:
		return m.viewSuggestion()
	default:
		return ""
	}
}

func (m interviewModel) viewLoading() string {
	title := m.styles.Title.Render("Generating Recommendations...")
	body := m.styles.Paragraph.Render("Please wait while we analyze your preferences.")
	return m.styles.App.Render(title + "\n\n" + body)
}

func (m interviewModel) viewExperience() string {
	title := m.styles.Title.Render("What's your experience level?")
	subtitle := m.styles.Help.Render("Use arrow keys to select, Enter to confirm, Esc to skip")
	return m.styles.App.Render(title + "\n" + subtitle + "\n\n" + m.experienceList.View())
}

func (m interviewModel) viewLanguages() string {
	title := m.styles.Title.Render("What's your primary programming language?")
	subtitle := m.styles.Help.Render("Use arrow keys to select, Enter to confirm")
	return m.styles.App.Render(title + "\n" + subtitle + "\n\n" + m.languageList.View())
}

func (m interviewModel) viewTools() string {
	title := m.styles.Title.Render("What tools do you prefer?")
	subtitle := m.styles.Help.Render("Use arrow keys to select, Enter to confirm")
	return m.styles.App.Render(title + "\n" + subtitle + "\n\n" + m.toolList.View())
}

func (m interviewModel) viewGoals() string {
	title := m.styles.Title.Render("What's your main goal?")
	subtitle := m.styles.Help.Render("Use arrow keys to select, Enter to confirm")
	return m.styles.App.Render(title + "\n" + subtitle + "\n\n" + m.goalList.View())
}

func (m interviewModel) viewSuggestion() string {
	title := m.styles.Title.Render("AI Recommendation")

	if m.recommendation == nil {
		body := m.styles.Paragraph.Render("No recommendation available. Press Enter to continue with manual selection.")
		return m.styles.App.Render(title + "\n\n" + body)
	}

	var parts []string
	parts = append(parts, title)
	parts = append(parts, "")

	if len(m.recommendation.Presets) > 0 {
		presets := m.styles.Info.Render("Suggested presets: " + strings.Join(m.recommendation.Presets, ", "))
		parts = append(parts, presets)
	}

	if len(m.recommendation.Layers) > 0 {
		layers := m.styles.Info.Render("Suggested layers: " + strings.Join(m.recommendation.Layers, ", "))
		parts = append(parts, layers)
	}

	if m.recommendation.Explanation != "" {
		parts = append(parts, "")
		explanation := m.styles.Paragraph.Render(m.recommendation.Explanation)
		parts = append(parts, explanation)
	}

	parts = append(parts, "")
	help := m.styles.Help.Render("Press Enter to accept, Esc to skip")
	parts = append(parts, help)

	return m.styles.App.Render(strings.Join(parts, "\n"))
}

// Result returns the interview result.
func (m interviewModel) Result() *advisor.AIRecommendation {
	return m.recommendation
}

// Skipped returns true if the interview was skipped.
func (m interviewModel) Skipped() bool {
	return m.skipped
}

// Cancelled returns true if the interview was cancelled.
func (m interviewModel) Cancelled() bool {
	return m.cancelled
}
