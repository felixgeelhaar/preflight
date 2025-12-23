package ui_test

import (
	"errors"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/tui/ui"
	"github.com/stretchr/testify/assert"
)

func TestWindowSizeMsg(t *testing.T) {
	t.Parallel()

	msg := ui.WindowSizeMsg{Width: 80, Height: 24}
	assert.Equal(t, 80, msg.Width)
	assert.Equal(t, 24, msg.Height)
}

func TestErrorMsg_Error(t *testing.T) {
	t.Parallel()

	err := errors.New("test error")
	msg := ui.ErrorMsg{Err: err}
	assert.Equal(t, "test error", msg.Error())
}

func TestSuccessMsg(t *testing.T) {
	t.Parallel()

	msg := ui.SuccessMsg{Message: "operation completed"}
	assert.Equal(t, "operation completed", msg.Message)
}

func TestProgressMsg(t *testing.T) {
	t.Parallel()

	msg := ui.ProgressMsg{Current: 5, Total: 10, Message: "processing"}
	assert.Equal(t, 5, msg.Current)
	assert.Equal(t, 10, msg.Total)
	assert.Equal(t, "processing", msg.Message)
}

func TestCompletedMsg(t *testing.T) {
	t.Parallel()

	result := map[string]string{"status": "done"}
	msg := ui.CompletedMsg{Result: result}
	assert.Equal(t, result, msg.Result)
}

func TestSelectedMsg(t *testing.T) {
	t.Parallel()

	msg := ui.SelectedMsg{ID: "item-1", Index: 0, Value: "test"}
	assert.Equal(t, "item-1", msg.ID)
	assert.Equal(t, 0, msg.Index)
	assert.Equal(t, "test", msg.Value)
}

func TestConfirmedMsg(t *testing.T) {
	t.Parallel()

	msg := ui.ConfirmedMsg{Confirmed: true}
	assert.True(t, msg.Confirmed)

	msg2 := ui.ConfirmedMsg{Confirmed: false}
	assert.False(t, msg2.Confirmed)
}

func TestNavigateMsg(t *testing.T) {
	t.Parallel()

	params := map[string]interface{}{"id": "123"}
	msg := ui.NavigateMsg{View: "detail", Params: params}
	assert.Equal(t, "detail", msg.View)
	assert.Equal(t, params, msg.Params)
}

func TestRefreshMsg(t *testing.T) {
	t.Parallel()

	msg := ui.RefreshMsg{}
	assert.NotNil(t, msg)
}

func TestQuitMsg(t *testing.T) {
	t.Parallel()

	msg := ui.QuitMsg{}
	assert.NotNil(t, msg)
}

func TestNewErrorMsg(t *testing.T) {
	t.Parallel()

	err := errors.New("test error")
	msg := ui.NewErrorMsg(err)

	errMsg, ok := msg.(ui.ErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, err, errMsg.Err)
}

func TestNewSuccessMsg(t *testing.T) {
	t.Parallel()

	msg := ui.NewSuccessMsg("done")

	successMsg, ok := msg.(ui.SuccessMsg)
	assert.True(t, ok)
	assert.Equal(t, "done", successMsg.Message)
}

func TestNewProgressMsg(t *testing.T) {
	t.Parallel()

	msg := ui.NewProgressMsg(3, 10, "loading")

	progressMsg, ok := msg.(ui.ProgressMsg)
	assert.True(t, ok)
	assert.Equal(t, 3, progressMsg.Current)
	assert.Equal(t, 10, progressMsg.Total)
	assert.Equal(t, "loading", progressMsg.Message)
}

func TestNewCompletedMsg(t *testing.T) {
	t.Parallel()

	result := "finished"
	msg := ui.NewCompletedMsg(result)

	completedMsg, ok := msg.(ui.CompletedMsg)
	assert.True(t, ok)
	assert.Equal(t, result, completedMsg.Result)
}

func TestNewSelectedMsg(t *testing.T) {
	t.Parallel()

	msg := ui.NewSelectedMsg("item-1", 5, "value")

	selectedMsg, ok := msg.(ui.SelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, "item-1", selectedMsg.ID)
	assert.Equal(t, 5, selectedMsg.Index)
	assert.Equal(t, "value", selectedMsg.Value)
}

func TestNewNavigateMsg(t *testing.T) {
	t.Parallel()

	params := map[string]interface{}{"key": "value"}
	msg := ui.NewNavigateMsg("settings", params)

	navMsg, ok := msg.(ui.NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "settings", navMsg.View)
	assert.Equal(t, params, navMsg.Params)
}

func TestNewConfirmedMsg(t *testing.T) {
	t.Parallel()

	msg := ui.NewConfirmedMsg(true)

	confirmedMsg, ok := msg.(ui.ConfirmedMsg)
	assert.True(t, ok)
	assert.True(t, confirmedMsg.Confirmed)

	msg2 := ui.NewConfirmedMsg(false)
	confirmedMsg2, ok := msg2.(ui.ConfirmedMsg)
	assert.True(t, ok)
	assert.False(t, confirmedMsg2.Confirmed)
}
