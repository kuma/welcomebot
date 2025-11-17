package bot

import (
	"context"
	"fmt"

	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/discord"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/core/queue"

	"github.com/bwmarrin/discordgo"
)

// Bot represents the Discord bot.
type Bot struct {
	session  *discordgo.Session
	registry *Registry
	logger   logger.Logger
}

// Config contains bot configuration.
type Config struct {
	Token    string
	Database database.Config
	Cache    cache.Config
	Queue    queue.Config
	Logger   logger.Config
}

// Dependencies contains all bot dependencies.
type Dependencies struct {
	DB      database.Client
	Cache   cache.Client
	Queue   queue.Client
	Discord discord.Helper
	Logger  logger.Logger
	I18n    i18n.I18n
}

// New creates a new bot instance.
func New(cfg Config) (*Bot, *Dependencies, error) {
	// Initialize logger
	log, err := logger.New(cfg.Logger)
	if err != nil {
		return nil, nil, fmt.Errorf("create logger: %w", err)
	}

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("create database: %w", err)
	}

	// Initialize cache
	cacheClient, err := cache.New(cfg.Cache)
	if err != nil {
		return nil, nil, fmt.Errorf("create cache: %w", err)
	}

	// Initialize queue
	queueClient, err := queue.New(cfg.Queue)
	if err != nil {
		return nil, nil, fmt.Errorf("create queue: %w", err)
	}

	// Create Discord session
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("create discord session: %w", err)
	}

	// Create Discord helper
	discordHelper := discord.New(session)

	// Create i18n manager
	i18nManager, err := i18n.New(i18n.Dependencies{
		DB:    db,
		Cache: cacheClient,
	}, "internal/core/i18n/translations")
	if err != nil {
		return nil, nil, fmt.Errorf("create i18n: %w", err)
	}

	// Run database migrations
	migrator := database.NewMigrator(db)
	if err := migrator.RunMigrations(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}
	log.Info("database migrations completed")

	// Create dependencies
	deps := &Dependencies{
		DB:      db,
		Cache:   cacheClient,
		Queue:   queueClient,
		Discord: discordHelper,
		Logger:  log,
		I18n:    i18nManager,
	}

	// Create feature registry
	registry := NewRegistry(log)

	bot := &Bot{
		session:  session,
		registry: registry,
		logger:   log,
	}

	return bot, deps, nil
}

// Start starts the bot and connects to Discord.
func (b *Bot) Start() error {
	// Set intents
	b.session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildMembers

	// Register event handlers
	b.session.AddHandler(b.handleInteraction)
	b.session.AddHandler(b.handleMessageCreate)
	b.session.AddHandler(b.handleMessageDelete)
	b.session.AddHandler(b.handleReactionAdd)
	b.session.AddHandler(b.handleVoiceStateUpdate)

	// Open connection
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open discord connection: %w", err)
	}

	b.logger.Info("bot connected", "user", b.session.State.User.String())

	// Register slash commands
	if err := b.registry.RegisterSlashCommands(b.session); err != nil {
		return fmt.Errorf("register slash commands: %w", err)
	}

	return nil
}

// Stop gracefully stops the bot.
func (b *Bot) Stop() error {
	if err := b.session.Close(); err != nil {
		return fmt.Errorf("close discord session: %w", err)
	}

	b.logger.Info("bot stopped")
	return nil
}

// Registry returns the feature registry.
func (b *Bot) Registry() *Registry {
	return b.registry
}

// Session returns the Discord session.
func (b *Bot) Session() *discordgo.Session {
	return b.session
}

// handleInteraction routes interaction events to features.
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	b.registry.HandleInteraction(ctx, s, i)
}

// handleMessageCreate routes message creation events to features.
func (b *Bot) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // Ignore own messages
	}

	ctx := context.Background()
	b.registry.HandleMessage(ctx, s, m)
}

// handleMessageDelete routes message deletion events to features.
func (b *Bot) handleMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	ctx := context.Background()
	b.registry.HandleMessageDelete(ctx, s, m)
}

// handleReactionAdd routes reaction add events to features.
func (b *Bot) handleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	ctx := context.Background()
	b.registry.HandleReactionAdd(ctx, s, r)
}

// handleVoiceStateUpdate routes voice state updates to features.
func (b *Bot) handleVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	ctx := context.Background()
	b.registry.HandleVoiceStateUpdate(ctx, s, v)
}

