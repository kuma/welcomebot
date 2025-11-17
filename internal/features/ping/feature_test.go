package ping_test

import (
	"testing"

	"welcomebot/internal/core/logger"
	"welcomebot/internal/features/ping"
)

func TestNew(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	deps := ping.Dependencies{
		Logger: log,
	}

	feature, err := ping.New(deps)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if feature == nil {
		t.Error("expected feature, got nil")
	}
}

func TestNew_MissingDependency(t *testing.T) {
	deps := ping.Dependencies{}

	_, err := ping.New(deps)
	if err == nil {
		t.Error("expected error for missing dependencies, got nil")
	}
}

func TestName(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	feature, _ := ping.New(ping.Dependencies{Logger: log})

	name := feature.Name()
	if name != "ping" {
		t.Errorf("expected name 'ping', got '%s'", name)
	}
}

func TestRegisterCommands(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	feature, _ := ping.New(ping.Dependencies{Logger: log})

	commands := feature.RegisterCommands()
	if len(commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(commands))
	}

	if commands[0].Name != "ping" {
		t.Errorf("expected command name 'ping', got '%s'", commands[0].Name)
	}
}

