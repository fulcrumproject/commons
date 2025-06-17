package config

import "log/slog"

// Fulcrum Log configuration
type Log struct {
	Format string `json:"format" env:"LOG_FORMAT" validate:"omitempty,oneof=text json"`
	Level  string `json:"level" env:"LOG_LEVEL" validate:"omitempty,oneof=silent error warn info"`
}

// GetLogLevel converts a string log level to slog.Level
func (c *Log) GetLogLevel() slog.Level {
	return logLevel(c.Level)
}

// Fulcrum DB configuration
type DB struct {
	DSN       string `json:"dsn" env:"DB_DSN" validate:"required"`
	LogLevel  string `json:"logLevel" env:"DB_LOG_LEVEL" validate:"omitempty,oneof=silent error warn info"`
	LogFormat string `json:"logFormat" env:"DB_LOG_FORMAT" validate:"omitempty,oneof=text json"`
}

// GetLogLevel converts the string log level to gorm logger.LogLevel
func (c *DB) GetLogLevel() slog.Level {
	return logLevel(c.LogLevel)
}

func logLevel(value string) slog.Level {
	switch value {
	case "silent":
		return slog.Level(99) // Higher than any standard level
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info", "": // Default to info if empty
		return slog.LevelInfo
	default:
		return slog.LevelInfo // Default to info for unknown levels
	}
}
