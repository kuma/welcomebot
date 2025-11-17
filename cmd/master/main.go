package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"welcomebot/internal/bot"
	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/core/queue"
	"welcomebot/internal/features/botinfo"
	"welcomebot/internal/features/gender"
	"welcomebot/internal/features/initialization"
	"welcomebot/internal/features/language"
	"welcomebot/internal/features/menu"
	"welcomebot/internal/features/ping"
	"welcomebot/internal/features/selfintro"
	"welcomebot/internal/features/welcome"
	"welcomebot/internal/features/agerange"
	"welcomebot/internal/features/voicetype"
	"welcomebot/internal/features/otherroles1"
	"welcomebot/internal/features/otherroles2"
)

func main() {
	// Load configuration from environment
	cfg := bot.Config{
		Token: getEnv("DISCORD_BOT_TOKEN", ""),
		Database: database.Config{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "welcomebot"),
			Password: getEnv("POSTGRES_PASSWORD", ""),
			Database: getEnv("POSTGRES_DB", "welcomebot"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Cache: cache.Config{
			SentinelAddrs: getSentinelAddrs(),
			MasterName:    getEnv("REDIS_MASTER_NAME", ""),
			Addr:          getEnv("REDIS_ADDR", "localhost:6379"),
			Password:      getEnv("REDIS_PASSWORD", ""),
			DB:            0,
		},
		Queue: queue.Config{
			SentinelAddrs: getSentinelAddrs(),
			MasterName:    getEnv("REDIS_MASTER_NAME", ""),
			RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
			RedisPassword: getEnv("REDIS_PASSWORD", ""),
			RedisDB:       0,
			QueueKey:      "welcomebot:tasks",
		},
		Logger: logger.Config{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	if cfg.Token == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable is required")
	}

	// Create bot
	bot, deps, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Register features in order
	
	// 1. Ping feature
	pingFeature, err := ping.New(ping.Dependencies{
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create ping feature: %v", err)
	}
	if err := bot.Registry().Register(pingFeature); err != nil {
		log.Fatalf("Failed to register ping feature: %v", err)
	}

	// 2. Bot Info feature
	botinfoFeature, err := botinfo.New(botinfo.Dependencies{
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create botinfo feature: %v", err)
	}
	if err := bot.Registry().Register(botinfoFeature); err != nil {
		log.Fatalf("Failed to register botinfo feature: %v", err)
	}

	// 3. Language feature
	languageFeature, err := language.New(language.Dependencies{
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create language feature: %v", err)
	}
	if err := bot.Registry().Register(languageFeature); err != nil {
		log.Fatalf("Failed to register language feature: %v", err)
	}

	// 3.5 Gender feature
	genderFeature, err := gender.New(gender.Dependencies{
		DB:     deps.DB,
		Cache:  deps.Cache,
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create gender feature: %v", err)
	}
	if err := bot.Registry().Register(genderFeature); err != nil {
		log.Fatalf("Failed to register gender feature: %v", err)
	}

	// 3.6 Self-Intro feature
	selfintroFeature, err := selfintro.New(selfintro.Dependencies{
		DB:     deps.DB,
		Cache:  deps.Cache,
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create selfintro feature: %v", err)
	}
	if err := bot.Registry().Register(selfintroFeature); err != nil {
		log.Fatalf("Failed to register selfintro feature: %v", err)
	}

	// 3.7 Welcome feature
	welcomeFeature, err := welcome.New(welcome.Dependencies{
		DB:      deps.DB,
		Cache:   deps.Cache,
		Queue:   deps.Queue,
		I18n:    deps.I18n,
		Logger:  deps.Logger,
		Session: bot.Session(),
	})
	if err != nil {
		log.Fatalf("Failed to create welcome feature: %v", err)
	}
	if err := bot.Registry().Register(welcomeFeature); err != nil {
		log.Fatalf("Failed to register welcome feature: %v", err)
	}

	// 3.8 Age Range feature
	ageRangeFeature, err := agerange.New(agerange.Dependencies{
		DB:     deps.DB,
		Cache:  deps.Cache,
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create age range feature: %v", err)
	}
	if err := bot.Registry().Register(ageRangeFeature); err != nil {
		log.Fatalf("Failed to register age range feature: %v", err)
	}

	// 3.9 Voice Type feature
	voiceTypeFeature, err := voicetype.New(voicetype.Dependencies{
		DB:     deps.DB,
		Cache:  deps.Cache,
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create voice type feature: %v", err)
	}
	if err := bot.Registry().Register(voiceTypeFeature); err != nil {
		log.Fatalf("Failed to register voice type feature: %v", err)
	}

	// 3.10 Other Roles 1 feature
	otherRoles1Feature, err := otherroles1.New(otherroles1.Dependencies{
		DB:     deps.DB,
		Cache:  deps.Cache,
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create other roles 1 feature: %v", err)
	}
	if err := bot.Registry().Register(otherRoles1Feature); err != nil {
		log.Fatalf("Failed to register other roles 1 feature: %v", err)
	}

	// 3.11 Other Roles 2 feature
	otherRoles2Feature, err := otherroles2.New(otherroles2.Dependencies{
		DB:     deps.DB,
		Cache:  deps.Cache,
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create other roles 2 feature: %v", err)
	}
	if err := bot.Registry().Register(otherRoles2Feature); err != nil {
		log.Fatalf("Failed to register other roles 2 feature: %v", err)
	}

	// 4. Initialization feature
	initFeature, err := initialization.New(initialization.Dependencies{
		I18n:   deps.I18n,
		Logger: deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create initialization feature: %v", err)
	}
	// Link language feature to init
	initFeature.SetLanguageFeature(languageFeature)
	if err := bot.Registry().Register(initFeature); err != nil {
		log.Fatalf("Failed to register initialization feature: %v", err)
	}

	// 5. Menu feature (must be registered last, depends on others)
	menuFeature, err := menu.New(menu.Dependencies{
		Registry: bot.Registry(),
		Init:     initFeature,
		I18n:     deps.I18n,
		Logger:   deps.Logger,
	})
	if err != nil {
		log.Fatalf("Failed to create menu feature: %v", err)
	}
	if err := bot.Registry().Register(menuFeature); err != nil {
		log.Fatalf("Failed to register menu feature: %v", err)
	}

	// Start bot
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	deps.Logger.Info("welcomebot Master Bot is running. Press CTRL-C to exit.")

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Graceful shutdown
	deps.Logger.Info("Shutting down...")
	if err := bot.Stop(); err != nil {
		deps.Logger.Error("Error during shutdown", "error", err)
	}

	// Close resources
	if err := deps.DB.Close(); err != nil {
		deps.Logger.Error("Error closing database", "error", err)
	}
	if err := deps.Cache.Close(); err != nil {
		deps.Logger.Error("Error closing cache", "error", err)
	}
	if err := deps.Queue.Close(); err != nil {
		deps.Logger.Error("Error closing queue", "error", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getSentinelAddrs() []string {
	addrs := getEnv("REDIS_SENTINEL_ADDRS", "")
	if addrs == "" {
		return nil
	}
	// Split by comma: "host1:26379,host2:26379"
	parts := make([]string, 0)
	for _, addr := range splitAndTrim(addrs, ",") {
		if addr != "" {
			parts = append(parts, addr)
		}
	}
	return parts
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimString(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	current := ""
	for _, char := range s {
		if string(char) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

