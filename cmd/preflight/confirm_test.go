package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmBootstrap_AllowsEmptySteps(t *testing.T) {
	reset := setBootstrapFlags(t, false, false)
	defer reset()

	assert.True(t, confirmBootstrap(nil))
}

func TestConfirmBootstrap_YesFlagSkipsPrompt(t *testing.T) {
	reset := setBootstrapFlags(t, true, false)
	defer reset()

	assert.True(t, confirmBootstrap([]string{"a step"}))
}

func TestConfirmBootstrap_AllowBootstrapFlagSkipsPrompt(t *testing.T) {
	reset := setBootstrapFlags(t, false, true)
	defer reset()

	assert.True(t, confirmBootstrap([]string{"another step"}))
}

func TestConfirmBootstrap_Interactive(t *testing.T) {
	reset := setBootstrapFlags(t, false, false)
	defer reset()

	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	go func() {
		_, _ = writer.WriteString("YeS\n")
		_ = writer.Close()
	}()
	os.Stdin = reader

	assert.True(t, confirmBootstrap([]string{"final step"}))
}

func setBootstrapFlags(t *testing.T, yes, allow bool) func() {
	t.Helper()
	prevYes := yesFlag
	prevAllow := allowBootstrapFlag
	yesFlag = yes
	allowBootstrapFlag = allow
	return func() {
		yesFlag = prevYes
		allowBootstrapFlag = prevAllow
	}
}
