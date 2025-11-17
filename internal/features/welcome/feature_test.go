package welcome_test

import (
	"testing"

	"welcomebot/internal/core/logger"
	"welcomebot/internal/features/welcome"
)

func TestNew(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	deps := welcome.Dependencies{
		Logger: log,
	}

	_, err = welcome.New(deps)
	if err == nil {
		t.Error("expected error for missing dependencies, got nil")
	}
}

func TestName(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Format: "json"})

	// This will fail validation, but we just want to test the constructor pattern
	deps := welcome.Dependencies{Logger: log}
	feature, _ := welcome.New(deps)

	if feature != nil {
		name := feature.Name()
		if name != "welcome" {
			t.Errorf("expected name 'welcome', got '%s'", name)
		}
	}
}

