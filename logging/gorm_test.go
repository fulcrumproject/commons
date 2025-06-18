package logging

import (
	"log/slog"
	"testing"

	"github.com/fulcrumproject/commons/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormLogger "gorm.io/gorm/logger"
)

func TestNewGormLogger(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.DB
		expectNil bool
	}{
		{
			name: "json format with info level",
			cfg: &config.DB{
				DSN:       "test-dsn",
				LogFormat: "json",
				LogLevel:  "info",
			},
			expectNil: false,
		},
		{
			name: "text format with error level",
			cfg: &config.DB{
				DSN:       "test-dsn",
				LogFormat: "text",
				LogLevel:  "error",
			},
			expectNil: false,
		},
		{
			name: "text format with warn level",
			cfg: &config.DB{
				DSN:       "test-dsn",
				LogFormat: "text",
				LogLevel:  "warn",
			},
			expectNil: false,
		},
		{
			name: "text format with silent level",
			cfg: &config.DB{
				DSN:       "test-dsn",
				LogFormat: "text",
				LogLevel:  "silent",
			},
			expectNil: false,
		},
		{
			name: "default format (text) with empty level (defaults to info)",
			cfg: &config.DB{
				DSN:       "test-dsn",
				LogFormat: "",
				LogLevel:  "",
			},
			expectNil: false,
		},
		{
			name: "json format with empty level (defaults to info)",
			cfg: &config.DB{
				DSN:       "test-dsn",
				LogFormat: "json",
				LogLevel:  "",
			},
			expectNil: false,
		},
		{
			name: "unknown format defaults to text",
			cfg: &config.DB{
				DSN:       "test-dsn",
				LogFormat: "unknown",
				LogLevel:  "info",
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewGormLogger(tt.cfg)

			if tt.expectNil {
				assert.Nil(t, logger)
			} else {
				require.NotNil(t, logger)
				assert.Implements(t, (*gormLogger.Interface)(nil), logger)
			}
		})
	}
}

func TestNewGormLogger_LogLevelMapping(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		expectedLevel slog.Level
	}{
		{
			name:          "info level",
			logLevel:      "info",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "error level",
			logLevel:      "error",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "warn level",
			logLevel:      "warn",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "silent level",
			logLevel:      "silent",
			expectedLevel: slog.Level(99),
		},
		{
			name:          "empty level defaults to info",
			logLevel:      "",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "unknown level defaults to info",
			logLevel:      "unknown",
			expectedLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.DB{
				DSN:       "test-dsn",
				LogFormat: "text",
				LogLevel:  tt.logLevel,
			}

			// Test that GetLogLevel returns expected level
			actualLevel := cfg.GetLogLevel()
			assert.Equal(t, tt.expectedLevel, actualLevel)

			// Test that NewGormLogger doesn't panic with this config
			logger := NewGormLogger(cfg)
			require.NotNil(t, logger)
		})
	}
}
