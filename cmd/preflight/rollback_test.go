package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRollbackCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "rollback", rollbackCmd.Use)
	assert.Equal(t, "Restore files from snapshots", rollbackCmd.Short)
}

func TestRollbackCommand_HasFlags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
	}{
		{"to_flag_exists", "to"},
		{"latest_flag_exists", "latest"},
		{"dry-run_flag_exists", "dry-run"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := rollbackCmd.Flags().Lookup(tt.flagName)
			assert.NotNil(t, flag, "flag %s should exist", tt.flagName)
		})
	}
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{"just_now", now.Add(-30 * time.Second), "just now"},
		{"1_min_ago", now.Add(-1 * time.Minute), "1 min ago"},
		{"5_mins_ago", now.Add(-5 * time.Minute), "5 mins ago"},
		{"1_hour_ago", now.Add(-1 * time.Hour), "1 hour ago"},
		{"3_hours_ago", now.Add(-3 * time.Hour), "3 hours ago"},
		{"1_day_ago", now.Add(-24 * time.Hour), "1 day ago"},
		{"3_days_ago", now.Add(-72 * time.Hour), "3 days ago"},
		{"1_week_ago", now.Add(-7 * 24 * time.Hour), "1 week ago"},
		{"2_weeks_ago", now.Add(-14 * 24 * time.Hour), "2 weeks ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
