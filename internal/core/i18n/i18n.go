package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
)

const (
	defaultLanguage = "en"
	cacheKeyPrefix  = "welcomebot:i18n:guild:"
)

// I18n provides internationalization functionality.
type I18n interface {
	T(ctx context.Context, guildID, key string) string
	TWithArgs(ctx context.Context, guildID, key string, args map[string]string) string
	SetGuildLanguage(ctx context.Context, guildID, langCode string) error
	GetGuildLanguage(ctx context.Context, guildID string) (string, error)
	HasGuildLanguage(ctx context.Context, guildID string) bool
	AvailableLanguages() []string
}

// Dependencies contains i18n dependencies.
type Dependencies struct {
	DB    database.Client
	Cache cache.Client
}

// manager implements I18n.
type manager struct {
	db           database.Client
	cache        cache.Client
	translations map[string]map[string]interface{}
	mu           sync.RWMutex
}

// New creates a new i18n manager.
func New(deps Dependencies, translationsDir string) (I18n, error) {
	m := &manager{
		db:           deps.DB,
		cache:        deps.Cache,
		translations: make(map[string]map[string]interface{}),
	}

	if err := m.loadTranslations(translationsDir); err != nil {
		return nil, fmt.Errorf("load translations: %w", err)
	}

	return m, nil
}

// loadTranslations loads all translation files.
func (m *manager) loadTranslations(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return fmt.Errorf("glob translations: %w", err)
	}

	for _, file := range files {
		langCode := extractLangCode(file)
		if err := m.loadTranslationFile(langCode, file); err != nil {
			return fmt.Errorf("load %s: %w", langCode, err)
		}
	}

	if len(m.translations) == 0 {
		return fmt.Errorf("no translations loaded from %s", dir)
	}

	return nil
}

// loadTranslationFile loads a single translation file.
func (m *manager) loadTranslationFile(langCode, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var translations map[string]interface{}
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	m.mu.Lock()
	m.translations[langCode] = translations
	m.mu.Unlock()

	return nil
}

// T translates a key for the given guild.
func (m *manager) T(ctx context.Context, guildID, key string) string {
	return m.TWithArgs(ctx, guildID, key, nil)
}

// TWithArgs translates a key with variable substitution.
func (m *manager) TWithArgs(ctx context.Context, guildID, key string, args map[string]string) string {
	lang, err := m.getGuildLang(ctx, guildID)
	if err != nil {
		lang = defaultLanguage
	}

	// Try guild's language
	value := m.lookup(lang, key)
	
	// Fallback to English if not found
	if value == "" && lang != defaultLanguage {
		value = m.lookup(defaultLanguage, key)
	}

	// Ultimate fallback: return key itself
	if value == "" {
		return key
	}

	// Substitute variables
	return m.substitute(value, args)
}

// SetGuildLanguage sets the language for a guild.
func (m *manager) SetGuildLanguage(ctx context.Context, guildID, langCode string) error {
	query := `
		INSERT INTO guild_languages (guild_id, language_code, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (guild_id) 
		DO UPDATE SET language_code = $2, updated_at = NOW()
	`

	_, err := m.db.Exec(ctx, query, guildID, langCode)
	if err != nil {
		return fmt.Errorf("set guild language: %w", err)
	}

	// Cache indefinitely (until changed)
	cacheKey := cacheKeyPrefix + guildID
	if err := m.cache.Set(ctx, cacheKey, langCode, 0); err != nil {
		// Log but don't fail - cache is optional
	}

	return nil
}

// GetGuildLanguage gets the language for a guild.
func (m *manager) GetGuildLanguage(ctx context.Context, guildID string) (string, error) {
	lang, err := m.getGuildLang(ctx, guildID)
	if err != nil {
		return defaultLanguage, nil
	}
	return lang, nil
}

// HasGuildLanguage checks if guild has explicitly configured language.
func (m *manager) HasGuildLanguage(ctx context.Context, guildID string) bool {
	_, err := m.getGuildLang(ctx, guildID)
	return err == nil
}

// AvailableLanguages returns list of supported languages.
func (m *manager) AvailableLanguages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	langs := make([]string, 0, len(m.translations))
	for lang := range m.translations {
		langs = append(langs, lang)
	}
	return langs
}

// getGuildLang retrieves guild language from cache or DB.
func (m *manager) getGuildLang(ctx context.Context, guildID string) (string, error) {
	cacheKey := cacheKeyPrefix + guildID

	// Try cache first
	lang, err := m.cache.Get(ctx, cacheKey)
	if err == nil && lang != "" {
		return lang, nil
	}

	// Query database
	query := "SELECT language_code FROM guild_languages WHERE guild_id = $1"
	row := m.db.QueryRow(ctx, query, guildID)

	var langCode string
	if err := row.Scan(&langCode); err != nil {
		return "", fmt.Errorf("query guild language: %w", err)
	}

	// Cache indefinitely
	m.cache.Set(ctx, cacheKey, langCode, 0)

	return langCode, nil
}

// lookup finds a translation value by key path.
func (m *manager) lookup(lang, key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	translations, ok := m.translations[lang]
	if !ok {
		return ""
	}

	return m.navigateJSON(translations, key)
}

// navigateJSON navigates nested JSON by dot-separated key.
func (m *manager) navigateJSON(data map[string]interface{}, key string) string {
	parts := strings.Split(key, ".")
	current := data

	for i, part := range parts {
		value, ok := current[part]
		if !ok {
			return ""
		}

		// Last part - should be string
		if i == len(parts)-1 {
			if str, ok := value.(string); ok {
				return str
			}
			return ""
		}

		// Navigate deeper
		if nested, ok := value.(map[string]interface{}); ok {
			current = nested
		} else {
			return ""
		}
	}

	return ""
}

// substitute replaces {key} placeholders with values.
func (m *manager) substitute(text string, args map[string]string) string {
	if args == nil {
		return text
	}

	result := text
	for key, value := range args {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// extractLangCode extracts language code from filename.
func extractLangCode(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".json")
}

