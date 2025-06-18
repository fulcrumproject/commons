package logging

import (
	"log/slog"
	"os"

	"github.com/fulcrumproject/commons/config"
	slogGorm "github.com/orandin/slog-gorm"
	gormLogger "gorm.io/gorm/logger"
)

// NewGormLogger configures the logger based on the log format and level from config
func NewGormLogger(cfg *config.DB) gormLogger.Interface {
	var handler slog.Handler

	// Get log level from config
	level := cfg.GetLogLevel()

	// Configure the options with the log level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Configure the handler based on format
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slogGorm.New(
		slogGorm.WithHandler(handler),
		slogGorm.WithTraceAll(),
	)
}
