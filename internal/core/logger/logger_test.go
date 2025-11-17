package logger_test

import (
	"testing"

	"welcomebot/internal/core/logger"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  logger.Config
		wantErr bool
	}{
		{
			name: "valid json config",
			config: logger.Config{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "valid text config",
			config: logger.Config{
				Level:  "debug",
				Format: "text",
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: logger.Config{
				Level:  "invalid",
				Format: "json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := logger.New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogger_Methods(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test that methods don't panic
	log.Debug("debug message", "key", "value")
	log.Info("info message", "key1", "value1", "key2", "value2")
	log.Warn("warn message")
	log.Error("error message", "error", "test error")
}

func TestLogger_WithField(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test that WithField doesn't panic
	newLog := log.WithField("component", "test")
	newLog.Info("test message")
}

func TestLogger_WithFields(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test that WithFields doesn't panic
	fields := map[string]interface{}{
		"component": "test",
		"user_id":   "123",
	}
	newLog := log.WithFields(fields)
	newLog.Info("test message")
}

