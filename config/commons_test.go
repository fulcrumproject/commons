package config

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{
			name:     "silent level",
			input:    "silent",
			expected: slog.Level(99),
		},
		{
			name:     "error level",
			input:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "warn level",
			input:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "info level",
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty string defaults to info",
			input:    "",
			expected: slog.LevelInfo,
		},
		{
			name:     "invalid level defaults to info",
			input:    "invalid",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogConfig_GetLogLevel(t *testing.T) {
	config := LogConfig{Level: "error"}
	result := config.GetLogLevel()
	assert.Equal(t, slog.LevelError, result)
}

func TestDBConfig_GetLogLevel(t *testing.T) {
	config := DBConfig{LogLevel: "warn"}
	result := config.GetLogLevel()
	assert.Equal(t, slog.LevelWarn, result)
}
