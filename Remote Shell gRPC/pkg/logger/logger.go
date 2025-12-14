package logger

import (
	"io"
	"log/slog"
	"os"
)

// Level represents logging levels
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level  Level
	Format string // "json" or "text"
	Output io.Writer
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:  LevelInfo,
		Format: "text",
		Output: os.Stdout,
	}
}

// New creates a new Logger with the given configuration
func New(cfg Config) *Logger {
	var level slog.Level
	switch cfg.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default creates a logger with default configuration
func Default() *Logger {
	return New(DefaultConfig())
}

// WithComponent returns a new logger with a component field added
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
	}
}

// WithSessionID returns a new logger with a session_id field added
func (l *Logger) WithSessionID(sessionID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("session_id", sessionID),
	}
}

// WithClientID returns a new logger with a client_id field added
func (l *Logger) WithClientID(clientID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("client_id", clientID),
	}
}

// WithError returns a new logger with an error field added
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With("error", err.Error()),
	}
}
