package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestNewInitWizardModel(t *testing.T) {
	t.Parallel()

	opts := InitWizardOptions{}
	model := newInitWizardModel(opts)

	assert.Equal(t, stepWelcome, model.step)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.False(t, model.cancelled)
}

func TestNewInitWizardModel_SkipWelcome(t *testing.T) {
	t.Parallel()

	opts := InitWizardOptions{SkipWelcome: true}
	model := newInitWizardModel(opts)

	assert.Equal(t, stepSelectProvider, model.step)
}

func TestInitWizardModel_Init(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestInitWizardModel_Update_WindowSize(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	updated, cmd := model.Update(msg)
	m := updated.(initWizardModel)

	assert.Nil(t, cmd)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestInitWizardModel_HandleKeyMsg_QuitFromWelcome(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}

	updated, cmd := model.Update(msg)
	m := updated.(initWizardModel)

	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd) // Should be tea.Quit
}

func TestInitWizardModel_HandleKeyMsg_EscFromWelcome(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	msg := tea.KeyMsg{Type: tea.KeyEsc}

	updated, cmd := model.Update(msg)
	m := updated.(initWizardModel)

	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd)
}

func TestInitWizardModel_HandleKeyMsg_EnterFromWelcome(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	msg := tea.KeyMsg{Type: tea.KeyEnter}

	updated, cmd := model.Update(msg)
	m := updated.(initWizardModel)

	assert.Nil(t, cmd)
	assert.Equal(t, stepSelectProvider, m.step)
}

func TestInitWizardModel_HandleKeyMsg_GoBack(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepSelectPreset

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := model.Update(msg)
	m := updated.(initWizardModel)

	assert.Equal(t, stepSelectProvider, m.step)
	assert.False(t, m.cancelled)
}

func TestInitWizardModel_HandleSelection_Provider(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepSelectProvider

	msg := components.ListSelectedMsg{
		Item: components.ListItem{ID: "nvim", Title: "Neovim"},
	}

	updated, _ := model.Update(msg)
	m := updated.(initWizardModel)

	assert.Equal(t, stepSelectPreset, m.step)
}

func TestInitWizardModel_HandleSelection_Preset(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepSelectPreset

	msg := components.ListSelectedMsg{
		Item: components.ListItem{ID: "nvim:balanced", Title: "Balanced"},
	}

	updated, _ := model.Update(msg)
	m := updated.(initWizardModel)

	assert.Equal(t, stepConfirm, m.step)
	assert.Equal(t, "nvim:balanced", m.selectedPreset)
}

func TestInitWizardModel_HandleConfirm_Yes(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepConfirm
	model.selectedPreset = "nvim:balanced"

	msg := components.ConfirmResultMsg{Confirmed: true}

	updated, _ := model.Update(msg)
	m := updated.(initWizardModel)

	// Now goes to preview step instead of complete
	assert.Equal(t, stepPreview, m.step)
	assert.Len(t, m.previewFiles, 2) // preflight.yaml and layers/base.yaml
}

func TestInitWizardModel_HandleConfirm_No(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepConfirm
	model.selectedPreset = "nvim:balanced"

	msg := components.ConfirmResultMsg{Confirmed: false}

	updated, _ := model.Update(msg)
	m := updated.(initWizardModel)

	assert.Equal(t, stepSelectPreset, m.step)
}

func TestInitWizardModel_GetPresetsForProvider(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})

	tests := []struct {
		provider string
		expected int
	}{
		{"nvim", 3},
		{"shell", 3},
		{"git", 2},
		{"brew", 2},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Parallel()
			presets := model.getPresetsForProvider(tt.provider)
			assert.Len(t, presets, tt.expected)
		})
	}
}

func TestInitWizardModel_View_Welcome(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepWelcome

	view := model.View()

	assert.Contains(t, view, "Welcome")
	assert.Contains(t, view, "Preflight")
}

func TestInitWizardModel_View_SelectProvider(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepSelectProvider

	view := model.View()

	assert.Contains(t, view, "Select a Provider")
}

func TestInitWizardModel_View_SelectPreset(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepSelectPreset

	view := model.View()

	assert.Contains(t, view, "Select a Preset")
}

func TestInitWizardModel_View_Confirm(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepConfirm
	model.selectedPreset = "nvim:balanced"

	view := model.View()

	assert.Contains(t, view, "Confirm")
	assert.Contains(t, view, "nvim:balanced")
}

func TestInitWizardModel_View_Complete(t *testing.T) {
	t.Parallel()

	model := newInitWizardModel(InitWizardOptions{})
	model.step = stepComplete

	view := model.View()

	assert.Contains(t, view, "Created")
}

func TestInitWizardModel_Update_ComponentForwarding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		step initWizardStep
	}{
		{"provider list", stepSelectProvider},
		{"preset list", stepSelectPreset},
		{"confirm", stepConfirm},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := newInitWizardModel(InitWizardOptions{})
			model.step = tt.step

			// Send a key that components can handle
			msg := tea.KeyMsg{Type: tea.KeyDown}
			updated, _ := model.Update(msg)

			// Just verify it doesn't panic
			assert.NotNil(t, updated)
		})
	}
}
