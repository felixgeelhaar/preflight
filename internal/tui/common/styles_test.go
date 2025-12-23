package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultStyles(t *testing.T) {
	t.Parallel()

	styles := DefaultStyles()

	// Verify styles are created and can render content
	assert.NotEmpty(t, styles.Title.Render("Test"))
	assert.NotEmpty(t, styles.Success.Render("Success"))
	assert.NotEmpty(t, styles.Error.Render("Error"))
}

func TestStyles_WithWidth(t *testing.T) {
	t.Parallel()

	styles := DefaultStyles()
	adapted := styles.WithWidth(80)

	// The adapted styles should have the width set
	assert.NotNil(t, adapted)
}

func TestDefaultKeyMap(t *testing.T) {
	t.Parallel()

	keys := DefaultKeyMap()

	assert.NotEmpty(t, keys.Up.Keys())
	assert.NotEmpty(t, keys.Down.Keys())
	assert.NotEmpty(t, keys.Select.Keys())
	assert.NotEmpty(t, keys.Quit.Keys())
}

func TestNewErrorMsg(t *testing.T) {
	t.Parallel()

	err := assert.AnError
	msg := NewErrorMsg(err)

	errMsg, ok := msg.(ErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, err, errMsg.Err)
}

func TestNewSuccessMsg(t *testing.T) {
	t.Parallel()

	msg := NewSuccessMsg("Operation completed")

	successMsg, ok := msg.(SuccessMsg)
	assert.True(t, ok)
	assert.Equal(t, "Operation completed", successMsg.Message)
}

func TestNewProgressMsg(t *testing.T) {
	t.Parallel()

	msg := NewProgressMsg(5, 10, "Processing")

	progressMsg, ok := msg.(ProgressMsg)
	assert.True(t, ok)
	assert.Equal(t, 5, progressMsg.Current)
	assert.Equal(t, 10, progressMsg.Total)
	assert.Equal(t, "Processing", progressMsg.Message)
}

func TestNewSelectedMsg(t *testing.T) {
	t.Parallel()

	msg := NewSelectedMsg("preset-1", 0, "nvim:balanced")

	selectedMsg, ok := msg.(SelectedMsg)
	assert.True(t, ok)
	assert.Equal(t, "preset-1", selectedMsg.ID)
	assert.Equal(t, 0, selectedMsg.Index)
}

func TestNewNavigateMsg(t *testing.T) {
	t.Parallel()

	params := map[string]interface{}{"category": "nvim"}
	msg := NewNavigateMsg("preset-select", params)

	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "preset-select", navMsg.View)
	assert.Equal(t, params, navMsg.Params)
}
