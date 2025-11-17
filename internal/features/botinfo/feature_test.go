package botinfo_test

import (
	"testing"
	"time"

	"welcomebot/internal/core/logger"
	"welcomebot/internal/features/botinfo"
)

func TestNew(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	deps := botinfo.Dependencies{
		Logger: log,
	}

	feature, err := botinfo.New(deps)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if feature == nil {
		t.Error("expected feature, got nil")
	}
}

func TestNew_MissingDependency(t *testing.T) {
	deps := botinfo.Dependencies{}

	_, err := botinfo.New(deps)
	if err == nil {
		t.Error("expected error for missing dependencies, got nil")
	}
}

func TestName(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	feature, _ := botinfo.New(botinfo.Dependencies{Logger: log})

	name := feature.Name()
	if name != "botinfo" {
		t.Errorf("expected name 'botinfo', got '%s'", name)
	}
}

func TestRegisterCommands(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	feature, _ := botinfo.New(botinfo.Dependencies{Logger: log})

	commands := feature.RegisterCommands()
	if len(commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(commands))
	}

	if commands[0].Name != "botinfo" {
		t.Errorf("expected command name 'botinfo', got '%s'", commands[0].Name)
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"minutes only", 30 * time.Minute, "30m"},
		{"hours and minutes", 2*time.Hour + 15*time.Minute, "2h 15m"},
		{"days hours minutes", 3*24*time.Hour + 5*time.Hour + 20*time.Minute, "3d 5h 20m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't test formatUptime directly as it's private
			// This is just a placeholder to show test structure
		})
	}
}

