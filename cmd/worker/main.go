package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/core/queue"
	"welcomebot/internal/worker"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// Load configuration from environment
	slaveID := getEnv("SLAVE_ID", "slave-1")
	botToken := getEnv("DISCORD_BOT_TOKEN", "")
	if botToken == "" {
		fmt.Fprintf(os.Stderr, "DISCORD_BOT_TOKEN is required\n")
		os.Exit(1)
	}

	logCfg := logger.Config{
		Level:  getEnv("LOG_LEVEL", "info"),
		Format: getEnv("LOG_FORMAT", "json"),
	}

	// Initialize logger
	lgr, err := logger.New(logCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	lgr.Info("Starting Welcomebot Worker Bot", "slave_id", slaveID)

	// Initialize database
	dbCfg := database.Config{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     getEnv("POSTGRES_PORT", "5432"),
		User:     getEnv("POSTGRES_USER", "welcomebot"),
		Password: getEnv("POSTGRES_PASSWORD", ""),
		Database: getEnv("POSTGRES_DB", "welcomebot"),
		SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}

	db, err := database.New(dbCfg)
	if err != nil {
		lgr.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	lgr.Info("Database connected")

	// Initialize cache
	cacheCfg := cache.Config{
		SentinelAddrs: getSentinelAddrs(),
		MasterName:    getEnv("REDIS_MASTER_NAME", ""),
		Addr:          getEnv("REDIS_ADDR", "localhost:6379"),
		Password:      getEnv("REDIS_PASSWORD", ""),
		DB:            0,
	}

	cacheClient, err := cache.New(cacheCfg)
	if err != nil {
		lgr.Error("Failed to connect to cache", "error", err)
		os.Exit(1)
	}
	defer cacheClient.Close()

	lgr.Info("Cache connected")

	// Initialize queue
	queueCfg := queue.Config{
		SentinelAddrs: getSentinelAddrs(),
		MasterName:    getEnv("REDIS_MASTER_NAME", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		QueueKey:      "welcomebot:tasks",
	}

	queueClient, err := queue.New(queueCfg)
	if err != nil {
		lgr.Error("Failed to connect to queue", "error", err)
		os.Exit(1)
	}
	defer queueClient.Close()

	lgr.Info("Queue connected")

	// Initialize i18n
	i18nClient, err := i18n.New(i18n.Dependencies{
		DB:    db,
		Cache: cacheClient,
	}, "internal/core/i18n/translations")
	if err != nil {
		lgr.Error("Failed to initialize i18n", "error", err)
		os.Exit(1)
	}

	// Initialize Discord session
	discordSession, err := discordgo.New("Bot " + botToken)
	if err != nil {
		lgr.Error("Failed to create Discord session", "error", err)
		os.Exit(1)
	}

	// Set intents
	discordSession.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMessages

	// Create worker
	workerBot := &Worker{
		slaveID:        slaveID,
		session:        discordSession,
		db:             db,
		cache:          cacheClient,
		queue:          queueClient,
		logger:         lgr,
		i18n:           i18nClient,
		activeSessions: make(map[string]*worker.OnboardingSession),
	}

	// Add interaction handler for guide selection
	discordSession.AddHandler(workerBot.handleInteraction)

	// Open Discord connection
	if err := discordSession.Open(); err != nil {
		lgr.Error("Failed to open Discord connection", "error", err)
		os.Exit(1)
	}
	defer discordSession.Close()

	lgr.Info("Discord connected", "user", discordSession.State.User.String())

	// Mark slave as available
	statusKey := fmt.Sprintf("welcomebot:slaves:status:%s", slaveID)
	if err := cacheClient.Set(context.Background(), statusKey, "available", 30*time.Minute); err != nil {
		lgr.Warn("Failed to set initial slave status", "error", err)
	}

	lgr.Info("Welcomebot Worker Bot is running. Press CTRL-C to exit.", "slave_id", slaveID)

	// Start heartbeat
	go workerBot.sendHeartbeats(context.Background())

	// Start processing tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	go func() {
		<-sigChan
		lgr.Info("Shutdown signal received, stopping worker...")
		cancel()
	}()

	// Process tasks until shutdown
	workerBot.Run(ctx)

	lgr.Info("Worker stopped gracefully")
}

// Worker processes tasks from the queue.
type Worker struct {
	slaveID        string
	session        *discordgo.Session
	db             database.Client
	cache          cache.Client
	queue          queue.Client
	logger         logger.Logger
	i18n           i18n.I18n
	activeSessions map[string]*worker.OnboardingSession // Map of guildID:userID -> session
	sessionsMutex  sync.RWMutex                         // Protect the map
}

// Run starts the worker task processing loop.
func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("Worker started, waiting for tasks...")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Context cancelled, stopping worker")
			return
		default:
			w.processNextTask(ctx)
		}
	}
}

// processNextTask dequeues and processes one task.
func (w *Worker) processNextTask(ctx context.Context) {
	// Wait for task (30 second timeout)
	task, err := w.queue.Dequeue(ctx, 30*time.Second)
	if err != nil {
		w.logger.Error("Failed to dequeue task", "error", err)
		time.Sleep(5 * time.Second)
		return
	}

	// No task available (timeout)
	if task == nil {
		return
	}

	w.logger.Info("Processing task",
		"task_id", task.ID,
		"task_type", task.Type,
		"guild_id", task.GuildID,
	)

	// Process task based on type
	if err := w.handleTask(ctx, task); err != nil {
		w.logger.Error("Task processing failed",
			"task_id", task.ID,
			"task_type", task.Type,
			"error", err,
		)
		// TODO: Implement retry logic if needed
		return
	}

	w.logger.Info("Task completed",
		"task_id", task.ID,
		"task_type", task.Type,
	)
}

// handleTask routes tasks to appropriate handlers.
func (w *Worker) handleTask(ctx context.Context, task *queue.Task) error {
	switch task.Type {
	case "onboarding_start":
		return w.handleOnboardingStart(ctx, task)
	case "onboarding_complete":
		return w.handleOnboardingComplete(ctx, task)
	default:
		w.logger.Warn("Unknown task type", "task_type", task.Type)
		return nil
	}
}

// handleOnboardingStart handles the start of an onboarding session.
func (w *Worker) handleOnboardingStart(ctx context.Context, task *queue.Task) error {
	w.logger.Info("Starting onboarding session", "task_id", task.ID)

	// Create onboarding session
	session, err := worker.NewOnboardingSession(
		ctx,
		task,
		w.session,
		w.db,
		w.cache,
		w.queue,
		w.logger,
		w.i18n,
	)
	if err != nil {
		w.logger.Error("Failed to create onboarding session", "error", err)
		return err
	}

	// Store session in active sessions map for interaction handling
	sessionKey := fmt.Sprintf("%s:%s", task.GuildID, session.GetUserID())
	w.sessionsMutex.Lock()
	w.activeSessions[sessionKey] = session
	w.sessionsMutex.Unlock()

	// Start the session (blocks until complete)
	err = session.Start()
	
	// Remove from active sessions when done
	w.sessionsMutex.Lock()
	delete(w.activeSessions, sessionKey)
	w.sessionsMutex.Unlock()

	if err != nil {
		w.logger.Error("Failed to start onboarding session", "error", err)
		return err
	}

	return nil
}

// handleOnboardingComplete handles completion notification from master.
func (w *Worker) handleOnboardingComplete(ctx context.Context, task *queue.Task) error {
	w.logger.Info("Onboarding completion received", "task_id", task.ID)
	// Master has acknowledged completion - no action needed
	return nil
}

// sendHeartbeats periodically sends heartbeat to indicate slave is alive.
func (w *Worker) sendHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			statusKey := fmt.Sprintf("welcomebot:slaves:status:%s", w.slaveID)
			status := "available" // TODO: Track actual status

			if err := w.cache.Set(ctx, statusKey, status, 2*time.Minute); err != nil {
				w.logger.Warn("Failed to send heartbeat", "error", err)
			}
		}
	}
}

// handleInteraction handles button clicks and dropdown selections for guide selection.
func (w *Worker) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Extract custom ID
	var customID string
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		customID = i.MessageComponentData().CustomID
	default:
		return
	}

	// Handle preview button: onboarding:preview:{guide}:{userID}
	if strings.HasPrefix(customID, "onboarding:preview:") {
		w.handlePreviewButton(ctx, s, i, customID)
		return
	}

	// Handle guide selection: onboarding:select_guide:{userID}
	if strings.HasPrefix(customID, "onboarding:select_guide:") {
		w.handleGuideSelection(ctx, s, i, customID)
		return
	}

	// Handle guide confirmation: onboarding:confirm_guide:{guide}:{userID}
	if strings.HasPrefix(customID, "onboarding:confirm_guide:") {
		w.handleGuideConfirmation(ctx, s, i, customID)
		return
	}

	// Handle back to guide selection: onboarding:back_to_guide_selection:{userID}
	if strings.HasPrefix(customID, "onboarding:back_to_guide_selection:") {
		w.handleBackToGuideSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 1 Next button: onboarding:step1_next:{userID}
	if strings.HasPrefix(customID, "onboarding:step1_next:") {
		w.handleStep1Next(ctx, s, i, customID)
		return
	}

	// Handle Step 1 Replay button: onboarding:step1_replay:{userID}
	if strings.HasPrefix(customID, "onboarding:step1_replay:") {
		w.handleStep1Replay(ctx, s, i, customID)
		return
	}

	// Handle Step 2 Next button: onboarding:step2_next:{userID}
	if strings.HasPrefix(customID, "onboarding:step2_next:") {
		w.handleStep2Next(ctx, s, i, customID)
		return
	}

	// Handle Step 2 Replay button: onboarding:step2_replay:{userID}
	if strings.HasPrefix(customID, "onboarding:step2_replay:") {
		w.handleStep2Replay(ctx, s, i, customID)
		return
	}

	// Handle Step 4 Next button: onboarding:step4_next:{userID}
	if strings.HasPrefix(customID, "onboarding:step4_next:") {
		w.handleStep4Next(ctx, s, i, customID)
		return
	}

	// Handle Step 4 Replay button: onboarding:step4_replay:{userID}
	if strings.HasPrefix(customID, "onboarding:step4_replay:") {
		w.handleStep4Replay(ctx, s, i, customID)
		return
	}

	// Handle Step 5 Next button: onboarding:step5_next:{userID}
	if strings.HasPrefix(customID, "onboarding:step5_next:") {
		w.handleStep5Next(ctx, s, i, customID)
		return
	}

	// Handle Step 5 Replay button: onboarding:step5_replay:{userID}
	if strings.HasPrefix(customID, "onboarding:step5_replay:") {
		w.handleStep5Replay(ctx, s, i, customID)
		return
	}

	// Handle Step 6 Next button: onboarding:step6_next:{userID}
	if strings.HasPrefix(customID, "onboarding:step6_next:") {
		w.handleStep6Next(ctx, s, i, customID)
		return
	}

	// Handle Step 6 Replay button: onboarding:step6_replay:{userID}
	if strings.HasPrefix(customID, "onboarding:step6_replay:") {
		w.handleStep6Replay(ctx, s, i, customID)
		return
	}

	// Handle Step 7 Complete button: onboarding:step7_complete:{userID}
	if strings.HasPrefix(customID, "onboarding:step7_complete:") {
		w.handleStep7Complete(ctx, s, i, customID)
		return
	}

	// Handle Step 7 Replay button: onboarding:step7_replay:{userID}
	if strings.HasPrefix(customID, "onboarding:step7_replay:") {
		w.handleStep7Replay(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Age selection: onboarding:age:{ageType}:{userID}
	if strings.HasPrefix(customID, "onboarding:age:") {
		w.handleStep3AgeSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Voice selection: onboarding:voice:{voiceType}:{userID}
	if strings.HasPrefix(customID, "onboarding:voice:") {
		w.handleStep3VoiceSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Eroipu selection: onboarding:eroipu:{choice}:{userID}
	if strings.HasPrefix(customID, "onboarding:eroipu:") {
		w.handleStep3EroipuSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Neochi OK/NG selection: onboarding:neochi:{choice}:{userID}
	if strings.HasPrefix(customID, "onboarding:neochi:") {
		w.handleStep3NeochiOkNgSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Neochi handling selection: onboarding:neochi_handling:{choice}:{userID}
	if strings.HasPrefix(customID, "onboarding:neochi_handling:") {
		w.handleStep3NeochiHandlingSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 DM selection: onboarding:dm:{choice}:{userID}
	if strings.HasPrefix(customID, "onboarding:dm:") {
		w.handleStep3DMSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Friend selection: onboarding:friend:{choice}:{userID}
	if strings.HasPrefix(customID, "onboarding:friend:") {
		w.handleStep3FriendSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Event selection: onboarding:event:{eventType}:{userID}
	if strings.HasPrefix(customID, "onboarding:event:") {
		w.handleStep3EventSelection(ctx, s, i, customID)
		return
	}

	// Handle Step 3 Next button: onboarding:step3_next:{userID}
	if strings.HasPrefix(customID, "onboarding:step3_next:") {
		w.handleStep3Next(ctx, s, i, customID)
		return
	}
}

// handlePreviewButton handles guide preview button clicks.
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

