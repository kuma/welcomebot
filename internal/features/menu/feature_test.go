package menu_test

import (
	"testing"

	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/features/menu"
)

func TestNew_MissingDependencies(t *testing.T) {
	deps := menu.Dependencies{}

	_, err := menu.New(deps)
	if err == nil {
		t.Error("expected error for missing dependencies, got nil")
	}
}

func TestName(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	tmpDir := t.TempDir()
	i18nSvc, _ := i18n.New(i18n.Dependencies{}, tmpDir)

	// Would need mock registry and init for full test
	deps := menu.Dependencies{
		Registry: nil,
		Init:     nil,
		I18n:     i18nSvc,
		Logger:   log,
	}

	_, err := menu.New(deps)
	if err == nil {
		t.Error("expected error without registry, got nil")
	}
}

