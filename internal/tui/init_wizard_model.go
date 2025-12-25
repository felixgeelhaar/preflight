package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// initWizardStep represents the current step in the wizard.
type initWizardStep int

const (
	stepWelcome initWizardStep = iota
	stepInterview
	stepSelectProvider
	stepSelectPreset
	stepConfirm
	stepPreview
	stepComplete
)

// initWizardModel implements the init wizard TUI.
type initWizardModel struct {
	step             initWizardStep
	opts             InitWizardOptions
	styles           ui.Styles
	keys             ui.KeyMap
	width            int
	height           int
	configPath       string
	selectedProvider string
	selectedPreset   string
	cancelled        bool
	catalogService   CatalogServiceInterface
	aiProvider       advisor.AIProvider

	// Components
	providerList components.List
	presetList   components.List
	confirm      components.Confirm
	interview    interviewModel

	// AI recommendation results
	aiRecommendation *advisor.AIRecommendation

	// Preview state
	previewFiles []PreviewFile
	previewModel layerPreviewModel
}

func newInitWizardModel(opts InitWizardOptions) initWizardModel {
	styles := ui.DefaultStyles()
	keys := ui.DefaultKeyMap()

	// Use catalog service if provided, otherwise use fallback
	catalogService := opts.CatalogService
	if catalogService == nil {
		catalogService = &fallbackCatalogService{}
	}

	// Initialize provider list from catalog
	providerNames := catalogService.GetProviders()
	providers := make([]components.ListItem, 0, len(providerNames))
	for _, name := range providerNames {
		// Add installation status indicator to title for editors
		title := providerDisplayName(name) + providerInstalledStatus(name)
		providers = append(providers, components.ListItem{
			ID:          name,
			Title:       title,
			Description: providerDescription(name),
		})
	}

	// If no providers from catalog, use fallback defaults with all editors
	if len(providers) == 0 {
		// Get all editors with installation status
		for _, editor := range AvailableEditors() {
			title := editor.Name
			if editor.IsInstalled {
				title += " ✓"
			}
			providers = append(providers, components.ListItem{
				ID:          editor.ID,
				Title:       title,
				Description: editor.Description,
			})
		}
		// Add other core providers
		providers = append(providers,
			components.ListItem{ID: "shell", Title: "Shell", Description: "Zsh/Bash configuration"},
			components.ListItem{ID: "git", Title: "Git", Description: "Version control configuration"},
			components.ListItem{ID: "brew", Title: "Homebrew", Description: "Package manager (macOS)"},
		)
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

	// Initialize interview model if advisor is available
	var interview interviewModel
	if opts.Advisor != nil && opts.Advisor.Available() {
		interview = newInterviewModel(opts.Advisor)
	}

	// Determine starting step
	startStep := stepWelcome
	if opts.SkipWelcome {
		// If we have an advisor and not skipping interview, go to interview
		if opts.Advisor != nil && opts.Advisor.Available() && !opts.SkipInterview {
			startStep = stepInterview
		} else {
			startStep = stepSelectProvider
		}
	}

	return initWizardModel{
		step:           startStep,
		opts:           opts,
		styles:         styles,
		keys:           keys,
		width:          80,
		height:         24,
		catalogService: catalogService,
		aiProvider:     opts.Advisor,
		providerList:   providerList,
		presetList:     presetList,
		confirm:        confirm,
		interview:      interview,
	}
}

// providerDisplayName returns a human-readable name for a provider.
func providerDisplayName(id string) string {
	// Check if it's an editor first
	if editor, ok := GetEditorByID(id); ok {
		return editor.Name
	}

	names := map[string]string{
		"shell":   "Shell",
		"git":     "Git",
		"brew":    "Homebrew",
		"apt":     "APT",
		"files":   "Files",
		"ssh":     "SSH",
		"runtime": "Runtime",
		"docker":  "Docker",
		"fonts":   "Fonts",
	}
	if name, ok := names[id]; ok {
		return name
	}
	return id
}

// providerDescription returns a description for a provider.
func providerDescription(id string) string {
	// Check if it's an editor first
	if editor, ok := GetEditorByID(id); ok {
		return editor.Description
	}

	descriptions := map[string]string{
		"shell":   "Zsh/Bash configuration",
		"git":     "Version control configuration",
		"brew":    "Package manager (macOS)",
		"apt":     "Package manager (Linux)",
		"files":   "Dotfile management",
		"ssh":     "SSH configuration",
		"runtime": "Runtime version management",
		"docker":  "Container platform",
		"fonts":   "Nerd Font installation",
	}
	if desc, ok := descriptions[id]; ok {
		return desc
	}
	return "Configuration for " + id
}

// providerInstalledStatus returns an installation status indicator.
func providerInstalledStatus(id string) string {
	if editor, ok := GetEditorByID(id); ok {
		if editor.IsInstalled {
			return " ✓"
		}
		return ""
	}
	return ""
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

	case interviewCompleteMsg:
		return m.handleInterviewComplete(msg)
	}

	// Update active component
	var cmd tea.Cmd
	switch m.step {
	case stepWelcome, stepComplete:
		// No component to update
	case stepInterview:
		var newInterview tea.Model
		newInterview, cmd = m.interview.Update(msg)
		m.interview = newInterview.(interviewModel)
	case stepSelectProvider:
		m.providerList, cmd = m.providerList.Update(msg)
	case stepSelectPreset:
		m.presetList, cmd = m.presetList.Update(msg)
	case stepConfirm:
		m.confirm, cmd = m.confirm.Update(msg)
	case stepPreview:
		var newPreview tea.Model
		newPreview, cmd = m.previewModel.Update(msg)
		m.previewModel = newPreview.(layerPreviewModel)

		// Check if preview was confirmed or cancelled
		if m.previewModel.confirmed {
			return m.writeConfigFiles()
		}
		if m.previewModel.cancelled {
			// Go back to confirm step
			m.step = stepConfirm
			return m, nil
		}
	}

	return m, cmd
}

func (m initWizardModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Let preview handle its own keys
	if m.step == stepPreview {
		return m, nil // Keys are handled in Update's stepPreview case
	}

	//nolint:exhaustive // We only handle specific key types
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		if m.step == stepWelcome || m.step == stepSelectProvider || m.step == stepInterview {
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
			// Transition to interview if advisor available, otherwise to provider selection
			if m.aiProvider != nil && m.aiProvider.Available() && !m.opts.SkipInterview {
				m.step = stepInterview
			} else {
				m.step = stepSelectProvider
			}
			return m, nil
		}
	}

	// Let component handle the key
	var cmd tea.Cmd
	//nolint:exhaustive // stepPreview is handled separately above
	switch m.step {
	case stepWelcome, stepComplete:
		// No component to update
	case stepInterview:
		var newInterview tea.Model
		newInterview, cmd = m.interview.Update(msg)
		m.interview = newInterview.(interviewModel)
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
	//nolint:exhaustive // We only handle selection in specific steps
	switch m.step {
	case stepWelcome, stepConfirm, stepComplete, stepInterview:
		// No selection handling in these steps
	case stepSelectProvider:
		// Provider selected, populate presets
		m.selectedProvider = msg.Item.ID
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
		// Generate preview files (without writing to disk)
		targetDir := "."
		if m.opts.TargetDir != "" {
			targetDir = m.opts.TargetDir
		}

		generator := NewConfigGenerator(targetDir)

		// Get preset details from catalog
		preset, found := m.catalogService.GetPreset(m.selectedPreset)
		if !found {
			// Fallback to basic preset info
			preset = PresetItem{
				ID:    m.selectedPreset,
				Title: m.selectedPreset,
			}
		}

		// Generate preview files
		previewFiles, err := generator.GeneratePreviewFiles(preset)
		if err != nil {
			// TODO: Show error in TUI
			m.step = stepComplete
			return m, tea.Quit
		}

		m.previewFiles = previewFiles
		m.previewModel = newLayerPreviewModel(previewFiles, LayerPreviewOptions{
			Title:        "Preview Configuration",
			ShowLineNums: true,
		})
		m.step = stepPreview
		return m, nil
	}
	// Go back to preset selection
	m.step = stepSelectPreset
	return m, nil
}

// writeConfigFiles writes the preview files to disk.
func (m initWizardModel) writeConfigFiles() (tea.Model, tea.Cmd) {
	targetDir := "."
	if m.opts.TargetDir != "" {
		targetDir = m.opts.TargetDir
	}

	generator := NewConfigGenerator(targetDir)

	// Get preset details from catalog
	preset, found := m.catalogService.GetPreset(m.selectedPreset)
	if !found {
		preset = PresetItem{
			ID:    m.selectedPreset,
			Title: m.selectedPreset,
		}
	}

	if err := generator.GenerateFromPreset(preset); err != nil {
		// TODO: Show error in TUI
		m.step = stepComplete
		return m, tea.Quit
	}

	m.configPath = "preflight.yaml"
	m.step = stepComplete
	return m, tea.Quit
}

func (m initWizardModel) handleInterviewComplete(msg interviewCompleteMsg) (tea.Model, tea.Cmd) {
	if m.interview.Cancelled() {
		m.cancelled = true
		return m, tea.Quit
	}

	// Store AI recommendation if available
	if msg.recommendation != nil {
		m.aiRecommendation = msg.recommendation

		// If we have preset recommendations, we could pre-select them
		// For now, just move to provider selection
	}

	// Move to provider selection
	m.step = stepSelectProvider
	return m, nil
}

func (m initWizardModel) getPresetsForProvider(provider string) []components.ListItem {
	// Use catalog service to get presets
	presetItems := m.catalogService.GetPresetsForProvider(provider)
	if len(presetItems) > 0 {
		items := make([]components.ListItem, len(presetItems))
		for i, p := range presetItems {
			items[i] = components.ListItem{
				ID:          p.ID,
				Title:       p.Title,
				Description: p.Description,
			}
		}
		return items
	}

	// Fallback to hardcoded presets if catalog returns nothing
	switch provider {
	// Editors with full provider support
	case "neovim", "nvim":
		return []components.ListItem{
			{ID: "nvim:minimal", Title: "Minimal", Description: "Essential plugins only"},
			{ID: "nvim:balanced", Title: "Balanced", Description: "Good balance of features"},
			{ID: "nvim:pro", Title: "Pro", Description: "Full IDE experience"},
		}
	case "vscode":
		return []components.ListItem{
			{ID: "vscode:minimal", Title: "Minimal", Description: "Essential extensions only"},
			{ID: "vscode:full", Title: "Full", Description: "Feature-rich configuration"},
		}
	case "cursor":
		return []components.ListItem{
			{ID: "vscode:minimal", Title: "Minimal", Description: "Essential extensions only"},
			{ID: "vscode:full", Title: "Full", Description: "Feature-rich configuration"},
		}
	case "vscodium":
		return []components.ListItem{
			{ID: "vscode:minimal", Title: "Minimal", Description: "Essential extensions only"},
			{ID: "vscode:full", Title: "Full", Description: "Feature-rich configuration"},
		}

	// Editors without full provider support (basic presets)
	case "vim":
		return []components.ListItem{
			{ID: "vim:basic", Title: "Basic", Description: "Sensible defaults"},
		}
	case "emacs":
		return []components.ListItem{
			{ID: "emacs:basic", Title: "Basic", Description: "Sensible defaults"},
		}
	case "helix":
		return []components.ListItem{
			{ID: "helix:basic", Title: "Basic", Description: "Default configuration"},
		}
	case "zed":
		return []components.ListItem{
			{ID: "zed:basic", Title: "Basic", Description: "Default configuration"},
		}
	case "sublime":
		return []components.ListItem{
			{ID: "sublime:basic", Title: "Basic", Description: "Default configuration"},
		}
	case "nano":
		return []components.ListItem{
			{ID: "nano:basic", Title: "Basic", Description: "Improved defaults"},
		}
	case "intellij":
		return []components.ListItem{
			{ID: "intellij:basic", Title: "Basic", Description: "Default configuration"},
		}
	case "webstorm":
		return []components.ListItem{
			{ID: "webstorm:basic", Title: "Basic", Description: "Default configuration"},
		}

	// Other providers
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
			{ID: "brew:cli-essentials", Title: "CLI Essentials", Description: "Must-have CLI tools"},
			{ID: "brew:dev-tools", Title: "Developer Tools", Description: "Common dev utilities"},
		}
	case "docker":
		return []components.ListItem{
			{ID: "docker:basic", Title: "Basic", Description: "Docker Desktop setup"},
			{ID: "docker:kubernetes", Title: "Kubernetes", Description: "With local Kubernetes"},
		}
	case "fonts":
		return []components.ListItem{
			{ID: "fonts:nerd-essential", Title: "Essential", Description: "Core Nerd Fonts"},
			{ID: "fonts:nerd-complete", Title: "Complete", Description: "All Nerd Fonts"},
		}
	case "runtime":
		return []components.ListItem{
			{ID: "runtime:mise-node", Title: "Node.js", Description: "Node.js via mise"},
			{ID: "runtime:mise-polyglot", Title: "Polyglot", Description: "Multiple languages"},
		}
	case "ssh":
		return []components.ListItem{
			{ID: "ssh:basic", Title: "Basic", Description: "Essential SSH settings"},
			{ID: "ssh:github", Title: "GitHub", Description: "Optimized for GitHub"},
			{ID: "ssh:multi-identity", Title: "Multi-Identity", Description: "Work/personal separation"},
		}
	default:
		return []components.ListItem{}
	}
}

// fallbackCatalogService provides hardcoded fallback data when no catalog is available.
type fallbackCatalogService struct{}

func (f *fallbackCatalogService) GetProviders() []string {
	// Build provider list with editors first (alphabetically), then other tools
	providers := make([]string, 0, 20)

	// Add all available editors (already sorted alphabetically)
	for _, editor := range AvailableEditors() {
		providers = append(providers, editor.ID)
	}

	// Add other core providers (alphabetically)
	providers = append(providers,
		"brew",
		"docker",
		"fonts",
		"git",
		"runtime",
		"shell",
		"ssh",
	)

	return providers
}

func (f *fallbackCatalogService) GetPresetsForProvider(_ string) []PresetItem {
	// Return empty to trigger hardcoded fallback in getPresetsForProvider
	return nil
}

func (f *fallbackCatalogService) GetCapabilityPacks() []PackItem {
	return nil
}

func (f *fallbackCatalogService) GetPreset(_ string) (PresetItem, bool) {
	return PresetItem{}, false
}

func (m initWizardModel) View() string {
	switch m.step {
	case stepWelcome:
		return m.viewWelcome()
	case stepInterview:
		return m.interview.View()
	case stepSelectProvider:
		return m.viewSelectProvider()
	case stepSelectPreset:
		return m.viewSelectPreset()
	case stepConfirm:
		return m.viewConfirm()
	case stepPreview:
		return m.previewModel.View()
	case stepComplete:
		return m.viewComplete()
	default:
		return ""
	}
}

func (m initWizardModel) viewWelcome() string {
	// ASCII art logo - Stylized "P" with verification badge
	logoLines := []string{
		"  ██████╗ ",
		"  ██╔══██╗",
		"  ██████╔╝",
		"  ██╔═══╝    ✓",
		"  ██║     ",
		"  ╚═╝     ",
	}

	// Render logo with brand colors
	var logo string
	for i, line := range logoLines {
		if i == 3 {
			// Line with checkmark badge - split P and badge
			pPart := line[:12]
			badge := line[12:]
			logo += m.styles.Logo.Render(pPart) + m.styles.LogoBadge.Render(badge) + "\n"
		} else {
			logo += m.styles.Logo.Render(line) + "\n"
		}
	}

	title := m.styles.Title.Render("Welcome to Preflight")
	subtitle := m.styles.Subtitle.Render("Deterministic workstation compiler")
	body := m.styles.Paragraph.Render(
		"Preflight helps you create a reproducible machine configuration.\n" +
			"This wizard will guide you through the initial setup.\n\n" +
			"Press Enter to continue or Esc to exit.",
	)
	return m.styles.App.Render(logo + "\n" + title + "\n" + subtitle + "\n\n" + body)
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
