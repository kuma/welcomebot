package queue_test

import (
	"testing"

	"welcomebot/internal/core/queue"
)

func TestDefaultConfig(t *testing.T) {
	cfg := queue.DefaultConfig()

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("expected addr 'localhost:6379', got '%s'", cfg.RedisAddr)
	}
	if cfg.RedisDB != 0 {
		t.Errorf("expected db 0, got %d", cfg.RedisDB)
	}
	if cfg.QueueKey == "" {
		t.Error("expected non-empty queue key")
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	cfg := queue.Config{
		RedisAddr:     "invalid-host-that-does-not-exist:6379",
		RedisPassword: "",
		RedisDB:       0,
		QueueKey:      "test:queue",
	}

	_, err := queue.New(cfg)
	if err == nil {
		t.Error("expected error for invalid config, got nil")
	}
}

