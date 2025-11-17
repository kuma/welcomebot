package database_test

import (
	"testing"

	"welcomebot/internal/core/database"
)

func TestDefaultConfig(t *testing.T) {
	cfg := database.DefaultConfig()

	if cfg.Host != "localhost" {
		t.Errorf("expected host 'localhost', got '%s'", cfg.Host)
	}
	if cfg.Port != "5432" {
		t.Errorf("expected port '5432', got '%s'", cfg.Port)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("expected sslmode 'disable', got '%s'", cfg.SSLMode)
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	cfg := database.Config{
		Host:     "invalid-host-that-does-not-exist",
		Port:     "5432",
		User:     "test",
		Password: "test",
		Database: "test",
		SSLMode:  "disable",
	}

	_, err := database.New(cfg)
	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}
}

