package selfintro_test

import (
	"os"
	"testing"

	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/features/selfintro"
)

func TestNew(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	i18nSvc, _ := i18n.New(i18n.Dependencies{}, tmpDir)

	deps := selfintro.Dependencies{
		DB:     nil,
		Cache:  nil,
		I18n:   i18nSvc,
		Logger: log,
	}

	_, err := selfintro.New(deps)
	if err == nil {
		t.Error("expected error for nil DB, got nil")
	}
}

func TestName(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	i18nSvc, _ := i18n.New(i18n.Dependencies{}, tmpDir)

	dbCfg := database.DefaultConfig()
	db, err := database.New(dbCfg)
	if err != nil {
		t.Skip("database not available")
	}
	defer db.Close()

	cacheCfg := cache.DefaultConfig()
	cacheClient, err := cache.New(cacheCfg)
	if err != nil {
		t.Skip("cache not available")
	}
	defer cacheClient.Close()

	feature, _ := selfintro.New(selfintro.Dependencies{
		DB:     db,
		Cache:  cacheClient,
		I18n:   i18nSvc,
		Logger: log,
	})

	name := feature.Name()
	if name != "selfintro" {
		t.Errorf("expected name 'selfintro', got '%s'", name)
	}
}

func TestGetMenuButton(t *testing.T) {
	log, _ := logger.New(logger.DefaultConfig())
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/en.json", []byte(`{}`), 0644)
	i18nSvc, _ := i18n.New(i18n.Dependencies{}, tmpDir)

	dbCfg := database.DefaultConfig()
	db, err := database.New(dbCfg)
	if err != nil {
		t.Skip("database not available")
	}
	defer db.Close()

	cacheCfg := cache.DefaultConfig()
	cacheClient, err := cache.New(cacheCfg)
	if err != nil {
		t.Skip("cache not available")
	}
	defer cacheClient.Close()

	feature, _ := selfintro.New(selfintro.Dependencies{
		DB:     db,
		Cache:  cacheClient,
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
}

