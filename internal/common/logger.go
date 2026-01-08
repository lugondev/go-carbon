package common

import "log/slog"

// Loggable interface for types that support custom logging.
type Loggable interface {
	SetLogger(logger *slog.Logger)
	GetLogger() *slog.Logger
}

// LoggerMixin provides common logging functionality.
type LoggerMixin struct {
	Logger *slog.Logger
}

// NewLoggerMixin creates a new logger mixin with default logger.
func NewLoggerMixin() LoggerMixin {
	return LoggerMixin{
		Logger: slog.Default(),
	}
}

// SetLogger sets a custom logger.
func (l *LoggerMixin) SetLogger(logger *slog.Logger) {
	if logger != nil {
		l.Logger = logger
	}
}

// GetLogger returns the logger.
func (l *LoggerMixin) GetLogger() *slog.Logger {
	if l.Logger == nil {
		l.Logger = slog.Default()
	}
	return l.Logger
}

// WithLoggerBuilder provides a fluent interface for setting loggers.
type WithLoggerBuilder[T any] interface {
	WithLogger(logger *slog.Logger) T
}
