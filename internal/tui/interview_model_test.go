package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

// mockAIProvider implements advisor.AIProvider for testing.
type mockAIProvider struct {
	available bool
	response  string
	err       error
}

func (m *mockAIProvider) Name() string {
	return "mock"
}

func (m *mockAIProvider) Available() bool {
	return m.available
}

func (m *mockAIProvider) Complete(_ context.Context, _ advisor.Prompt) (advisor.Response, error) {
	if m.err != nil {
		return advisor.Response{}, m.err
	}
	return advisor.NewResponse(m.response, 100, "mock-model"), nil
}

func TestNewInterviewModel(t *testing.T) {
	t.Parallel()

	provider := &mockAIProvider{available: true}
	model := newInterviewModel(context.Background(), provider)

	assert.Equal(t, interviewStepExperience, model.step)
	assert.Equal(t, provider, model.provider)
	assert.False(t, model.loading)
	assert.False(t, model.skipped)
	assert.False(t, model.cancelled)
}

func TestInterviewModel_Init(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestInterviewModel_WindowResize(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	m := newModel.(interviewModel)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestInterviewModel_EscapeSkips(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

	m := newModel.(interviewModel)
	assert.True(t, m.skipped)
	assert.NotNil(t, cmd)
}

func TestInterviewModel_CtrlCCancels(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	m := newModel.(interviewModel)
	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd)
}

func TestInterviewModel_ExperienceSelection(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)

	// Simulate selection
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(interviewModel)

	// Should still be on experience step until list sends selection
	assert.Equal(t, interviewStepExperience, m.step)
}

func TestInterviewModel_ViewExperience(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepExperience

	view := model.View()

	assert.Contains(t, view, "experience level")
}

func TestInterviewModel_ViewLanguages(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepLanguages

	view := model.View()

	assert.Contains(t, view, "programming language")
}

func TestInterviewModel_ViewTools(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepTools

	view := model.View()

	assert.Contains(t, view, "tools")
}

func TestInterviewModel_ViewGoals(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepGoals

	view := model.View()

	assert.Contains(t, view, "goal")
}

func TestInterviewModel_ViewLoading(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.loading = true

	view := model.View()

	assert.Contains(t, view, "Generating")
}

func TestInterviewModel_ViewSuggestion(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepSuggestion
	model.recommendation = &advisor.AIRecommendation{
		Presets:     []string{"developer"},
		Layers:      []string{"base", "role.go"},
		Explanation: "Great for Go development",
	}

	view := model.View()

	assert.Contains(t, view, "Recommendation")
	assert.Contains(t, view, "developer")
	assert.Contains(t, view, "role.go")
}

func TestInterviewModel_ViewSuggestion_NoRecommendation(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepSuggestion
	model.recommendation = nil

	view := model.View()

	assert.Contains(t, view, "No recommendation")
}

func TestInterviewModel_AIResponse(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.loading = true

	rec := &advisor.AIRecommendation{
		Presets: []string{"developer"},
		Layers:  []string{"base"},
	}

	newModel, _ := model.Update(aiResponseMsg{recommendation: rec})
	m := newModel.(interviewModel)

	assert.False(t, m.loading)
	assert.Equal(t, interviewStepSuggestion, m.step)
	assert.Equal(t, rec, m.recommendation)
}

func TestInterviewModel_AIResponseError(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.loading = true

	newModel, cmd := model.Update(aiResponseMsg{err: assert.AnError})
	m := newModel.(interviewModel)

	assert.False(t, m.loading)
	assert.True(t, m.skipped)
	assert.NotNil(t, cmd) // Should signal completion
}

func TestInterviewModel_Result(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	rec := &advisor.AIRecommendation{
		Presets: []string{"developer"},
	}
	model.recommendation = rec

	assert.Equal(t, rec, model.Result())
}

func TestInterviewModel_Skipped(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	assert.False(t, model.Skipped())

	model.skipped = true
	assert.True(t, model.Skipped())
}

func TestInterviewModel_Cancelled(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	assert.False(t, model.Cancelled())

	model.cancelled = true
	assert.True(t, model.Cancelled())
}

func TestInterviewModel_HandleSelection_Experience(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepExperience

	newModel, _ := model.handleSelection(components.ListSelectedMsg{
		Item: components.ListItem{ID: "intermediate", Title: "Intermediate"},
	})
	m := newModel.(interviewModel)

	assert.Equal(t, "intermediate", m.profile.ExperienceLevel)
	assert.Equal(t, interviewStepLanguages, m.step)
}

func TestInterviewModel_HandleSelection_Languages(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepLanguages

	newModel, _ := model.handleSelection(components.ListSelectedMsg{
		Item: components.ListItem{ID: "go", Title: "Go"},
	})
	m := newModel.(interviewModel)

	assert.Equal(t, "go", m.profile.PrimaryLanguage)
	assert.Contains(t, m.profile.Languages, "go")
	assert.Equal(t, interviewStepTools, m.step)
}

func TestInterviewModel_HandleSelection_Tools(t *testing.T) {
	t.Parallel()

	model := newInterviewModel(context.Background(), nil)
	model.step = interviewStepTools

	newModel, _ := model.handleSelection(components.ListSelectedMsg{
		Item: components.ListItem{ID: "neovim", Title: "Neovim"},
	})
	m := newModel.(interviewModel)

	assert.Contains(t, m.profile.Tools, "neovim")
	assert.Equal(t, interviewStepGoals, m.step)
}
