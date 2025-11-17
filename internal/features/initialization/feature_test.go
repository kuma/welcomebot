package initialization_test

import (
	"os"
	"testing"

	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/features/initialization"
)

func TestNew(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	i18nSvc, err := i18n.New(i18n.Dependencies{}, tmpDir)
	if err != nil {
		t.Fatalf("failed to create i18n: %v", err)
	}

	deps := initialization.Dependencies{
		I18n:   i18nSvc,
		Logger: log,
	}

	feature, err := initialization.New(deps)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if feature == nil {
		t.Error("expected feature, got nil")
	}
}

func TestName(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	i18nSvc, _ := i18n.New(i18n.Dependencies{}, tmpDir)

	feature, _ := initialization.New(initialization.Dependencies{
		I18n:   i18nSvc,
		Logger: log,
	})

	name := feature.Name()
	if name != "initialization" {
		t.Errorf("expected name 'initialization', got '%s'", name)
	}
}

func TestGetMenuButton(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	i18nSvc, _ := i18n.New(i18n.Dependencies{}, tmpDir)

	feature, _ := initialization.New(initialization.Dependencies{
		I18n:   i18nSvc,
		Logger: log,
	})

	btn := feature.GetMenuButton()
	if btn != nil {
		t.Error("expected nil menu button for initialization feature, got non-nil")
	}
}

