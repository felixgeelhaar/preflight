package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// initWizardStep represents the current step in the wizard.
type initWizardStep int

const (
	stepWelcome initWizardStep = iota
	stepSelectProvider
	stepSelectPreset
	stepConfirm
	stepComplete
)

// initWizardModel implements the init wizard TUI.
type initWizardModel struct {
	step           initWizardStep
	opts           InitWizardOptions
	styles         ui.Styles
	keys           ui.KeyMap
	width          int
	height         int
	configPath     string
	selectedPreset string
	cancelled      bool

	// Components
	providerList components.List
	presetList   components.List
	confirm      components.Confirm
}

func newInitWizardModel(opts InitWizardOptions) initWizardModel {
	styles := ui.DefaultStyles()
	keys := ui.DefaultKeyMap()

	// Initialize provider list
	providers := []components.ListItem{
		{ID: "nvim", Title: "Neovim", Description: "Terminal-based code editor"},
		{ID: "shell", Title: "Shell", Description: "Zsh/Bash configuration"},
		{ID: "git", Title: "Git", Description: "Version control configuration"},
		{ID: "brew", Title: "Homebrew", Description: "Package manager (macOS)"},
	}
	providerList := components.NewList(providers).
		WithWidth(60).
		WithHeight(10)

	// Initialize preset list (will be populated based on provider)
	presetList := components.NewList([]components.ListItem{}).
		WithWidth(60).
		WithHeight(10)

	// Initialize confirm dialog
	confirm := components.NewConfirm("Create configuration with these settings?").
		WithYesLabel("Create").
		WithNoLabel("Back")

	startStep := stepWelcome
	if opts.SkipWelcome {
		startStep = stepSelectProvider
	}

	return initWizardModel{
		step:         startStep,
		opts:         opts,
		styles:       styles,
		keys:         keys,
		width:        80,
		height:       24,
		providerList: providerList,
		presetList:   presetList,
		confirm:      confirm,
	}
}

func (m initWizardModel) Init() tea.Cmd {
	return nil
}

func (m initWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case components.ConfirmResultMsg:
		return m.handleConfirm(msg)
	}

	// Update active component
	var cmd tea.Cmd
	switch m.step {
	case stepWelcome, stepComplete:
		// No component to update
	case stepSelectProvider:
		m.providerList, cmd = m.providerList.Update(msg)
	case stepSelectPreset:
		m.presetList, cmd = m.presetList.Update(msg)
	case stepConfirm:
		m.confirm, cmd = m.confirm.Update(msg)
	}

	return m, cmd
}

func (m initWizardModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	//nolint:exhaustive // We only handle specific key types
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		if m.step == stepWelcome || m.step == stepSelectProvider {
			m.cancelled = true
			return m, tea.Quit
		}
		// Go back
		if m.step > stepWelcome {
			m.step--
		}
		return m, nil

	case tea.KeyEnter:
		if m.step == stepWelcome {
			m.step = stepSelectProvider
			return m, nil
		}
	}

	// Let component handle the key
	var cmd tea.Cmd
	switch m.step {
	case stepWelcome, stepComplete:
		// No component to update
	case stepSelectProvider:
		m.providerList, cmd = m.providerList.Update(msg)
	case stepSelectPreset:
		m.presetList, cmd = m.presetList.Update(msg)
	case stepConfirm:
		m.confirm, cmd = m.confirm.Update(msg)
	}

	return m, cmd
}

func (m initWizardModel) handleSelection(msg components.ListSelectedMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepWelcome, stepConfirm, stepComplete:
		// No selection handling in these steps
	case stepSelectProvider:
		// Provider selected, populate presets
		presets := m.getPresetsForProvider(msg.Item.ID)
		m.presetList = m.presetList.SetItems(presets)
		m.step = stepSelectPreset
		return m, nil

	case stepSelectPreset:
		m.selectedPreset = msg.Item.ID
		m.step = stepConfirm
		return m, nil
	}

	return m, nil
}

func (m initWizardModel) handleConfirm(msg components.ConfirmResultMsg) (tea.Model, tea.Cmd) {
	if msg.Confirmed {
		m.configPath = "preflight.yaml"
		m.step = stepComplete
		return m, tea.Quit
	}
	// Go back to preset selection
	m.step = stepSelectPreset
	return m, nil
}

func (m initWizardModel) getPresetsForProvider(provider string) []components.ListItem {
	switch provider {
	case "nvim":
		return []components.ListItem{
			{ID: "nvim:minimal", Title: "Minimal", Description: "Essential plugins only"},
			{ID: "nvim:balanced", Title: "Balanced", Description: "Recommended for most users"},
			{ID: "nvim:pro", Title: "Pro", Description: "Full IDE experience"},
		}
	case "shell":
		return []components.ListItem{
			{ID: "shell:zsh", Title: "Zsh", Description: "Basic Zsh configuration"},
			{ID: "shell:oh-my-zsh", Title: "Oh My Zsh", Description: "Popular Zsh framework"},
			{ID: "shell:starship", Title: "Starship", Description: "Cross-shell prompt"},
		}
	case "git":
		return []components.ListItem{
			{ID: "git:standard", Title: "Standard", Description: "Common git configuration"},
			{ID: "git:secure", Title: "Secure", Description: "GPG signing and security"},
		}
	case "brew":
		return []components.ListItem{
			{ID: "brew:minimal", Title: "Minimal", Description: "Essential formulae"},
			{ID: "brew:developer", Title: "Developer", Description: "Full developer setup"},
		}
	default:
		return []components.ListItem{}
	}
}

func (m initWizardModel) View() string {
	switch m.step {
	case stepWelcome:
		return m.viewWelcome()
	case stepSelectProvider:
		return m.viewSelectProvider()
	case stepSelectPreset:
		return m.viewSelectPreset()
	case stepConfirm:
		return m.viewConfirm()
	case stepComplete:
		return m.viewComplete()
	default:
		return ""
	}
}

func (m initWizardModel) viewWelcome() string {
	title := m.styles.Title.Render("Welcome to Preflight")
	subtitle := m.styles.Subtitle.Render("Deterministic workstation compiler")
	body := m.styles.Paragraph.Render(
		"Preflight helps you create a reproducible machine configuration.\n" +
			"This wizard will guide you through the initial setup.\n\n" +
			"Press Enter to continue or Esc to exit.",
	)
	return m.styles.App.Render(title + "\n" + subtitle + "\n\n" + body)
}

func (m initWizardModel) viewSelectProvider() string {
	title := m.styles.Title.Render("Select a Provider")
	subtitle := m.styles.Help.Render("Use ↑/↓ or j/k to navigate, Enter to select")
	return m.styles.App.Render(title + "\n" + subtitle + "\n\n" + m.providerList.View())
}

func (m initWizardModel) viewSelectPreset() string {
	title := m.styles.Title.Render("Select a Preset")
	subtitle := m.styles.Help.Render("Use ↑/↓ or j/k to navigate, Enter to select, Esc to go back")
	return m.styles.App.Render(title + "\n" + subtitle + "\n\n" + m.presetList.View())
}

func (m initWizardModel) viewConfirm() string {
	title := m.styles.Title.Render("Confirm Configuration")
	summary := m.styles.Paragraph.Render("Preset: " + m.selectedPreset)
	return m.styles.App.Render(title + "\n\n" + summary + "\n\n" + m.confirm.View())
}

func (m initWizardModel) viewComplete() string {
	title := m.styles.Title.Render("Configuration Created!")
	body := m.styles.Success.Render("Your preflight.yaml has been created.\n\n") +
		m.styles.Paragraph.Render("Next steps:\n") +
		m.styles.Help.Render("  preflight plan    - Review the execution plan\n") +
		m.styles.Help.Render("  preflight apply   - Apply the configuration\n")
	return m.styles.App.Render(title + "\n\n" + body)
}
