package advisor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRecommendation_Valid(t *testing.T) {
	t.Parallel()

	rec, err := NewRecommendation(
		"nvim:balanced",
		"Use the balanced Neovim preset for a good starting point",
		ConfidenceHigh,
	)

	require.NoError(t, err)
	assert.Equal(t, "nvim:balanced", rec.PresetID())
	assert.Equal(t, "Use the balanced Neovim preset for a good starting point", rec.Rationale())
	assert.Equal(t, ConfidenceHigh, rec.Confidence())
}

func TestNewRecommendation_EmptyPresetID(t *testing.T) {
	t.Parallel()

	_, err := NewRecommendation("", "Some rationale", ConfidenceMedium)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyPresetID)
}

func TestNewRecommendation_EmptyRationale(t *testing.T) {
	t.Parallel()

	_, err := NewRecommendation("nvim:balanced", "", ConfidenceMedium)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyRationale)
}

func TestRecommendation_WithTradeoffs(t *testing.T) {
	t.Parallel()

	rec, _ := NewRecommendation("nvim:balanced", "Good balance", ConfidenceHigh)
	tradeoffs := []string{
		"More plugins mean slower startup",
		"Requires Node.js for some features",
	}

	updated := rec.WithTradeoffs(tradeoffs)

	assert.Empty(t, rec.Tradeoffs())
	assert.Equal(t, tradeoffs, updated.Tradeoffs())
}

func TestRecommendation_WithAlternatives(t *testing.T) {
	t.Parallel()

	rec, _ := NewRecommendation("nvim:balanced", "Good balance", ConfidenceHigh)
	alternatives := []string{"nvim:minimal", "nvim:pro"}

	updated := rec.WithAlternatives(alternatives)

	assert.Empty(t, rec.Alternatives())
	assert.Equal(t, alternatives, updated.Alternatives())
}

func TestRecommendation_WithDocLinks(t *testing.T) {
	t.Parallel()

	rec, _ := NewRecommendation("nvim:balanced", "Good balance", ConfidenceHigh)
	links := map[string]string{
		"Neovim": "https://neovim.io",
	}

	updated := rec.WithDocLinks(links)

	assert.Empty(t, rec.DocLinks())
	assert.Equal(t, links, updated.DocLinks())
}

func TestRecommendation_IsZero(t *testing.T) {
	t.Parallel()

	var zero Recommendation
	assert.True(t, zero.IsZero())

	nonZero, _ := NewRecommendation("nvim:balanced", "Good", ConfidenceHigh)
	assert.False(t, nonZero.IsZero())
}

func TestRecommendation_String(t *testing.T) {
	t.Parallel()

	rec, _ := NewRecommendation("nvim:balanced", "Good balance", ConfidenceHigh)

	assert.Equal(t, "nvim:balanced (high confidence)", rec.String())
}

func TestConfidenceLevel_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "low", ConfidenceLow.String())
	assert.Equal(t, "medium", ConfidenceMedium.String())
	assert.Equal(t, "high", ConfidenceHigh.String())
}

func TestConfidenceLevel_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, ConfidenceLow.IsValid())
	assert.True(t, ConfidenceMedium.IsValid())
	assert.True(t, ConfidenceHigh.IsValid())
	assert.False(t, ConfidenceLevel("unknown").IsValid())
}

func TestParseConfidenceLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected ConfidenceLevel
		wantErr  bool
	}{
		{"low", ConfidenceLow, false},
		{"medium", ConfidenceMedium, false},
		{"high", ConfidenceHigh, false},
		{"LOW", ConfidenceLow, false}, // case insensitive
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			level, err := ParseConfidenceLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}
