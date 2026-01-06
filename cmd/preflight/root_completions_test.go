package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootFlagCompletions(t *testing.T) {
	registerFlagCompletions()

	cases := []struct {
		flag      string
		want      []string
		directive cobra.ShellCompDirective
	}{
		{
			flag:      "config",
			want:      []string{"yaml", "yml"},
			directive: cobra.ShellCompDirectiveFilterFileExt,
		},
		{
			flag: "ai-provider",
			want: []string{
				"anthropic\tAnthropic Claude models",
				"gemini\tGoogle Gemini models",
				"openai\tOpenAI GPT models",
				"ollama\tLocal Ollama models",
			},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			flag: "mode",
			want: []string{
				"intent\tInstall latest compatible versions",
				"locked\tPrefer lockfile, update intentionally",
				"frozen\tFail if resolution differs from lock",
			},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.flag, func(t *testing.T) {
			fn, ok := rootCmd.GetFlagCompletionFunc(tc.flag)
			require.True(t, ok, "completion function for %s should exist", tc.flag)

			results, directive := fn(rootCmd, nil, "")
			assert.Equal(t, tc.directive, directive)
			assert.Equal(t, tc.want, results)
		})
	}
}
