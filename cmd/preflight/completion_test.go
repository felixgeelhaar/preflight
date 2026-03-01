package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletionCmd_GeneratesScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		shell := shell
		t.Run(shell, func(t *testing.T) {
			var err error
			captureStdout(t, func() {
				err = completionCmd.RunE(completionCmd, []string{shell})
			})
			require.NoError(t, err)
		})
	}
}
