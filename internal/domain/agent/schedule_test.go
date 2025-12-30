package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIntervalSchedule(t *testing.T) {
	schedule := NewIntervalSchedule(30 * time.Minute)

	assert.Equal(t, 30*time.Minute, schedule.Interval())
	assert.False(t, schedule.IsCron())
	assert.Empty(t, schedule.Cron())
	assert.Equal(t, "30m0s", schedule.String())
}

func TestNewCronSchedule(t *testing.T) {
	schedule := NewCronSchedule("0 */30 * * *")

	assert.True(t, schedule.IsCron())
	assert.Equal(t, "0 */30 * * *", schedule.Cron())
	assert.Equal(t, time.Hour, schedule.Interval()) // Default for cron
	assert.Equal(t, "cron(0 */30 * * *)", schedule.String())
}

func TestParseSchedule_Duration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30s", 30 * time.Second},
		{"5m", 5 * time.Minute},
		{"1h", time.Hour},
		{"2h30m", 2*time.Hour + 30*time.Minute},
		{"1d", 24 * time.Hour},
		{"2d12h", 2*24*time.Hour + 12*time.Hour},
		{"  30m  ", 30 * time.Minute}, // Whitespace trimmed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			schedule, err := ParseSchedule(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, schedule.Interval())
			assert.False(t, schedule.IsCron())
		})
	}
}

func TestParseSchedule_Cron(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"0 */30 * * *"},
		{"30 8 * * 1-5"},
		{"* * * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			schedule, err := ParseSchedule(tt.input)

			require.NoError(t, err)
			assert.True(t, schedule.IsCron())
			assert.Equal(t, tt.input, schedule.Cron())
		})
	}
}

func TestParseSchedule_Invalid(t *testing.T) {
	tests := []struct {
		input   string
		wantErr string
	}{
		{"", "empty schedule"},
		{"invalid", "invalid schedule format"},
		{"xyz123", "invalid schedule format"},
		{"0 * * *", "invalid schedule format"}, // Only 4 fields, not detected as cron
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseSchedule(tt.input)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSchedule_MarshalText(t *testing.T) {
	t.Run("interval", func(t *testing.T) {
		schedule := NewIntervalSchedule(30 * time.Minute)
		text, err := schedule.MarshalText()

		require.NoError(t, err)
		assert.Equal(t, "30m0s", string(text))
	})

	t.Run("cron", func(t *testing.T) {
		schedule := NewCronSchedule("0 */30 * * *")
		text, err := schedule.MarshalText()

		require.NoError(t, err)
		assert.Equal(t, "0 */30 * * *", string(text))
	})
}

func TestSchedule_UnmarshalText(t *testing.T) {
	t.Run("interval", func(t *testing.T) {
		var schedule Schedule
		err := schedule.UnmarshalText([]byte("1h30m"))

		require.NoError(t, err)
		assert.Equal(t, time.Hour+30*time.Minute, schedule.Interval())
	})

	t.Run("cron", func(t *testing.T) {
		var schedule Schedule
		err := schedule.UnmarshalText([]byte("0 */30 * * *"))

		require.NoError(t, err)
		assert.True(t, schedule.IsCron())
		assert.Equal(t, "0 */30 * * *", schedule.Cron())
	})
}

func TestSchedule_YAMLRoundTrip(t *testing.T) {
	t.Run("interval", func(t *testing.T) {
		original := NewIntervalSchedule(45 * time.Minute)

		// Marshal
		value, err := original.MarshalYAML()
		require.NoError(t, err)
		assert.Equal(t, "45m0s", value)

		// Unmarshal
		var parsed Schedule
		err = parsed.UnmarshalYAML(func(v interface{}) error {
			*(v.(*string)) = value.(string)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, original.Interval(), parsed.Interval())
	})

	t.Run("cron", func(t *testing.T) {
		original := NewCronSchedule("30 8 * * 1-5")

		// Marshal
		value, err := original.MarshalYAML()
		require.NoError(t, err)
		assert.Equal(t, "30 8 * * 1-5", value)

		// Unmarshal
		var parsed Schedule
		err = parsed.UnmarshalYAML(func(v interface{}) error {
			*(v.(*string)) = value.(string)
			return nil
		})
		require.NoError(t, err)
		assert.True(t, parsed.IsCron())
		assert.Equal(t, original.Cron(), parsed.Cron())
	})
}

func TestIsCronExpression(t *testing.T) {
	tests := []struct {
		input  string
		isCron bool
	}{
		{"0 * * * *", true},
		{"*/15 * * * *", true},
		{"30 8 1 1-6 *", true},
		{"30m", false},
		{"1h", false},
		{"invalid", false},
		{"* * *", false}, // Not enough fields
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.isCron, isCronExpression(tt.input))
		})
	}
}
