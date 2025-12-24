package fonts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		raw           map[string]interface{}
		wantNerdFonts []string
		wantErr       bool
	}{
		{
			name:          "empty config",
			raw:           map[string]interface{}{},
			wantNerdFonts: []string{},
		},
		{
			name: "nerd fonts list",
			raw: map[string]interface{}{
				"nerd_fonts": []interface{}{"JetBrainsMono", "FiraCode", "Hack"},
			},
			wantNerdFonts: []string{"JetBrainsMono", "FiraCode", "Hack"},
		},
		{
			name: "nerd fonts with nf suffix",
			raw: map[string]interface{}{
				"nerd_fonts": []interface{}{"JetBrainsMonoNF"},
			},
			wantNerdFonts: []string{"JetBrainsMonoNF"},
		},
		{
			name: "invalid nerd fonts type",
			raw: map[string]interface{}{
				"nerd_fonts": "not-a-list",
			},
			wantErr: true,
		},
		{
			name: "invalid nerd font entry",
			raw: map[string]interface{}{
				"nerd_fonts": []interface{}{123},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := ParseConfig(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantNerdFonts, cfg.NerdFonts)
		})
	}
}

func TestNerdFontCaskName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fontName     string
		wantCaskName string
	}{
		{
			name:         "simple font name",
			fontName:     "JetBrainsMono",
			wantCaskName: "font-jetbrains-mono-nerd-font",
		},
		{
			name:         "fira code",
			fontName:     "FiraCode",
			wantCaskName: "font-fira-code-nerd-font",
		},
		{
			name:         "hack",
			fontName:     "Hack",
			wantCaskName: "font-hack-nerd-font",
		},
		{
			name:         "meslo",
			fontName:     "Meslo",
			wantCaskName: "font-meslo-lg-nerd-font",
		},
		{
			name:         "source code pro",
			fontName:     "SourceCodePro",
			wantCaskName: "font-sauce-code-pro-nerd-font",
		},
		{
			name:         "ubuntu mono",
			fontName:     "UbuntuMono",
			wantCaskName: "font-ubuntu-mono-nerd-font",
		},
		{
			name:         "inconsolata",
			fontName:     "Inconsolata",
			wantCaskName: "font-inconsolata-nerd-font",
		},
		{
			name:         "cascadia code",
			fontName:     "CascadiaCode",
			wantCaskName: "font-caskaydia-cove-nerd-font",
		},
		{
			name:         "droid sans mono",
			fontName:     "DroidSansMono",
			wantCaskName: "font-droid-sans-mono-nerd-font",
		},
		{
			name:         "roboto mono",
			fontName:     "RobotoMono",
			wantCaskName: "font-roboto-mono-nerd-font",
		},
		{
			name:         "iosevka",
			fontName:     "Iosevka",
			wantCaskName: "font-iosevka-nerd-font",
		},
		{
			name:         "victor mono",
			fontName:     "VictorMono",
			wantCaskName: "font-victor-mono-nerd-font",
		},
		{
			name:         "already has nf suffix",
			fontName:     "JetBrainsMonoNF",
			wantCaskName: "font-jetbrains-mono-nerd-font",
		},
		{
			name:         "lowercase input",
			fontName:     "jetbrainsmono",
			wantCaskName: "font-jetbrains-mono-nerd-font",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			caskName := NerdFontCaskName(tt.fontName)
			assert.Equal(t, tt.wantCaskName, caskName)
		})
	}
}

func TestCaskFontsTap(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "homebrew/cask-fonts", CaskFontsTap)
}
