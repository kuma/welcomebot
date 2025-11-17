package logger

import (
	"github.com/sirupsen/logrus"
)

// Logger provides structured logging capabilities.
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// Config contains logger configuration.
type Config struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json", "text"
}

// DefaultConfig returns the default logger configuration.
func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "json",
	}
}

// logrusLogger wraps logrus.Logger to implement our Logger interface.
type logrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// New creates a new logger with the given configuration.
func New(cfg Config) (Logger, error) {
	log := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	log.SetLevel(level)

	// Set format
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{})
	}

	return &logrusLogger{
		logger: log,
		entry:  logrus.NewEntry(log),
	}, nil
}

// Debug logs a debug message with structured fields.
func (l *logrusLogger) Debug(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Debug(msg)
}

// Info logs an info message with structured fields.
func (l *logrusLogger) Info(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Info(msg)
}

// Warn logs a warning message with structured fields.
func (l *logrusLogger) Warn(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Warn(msg)
}

// Error logs an error message with structured fields.
func (l *logrusLogger) Error(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Error(msg)
}

// WithField returns a new logger with the added field.
func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields returns a new logger with the added fields.
func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &logrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
	}
}

// parseFields converts variadic key-value pairs to logrus.Fields.
func parseFields(fields ...interface{}) logrus.Fields {
	if len(fields) == 0 {
		return logrus.Fields{}
	}

	result := make(logrus.Fields)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			result[key] = fields[i+1]
		}
	}

	return result
}

