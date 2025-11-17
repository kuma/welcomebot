package cache_test

import (
	"testing"

	"welcomebot/internal/core/cache"
)

func TestDefaultConfig(t *testing.T) {
	cfg := cache.DefaultConfig()

	if cfg.Addr != "localhost:6379" {
		t.Errorf("expected addr 'localhost:6379', got '%s'", cfg.Addr)
	}
	if cfg.DB != 0 {
		t.Errorf("expected db 0, got %d", cfg.DB)
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	cfg := cache.Config{
		Addr:     "invalid-host-that-does-not-exist:6379",
		Password: "",
		DB:       0,
	}

	_, err := cache.New(cfg)
	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}
}
