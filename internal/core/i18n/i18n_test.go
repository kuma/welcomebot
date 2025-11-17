package i18n_test

import (
	"os"
	"path/filepath"
	"testing"

	"welcomebot/internal/core/i18n"
)

func TestNew(t *testing.T) {
	// Create temp directory with test translations
	tmpDir := t.TempDir()

	enFile := filepath.Join(tmpDir, "en.json")
	os.WriteFile(enFile, []byte(`{"test": {"key": "value"}}`), 0644)

	deps := i18n.Dependencies{
		DB:    nil, // Will be needed for full integration
		Cache: nil,
	}

	mgr, err := i18n.New(deps, tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if mgr == nil {
		t.Error("expected manager, got nil")
	}
}

func TestAvailableLanguages(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "en.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "ja.json"), []byte(`{}`), 0644)

	deps := i18n.Dependencies{}
	mgr, err := i18n.New(deps, tmpDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	langs := mgr.AvailableLanguages()
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d", len(langs))
	}
}

