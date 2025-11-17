package language_test

import (
	"os"
	"testing"

	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/features/language"
)

func TestNew(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Create mock i18n (would need proper implementation for full test)
	deps := language.Dependencies{
		I18n:   nil, // Would need mock
		Logger: log,
	}

	_, err = language.New(deps)
	if err == nil {
		t.Error("expected error for nil i18n, got nil")
	}
}

func TestNew_MissingDependency(t *testing.T) {
	deps := language.Dependencies{}

	_, err := language.New(deps)
	if err == nil {
		t.Error("expected error for missing dependencies, got nil")
	}
}

func TestName(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())

	// Create minimal test setup with translation file
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	
	i18nSvc, err := i18n.New(i18n.Dependencies{}, tmpDir)
	if err != nil {
		t.Fatalf("failed to create i18n: %v", err)
	}

	feature, err := language.New(language.Dependencies{
		I18n:   i18nSvc,
		Logger: log,
	})
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	name := feature.Name()
	if name != "language" {
		t.Errorf("expected name 'language', got '%s'", name)
	}
}

func TestGetMenuButton(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	i18nSvc, _ := i18n.New(i18n.Dependencies{}, tmpDir)

	feature, _ := language.New(language.Dependencies{
		I18n:   i18nSvc,
		Logger: log,
	})

	btn := feature.GetMenuButton()
	if btn == nil {
		t.Error("expected menu button, got nil")
	}

	if btn.Category != "admin" {
		t.Errorf("expected category 'admin', got '%s'", btn.Category)
	}

	if btn.SubCategory != "configuration" {
		t.Errorf("expected subcategory 'configuration', got '%s'", btn.SubCategory)
	}

	if !btn.AdminOnly {
		t.Error("expected AdminOnly to be true")
	}
	
	if btn.Tier != 3 {
		t.Errorf("expected tier 3, got %d", btn.Tier)
	}
}

