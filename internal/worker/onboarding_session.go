package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/core/queue"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

const (
	sessionTimeout = 60 * time.Minute
	inactivityTimeout = 20 * time.Minute
)

// OnboardingSession handles a single user's onboarding session.
type OnboardingSession struct {
	guildID          string
	userID           string
	slaveID          string
	categoryID       string
	vcChannelID      string
	selectedGuide    string // Selected guide name (e.g., "kk")
	currentStep      int    // Current tutorial step (0-7)
	currentSubStep   int    // Current sub-step within a step (for multi-part steps like Step 3)
	currentAudioFile string // Current audio file being played
	inProgressRoleID string
	completedRoleID  string
	entranceRoleID      string
	nyukaiRoleID        string
	Setsumeikai1RoleID  string // Exported for handler access
	Setsumeikai2RoleID  string // Exported for handler access
	Setsumeikai3RoleID  string // Exported for handler access
	MemberRoleID        string // Exported for handler access
	VisitorRoleID       string // Exported for handler access
	// Age range roles (exported for handler access)
	Age20EarlyRoleID string
	Age20LateRoleID  string
	Age30EarlyRoleID string
	Age30LateRoleID  string
	Age40EarlyRoleID string
	Age40LateRoleID  string
	// Voice type roles (exported for handler access)
	HighVoiceRoleID    string
	MidHighVoiceRoleID string
	MidVoiceRoleID     string
	MidLowVoiceRoleID  string
	LowVoiceRoleID     string
	// Other roles (exported for handler access)
	EroOkRoleID            string
	EroNgRoleID            string
	NeochiOkRoleID         string
	NeochiNgRoleID         string
	NeochiDisconnectRoleID string
	DmOkRoleID             string
	DmNgRoleID             string
	FriendOkRoleID         string
	FriendNgRoleID         string
	BunnyclubEventRoleID   string
	UserEventRoleID        string
	startedAt              time.Time
	lastActivity           time.Time

	session       *discordgo.Session
	db            database.Client
	cache         cache.Client
	queue         queue.Client
	logger        logger.Logger
	i18n          i18n.I18n
	voiceConn     *discordgo.VoiceConnection
	currentStream *dca.StreamingSession // Active audio stream
	stopStream    chan struct{}         // Channel to signal stream stop
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewOnboardingSession creates a new onboarding session.
func NewOnboardingSession(
	ctx context.Context,
	task *queue.Task,
	session *discordgo.Session,
	db database.Client,
	cache cache.Client,
	queue queue.Client,
	logger logger.Logger,
	i18nClient i18n.I18n,
) (*OnboardingSession, error) {
	// Extract task payload
	userID, ok := task.Payload["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user_id in task payload")
	}

	categoryID, ok := task.Payload["category_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing category_id in task payload")
	}

	slaveID, ok := task.Payload["slave_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing slave_id in task payload")
	}

	// Optional role IDs
	inProgressRole, _ := task.Payload["in_progress_role"].(string)
	completedRole, _ := task.Payload["completed_role"].(string)
	entranceRole, _ := task.Payload["entrance_role"].(string)
	nyukaiRole, _ := task.Payload["nyukai_role"].(string)
	setsumeikai1Role, _ := task.Payload["setsumeikai_1_role"].(string)
	setsumeikai2Role, _ := task.Payload["setsumeikai_2_role"].(string)
	setsumeikai3Role, _ := task.Payload["setsumeikai_3_role"].(string)
	memberRole, _ := task.Payload["member_role"].(string)
	visitorRole, _ := task.Payload["visitor_role"].(string)
	// Age range roles
	age20Early, _ := task.Payload["age_20_early_role"].(string)
	age20Late, _ := task.Payload["age_20_late_role"].(string)
	age30Early, _ := task.Payload["age_30_early_role"].(string)
	age30Late, _ := task.Payload["age_30_late_role"].(string)
	age40Early, _ := task.Payload["age_40_early_role"].(string)
	age40Late, _ := task.Payload["age_40_late_role"].(string)
	// Voice type roles
	highVoice, _ := task.Payload["high_voice_role"].(string)
	midHighVoice, _ := task.Payload["mid_high_voice_role"].(string)
	midVoice, _ := task.Payload["mid_voice_role"].(string)
	midLowVoice, _ := task.Payload["mid_low_voice_role"].(string)
	lowVoice, _ := task.Payload["low_voice_role"].(string)
	// Other roles
	eroOk, _ := task.Payload["ero_ok_role"].(string)
	eroNg, _ := task.Payload["ero_ng_role"].(string)
	neochiOk, _ := task.Payload["neochi_ok_role"].(string)
	neochiNg, _ := task.Payload["neochi_ng_role"].(string)
	neochiDisconnect, _ := task.Payload["neochi_disconnect_role"].(string)
	dmOk, _ := task.Payload["dm_ok_role"].(string)
	dmNg, _ := task.Payload["dm_ng_role"].(string)
	friendOk, _ := task.Payload["friend_ok_role"].(string)
	friendNg, _ := task.Payload["friend_ng_role"].(string)
	bunnyclubEvent, _ := task.Payload["bunnyclub_event_role"].(string)
	userEvent, _ := task.Payload["user_event_role"].(string)

	sessionCtx, cancel := context.WithTimeout(ctx, sessionTimeout)

	return &OnboardingSession{
		guildID:                task.GuildID,
		userID:                 userID,
		slaveID:                slaveID,
		categoryID:             categoryID,
		inProgressRoleID:       inProgressRole,
		completedRoleID:        completedRole,
		entranceRoleID:         entranceRole,
		nyukaiRoleID:           nyukaiRole,
		Setsumeikai1RoleID:     setsumeikai1Role,
		Setsumeikai2RoleID:     setsumeikai2Role,
		Setsumeikai3RoleID:     setsumeikai3Role,
		MemberRoleID:           memberRole,
		VisitorRoleID:          visitorRole,
		Age20EarlyRoleID:       age20Early,
		Age20LateRoleID:        age20Late,
		Age30EarlyRoleID:       age30Early,
		Age30LateRoleID:        age30Late,
		Age40EarlyRoleID:       age40Early,
		Age40LateRoleID:        age40Late,
		HighVoiceRoleID:        highVoice,
		MidHighVoiceRoleID:     midHighVoice,
		MidVoiceRoleID:         midVoice,
		MidLowVoiceRoleID:      midLowVoice,
		LowVoiceRoleID:         lowVoice,
		EroOkRoleID:            eroOk,
		EroNgRoleID:            eroNg,
		NeochiOkRoleID:         neochiOk,
		NeochiNgRoleID:         neochiNg,
		NeochiDisconnectRoleID: neochiDisconnect,
		DmOkRoleID:             dmOk,
		DmNgRoleID:             dmNg,
		FriendOkRoleID:         friendOk,
		FriendNgRoleID:         friendNg,
		BunnyclubEventRoleID:   bunnyclubEvent,
		UserEventRoleID:        userEvent,
		startedAt:              time.Now(),
		lastActivity:           time.Now(),
		session:                session,
		db:                     db,
		cache:                  cache,
		queue:                  queue,
		logger:                 logger,
		i18n:                   i18nClient,
		stopStream:             make(chan struct{}),
		ctx:                    sessionCtx,
		cancel:                 cancel,
	}, nil
}

// Start begins the onboarding session.
func (s *OnboardingSession) Start() error {
	s.logger.Info("starting onboarding session",
		"guild_id", s.guildID,
		"user_id", s.userID,
		"slave_id", s.slaveID,
	)

	// Add in-progress role if configured
	if s.inProgressRoleID != "" {
		if err := s.addRole(s.inProgressRoleID); err != nil {
			s.logger.Warn("failed to add in-progress role", "error", err)
		}
	}

	// Create voice channel
	vcChannel, err := s.createVoiceChannel()
	if err != nil {
		return fmt.Errorf("create voice channel: %w", err)
	}
	s.vcChannelID = vcChannel.ID

	s.logger.Info("voice channel created",
		"channel_id", s.vcChannelID,
		"channel_name", vcChannel.Name,
	)

	// Join voice channel
	if err := s.joinVoiceChannel(); err != nil {
		s.cleanup()
		return fmt.Errorf("join voice channel: %w", err)
	}

	// Save session data to Redis for interaction handlers
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Send welcome message in VC text channel
	if err := s.sendWelcomeMessage(); err != nil {
		s.logger.Warn("failed to send welcome message", "error", err)
	}

	// Start inactivity monitor
	go s.monitorInactivity()

	// Block until session completes or times out
	select {
	case <-s.ctx.Done():
		s.logger.Info("session context cancelled")
	case <-time.After(sessionTimeout):
		s.logger.Warn("session exceeded maximum duration")
	}

	// Cleanup
	s.cleanup()

	return nil
}

// createVoiceChannel creates a temporary voice channel for the user.
func (s *OnboardingSession) createVoiceChannel() (*discordgo.Channel, error) {
	// Get user info for channel name
	user, err := s.session.User(s.userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	channelName := fmt.Sprintf("onboarding-%s", user.Username)

	bitrate := 96000 // 96kbps (Discord's maximum)
	userLimit := 2   // Max 2 users (user + bot)

	channel, err := s.session.GuildChannelCreateComplex(s.guildID, discordgo.GuildChannelCreateData{
		Name:      channelName,
		Type:      discordgo.ChannelTypeGuildVoice,
		ParentID:  s.categoryID,
		Bitrate:   bitrate,
		UserLimit: userLimit,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			// Only the user and the bot can see/join
			{
				ID:   s.userID,
				Type: discordgo.PermissionOverwriteTypeMember,
				Allow: discordgo.PermissionViewChannel |
					discordgo.PermissionVoiceConnect |
					discordgo.PermissionVoiceSpeak,
			},
			{
				ID:   s.session.State.User.ID,
				Type: discordgo.PermissionOverwriteTypeMember,
				Allow: discordgo.PermissionViewChannel |
					discordgo.PermissionVoiceConnect |
					discordgo.PermissionVoiceSpeak,
			},
			// Hide from @everyone
			{
				ID:   s.guildID,
				Type: discordgo.PermissionOverwriteTypeRole,
				Deny: discordgo.PermissionViewChannel,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	return channel, nil
}

// joinVoiceChannel joins the created voice channel.
func (s *OnboardingSession) joinVoiceChannel() error {
	// Use context with timeout for voice join
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	vc, err := s.session.ChannelVoiceJoin(ctx, s.guildID, s.vcChannelID, false, true)
	if err != nil {
		return fmt.Errorf("join voice: %w", err)
	}

	s.voiceConn = vc

	// Wait for voice connection to be ready (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for voice connection to be ready")
		case <-ticker.C:
			if vc.Status == discordgo.VoiceConnectionStatusReady {
				s.logger.Info("joined voice channel successfully", "channel_id", s.vcChannelID)
				return nil
			}
		}
	}
}

// sendWelcomeMessage sends a welcome message with guide selection.
func (s *OnboardingSession) sendWelcomeMessage() error {
	ctx := context.Background()
	title := s.i18n.T(ctx, s.guildID, "onboarding.session_started_title")
	description := s.i18n.TWithArgs(ctx, s.guildID, "onboarding.session_started_description", map[string]string{
		"user": fmt.Sprintf("<@%s>", s.userID),
	})

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x5865F2, // Discord blurple
	}

	// Build guide selection components
	components := s.BuildGuideSelectionComponents()

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// BuildGuideSelectionComponents builds the UI for guide selection (exported for handlers).
func (s *OnboardingSession) BuildGuideSelectionComponents() []discordgo.MessageComponent {
	// TODO: In future, scan audio/ directory for available guides
	// For now, hardcode "kk" as the only available guide
	guides := []string{"kk"}
	ctx := context.Background()

	components := []discordgo.MessageComponent{}

	// Preview buttons (one per guide)
	previewButtons := []discordgo.MessageComponent{}
	for _, guide := range guides {
		guideName := s.i18n.T(ctx, s.guildID, fmt.Sprintf("onboarding.guides.%s.name", guide))
		previewButtons = append(previewButtons, discordgo.Button{
			Label:    guideName + " 🎧",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("onboarding:preview:%s:%s", guide, s.userID),
		})
	}

	components = append(components, discordgo.ActionsRow{
		Components: previewButtons,
	})

	// Dropdown menu for final selection
	options := []discordgo.SelectMenuOption{}
	for _, guide := range guides {
		guideName := s.i18n.T(ctx, s.guildID, fmt.Sprintf("onboarding.guides.%s.name", guide))
		
		options = append(options, discordgo.SelectMenuOption{
			Label: guideName,
			Value: guide,
			Emoji: &discordgo.ComponentEmoji{
				Name: "👤",
			},
		})
	}

	placeholder := s.i18n.T(ctx, s.guildID, "onboarding.choose_guide")
	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    fmt.Sprintf("onboarding:select_guide:%s", s.userID),
				Placeholder: placeholder,
				Options:     options,
			},
		},
	})

	return components
}

// runOnboardingFlow executes the interactive onboarding flow.
func (s *OnboardingSession) runOnboardingFlow() {
	defer s.Complete()

	// Step 1: Play welcome audio and show button
	if err := s.step1Welcome(); err != nil {
		s.logger.Error("step 1 failed", "error", err)
		return
	}

	// Wait for user interaction or timeout
	select {
	case <-s.ctx.Done():
		s.logger.Info("session timed out")
		return
	case <-time.After(2 * time.Minute):
		// Continue to next step after waiting
	}

	// Step 2: Voice selection
	if err := s.step2VoiceSelection(); err != nil {
		s.logger.Error("step 2 failed", "error", err)
		return
	}

	// Wait for completion
	select {
	case <-s.ctx.Done():
		return
	case <-time.After(1 * time.Minute):
		// Auto-complete after demonstration
	}
}

// step1Welcome plays welcome audio and shows initial message.
func (s *OnboardingSession) step1Welcome() error {
	s.UpdateActivity()

	// TODO: This is old flow - will be replaced by guide selection flow
	// Play welcome audio (placeholder)
	// if err := s.playAudioFile("guide_name", "1-intro.dca"); err != nil {
	// 	s.logger.Warn("failed to play welcome audio", "error", err)
	// }

	// Send message with button
	embed := &discordgo.MessageEmbed{
		Title:       "Step 1: Welcome",
		Description: "Welcome to the server! Let's get you set up.",
		Color:       0x3498db,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Continue",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboard:step1:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})

	return err
}

// step2VoiceSelection allows user to select voice preference.
func (s *OnboardingSession) step2VoiceSelection() error {
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Title:       "Step 2: Voice Selection",
		Description: "Which voice would you like to hear?",
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Voice A",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboard:voice:a:%s", s.userID),
				},
				discordgo.Button{
					Label:    "Voice B",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("onboard:voice:b:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})

	return err
}

// GetUserID returns the user ID for this session.
func (s *OnboardingSession) GetUserID() string {
	return s.userID
}

// PlayAudioFile plays an audio file in the voice channel using DCA StreamingSession.
// This is exported so interaction handlers can trigger audio playback.
func (s *OnboardingSession) PlayAudioFile(guide, filename string) error {
	return s.playAudioFile(guide, filename)
}

// playAudioFile plays an audio file in the voice channel using DCA StreamingSession.
// This runs in a goroutine and can be stopped via StopCurrentAudio()
func (s *OnboardingSession) playAudioFile(guide, filename string) error {
	s.UpdateActivity()

	audioPath := fmt.Sprintf("audio/%s/%s", guide, filename)
	s.logger.Info("playing audio", "path", audioPath)

	// Check if file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", audioPath)
	}

	// Check if voice connection is ready
	if s.voiceConn == nil || s.voiceConn.Status != discordgo.VoiceConnectionStatusReady {
		return fmt.Errorf("voice connection not ready")
	}

	// Stop any currently playing audio
	if s.currentStream != nil {
		s.currentStream.SetPaused(true)
		s.currentStream = nil
	}

	// Open DCA file
	file, err := os.Open(audioPath)
	if err != nil {
		return fmt.Errorf("open audio file: %w", err)
	}

	// Create decoder (implements OpusReader interface)
	decoder := dca.NewDecoder(file)
	
	// Create streaming session - this handles sending frames automatically
	done := make(chan error)
	stream := dca.NewStream(decoder, s.voiceConn, done)
	
	// Store stream reference and audio file name
	s.currentStream = stream
	s.currentAudioFile = filename
	
	// Run in goroutine to allow non-blocking playback
	go func() {
		defer file.Close()
		
		// Wait for playback to complete or stop signal
		select {
		case err := <-done:
			if err != nil && err != io.EOF {
				s.logger.Error("playback error", "error", err)
			} else {
				s.logger.Info("audio playback completed", "path", audioPath)
			}
			s.currentStream = nil
		case <-s.stopStream:
			stream.SetPaused(true)
			s.logger.Info("audio playback stopped", "path", audioPath)
			s.currentStream = nil
		case <-s.ctx.Done():
			stream.SetPaused(true)
			s.logger.Info("audio playback cancelled", "path", audioPath)
			s.currentStream = nil
		}
	}()

	return nil
}

// stopAudio stops any currently playing audio.
// Note: With StreamingSession, pausing is handled in playAudioFile via context cancellation
func (s *OnboardingSession) stopAudio() {
	// Context cancellation in playAudioFile will stop the stream
	// This function is kept for compatibility
	s.logger.Info("stop audio requested")
}

// UpdateActivity updates the last activity timestamp.
// This should be called whenever the user interacts with the onboarding session.
func (s *OnboardingSession) UpdateActivity() {
	s.lastActivity = time.Now()
}

// monitorInactivity monitors for user inactivity.
func (s *OnboardingSession) monitorInactivity() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if time.Since(s.lastActivity) > inactivityTimeout {
				s.logger.Info("session inactive, closing")
				s.cancel()
				return
			}
		}
	}
}

// addRole adds a role to the user.
func (s *OnboardingSession) addRole(roleID string) error {
	if roleID == "" {
		return nil
	}

	err := s.session.GuildMemberRoleAdd(s.guildID, s.userID, roleID)
	if err != nil {
		return fmt.Errorf("add role: %w", err)
	}

	s.logger.Info("role added", "role_id", roleID, "user_id", s.userID)
	return nil
}

// removeRole removes a role from the user.
func (s *OnboardingSession) removeRole(roleID string) error {
	if roleID == "" {
		return nil
	}

	err := s.session.GuildMemberRoleRemove(s.guildID, s.userID, roleID)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}

	s.logger.Info("role removed", "role_id", roleID, "user_id", s.userID)
	return nil
}

// Complete completes the onboarding session.
func (s *OnboardingSession) Complete() {
	s.logger.Info("completing onboarding session", "user_id", s.userID)

	// Remove in-progress role and add completed role
	if s.inProgressRoleID != "" {
		if err := s.removeRole(s.inProgressRoleID); err != nil {
			s.logger.Warn("failed to remove in-progress role", "error", err)
		}
	}

	if s.completedRoleID != "" {
		if err := s.addRole(s.completedRoleID); err != nil {
			s.logger.Warn("failed to add completed role", "error", err)
		}
	}

	// Send completion task to master
	completionTask := queue.Task{
		ID:      fmt.Sprintf("complete-%s-%s-%d", s.guildID, s.userID, time.Now().Unix()),
		Type:    "onboarding_complete",
		GuildID: s.guildID,
		Payload: map[string]interface{}{
			"user_id":  s.userID,
			"slave_id": s.slaveID,
		},
		CreatedAt: time.Now(),
	}

	if err := s.queue.Enqueue(context.Background(), completionTask); err != nil {
		s.logger.Error("failed to enqueue completion task", "error", err)
	}

	// Cancel context to trigger Start() to unblock and cleanup
	s.cancel()
}

// saveSessionToCache stores session data in Redis for interaction handlers.
func (s *OnboardingSession) saveSessionToCache() error {
	sessionKey := fmt.Sprintf("welcomebot:session:%s:%s", s.guildID, s.userID)
	
	sessionData := map[string]interface{}{
		"guild_id":       s.guildID,
		"user_id":        s.userID,
		"slave_id":       s.slaveID,
		"vc_channel_id":  s.vcChannelID,
		"selected_guide": s.selectedGuide,
		"current_step":   s.currentStep,
		"started_at":     s.startedAt.Unix(),
	}

	// Store with expiration (session timeout)
	return s.cache.SetJSON(context.Background(), sessionKey, sessionData, sessionTimeout)
}

// cleanup cleans up resources and deletes the voice channel.
func (s *OnboardingSession) cleanup() {
	s.logger.Info("cleaning up session", "user_id", s.userID)

	// Remove session from cache
	sessionKey := fmt.Sprintf("welcomebot:session:%s:%s", s.guildID, s.userID)
	if err := s.cache.Delete(context.Background(), sessionKey); err != nil {
		s.logger.Warn("failed to delete session from cache", "error", err)
	}

	// Disconnect from voice
	if s.voiceConn != nil {
		// Use background context with timeout for cleanup to avoid indefinite hang
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.voiceConn.Disconnect(ctx); err != nil {
			s.logger.Warn("failed to disconnect voice", "error", err)
		}
	}

	// Delete voice channel
	if s.vcChannelID != "" {
		if _, err := s.session.ChannelDelete(s.vcChannelID); err != nil {
			s.logger.Warn("failed to delete voice channel", "error", err)
		}
	}

	// Mark slave as available
	key := fmt.Sprintf("welcomebot:slaves:status:%s", s.slaveID)
	if err := s.cache.Set(context.Background(), key, "available", 30*time.Minute); err != nil {
		s.logger.Warn("failed to mark slave as available", "error", err)
	}

	// Cancel context
	s.cancel()

	s.logger.Info("session cleanup complete")
}

// StartStep1 begins step 1 of the onboarding tutorial.
// This is called after the user confirms their guide selection.
func (s *OnboardingSession) StartStep1(guide string) error {
	s.selectedGuide = guide
	s.currentStep = 1
	s.UpdateActivity()

	// Remove "Entrance" role if configured
	if s.entranceRoleID != "" {
		if err := s.session.GuildMemberRoleRemove(s.guildID, s.userID, s.entranceRoleID); err != nil {
			s.logger.Warn("failed to remove entrance role", "error", err, "role_id", s.entranceRoleID)
		} else {
			s.logger.Info("removed entrance role", "user_id", s.userID, "role_id", s.entranceRoleID)
		}
	}

	// Show Step 1 UI with buttons
	embed := &discordgo.MessageEmbed{
		Title:       s.i18n.T(s.ctx, s.guildID, "onboarding.step1_title"),
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step1_description"),
		Color:       0x3498db,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_next"),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:step1_next:%s", s.userID),
				},
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_replay"),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("onboarding:step1_replay:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send step 1 message: %w", err)
	}

	// Save updated session state
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Send guide image (if available)
	if err := s.sendGuideImage("step1.png"); err != nil {
		s.logger.Warn("failed to send step 1 guide image", "error", err)
		// Don't fail the step if image sending fails
	}

	// Play step 1 intro audio
	if err := s.playAudioFile(guide, "1-intro.dca"); err != nil {
		s.logger.Error("failed to play step 1 audio", "error", err)
		return fmt.Errorf("play step 1 audio: %w", err)
	}

	return nil
}

// StopCurrentAudio stops the currently playing audio.
func (s *OnboardingSession) StopCurrentAudio() {
	if s.currentStream != nil {
		select {
		case s.stopStream <- struct{}{}:
			s.logger.Info("sent stop signal to audio stream")
		default:
			s.logger.Warn("stop channel full, audio may already be stopping")
		}
	}
}

// ReplayCurrentAudio replays the current step's audio from the beginning.
func (s *OnboardingSession) ReplayCurrentAudio() error {
	if s.selectedGuide == "" || s.currentAudioFile == "" {
		return fmt.Errorf("no audio file to replay")
	}

	s.logger.Info("replaying audio", "guide", s.selectedGuide, "file", s.currentAudioFile)
	
	// Stop current playback
	s.StopCurrentAudio()
	
	// Small delay to ensure previous playback stops
	time.Sleep(500 * time.Millisecond)
	
	// Replay the same audio file
	return s.playAudioFile(s.selectedGuide, s.currentAudioFile)
}

// StartStep2 begins step 2 of the onboarding tutorial.
func (s *OnboardingSession) StartStep2() error {
	s.currentStep = 2
	s.UpdateActivity()

	// Add "説明会②" role if configured
	if s.Setsumeikai2RoleID != "" {
		if err := s.session.GuildMemberRoleAdd(s.guildID, s.userID, s.Setsumeikai2RoleID); err != nil {
			s.logger.Warn("failed to add setsumeikai2 role", "error", err, "role_id", s.Setsumeikai2RoleID)
		} else {
			s.logger.Info("added setsumeikai2 role", "user_id", s.userID, "role_id", s.Setsumeikai2RoleID)
		}
	}

	// Remove "入会手続き" role if configured
	if s.nyukaiRoleID != "" {
		if err := s.session.GuildMemberRoleRemove(s.guildID, s.userID, s.nyukaiRoleID); err != nil {
			s.logger.Warn("failed to remove nyukai role", "error", err, "role_id", s.nyukaiRoleID)
		} else {
			s.logger.Info("removed nyukai role", "user_id", s.userID, "role_id", s.nyukaiRoleID)
		}
	}

	// Message 1: First part of text
	part1 := s.i18n.T(s.ctx, s.guildID, "onboarding.step2_description_part1")
	_, err := s.session.ChannelMessageSend(s.vcChannelID, part1)
	if err != nil {
		return fmt.Errorf("send step 2 part 1: %w", err)
	}

	// Message 2: Image
	imagePath := "assets/images/onboarding/step2.png"
	file, err := os.Open(imagePath)
	if err != nil {
		s.logger.Warn("failed to open step 2 image", "error", err, "path", imagePath)
	} else {
		defer file.Close()
		_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
			Files: []*discordgo.File{
				{
					Name:   "step2.png",
					Reader: file,
				},
			},
		})
		if err != nil {
			s.logger.Warn("failed to send step 2 image", "error", err)
		} else {
			s.logger.Info("sent step 2 guide image")
		}
	}

	// Message 3: Second part of text with buttons
	part2 := s.i18n.T(s.ctx, s.guildID, "onboarding.step2_description_part2")
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_next"),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:step2_next:%s", s.userID),
				},
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_replay"),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("onboarding:step2_replay:%s", s.userID),
				},
			},
		},
	}

	_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Content:    part2,
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send step 2 part 2: %w", err)
	}

	// Save updated session state
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Play step 2 profile audio
	if err := s.playAudioFile(s.selectedGuide, "2-profile.dca"); err != nil {
		s.logger.Error("failed to play step 2 audio", "error", err)
		return fmt.Errorf("play step 2 audio: %w", err)
	}

	return nil
}

// StartStep3 begins step 3 of the onboarding tutorial (role selection).
func (s *OnboardingSession) StartStep3() error {
	s.currentStep = 3
	s.currentSubStep = 0 // Reset sub-step
	s.UpdateActivity()

	// Show initial message (plain markdown)
	content := s.i18n.T(s.ctx, s.guildID, "onboarding.step3_description")
	_, err := s.session.ChannelMessageSend(s.vcChannelID, content)
	if err != nil {
		return fmt.Errorf("send step 3 initial message: %w", err)
	}

	// Play step 3 role audio (non-blocking)
	go func() {
		if err := s.playAudioFile(s.selectedGuide, "3-role.dca"); err != nil {
			s.logger.Error("failed to play step 3 audio", "error", err)
		}
	}()

	// Save updated session state
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Immediately show age selection buttons
	return s.ShowAgeSelection()
}

// ShowAgeSelection displays age range selection buttons.
func (s *OnboardingSession) ShowAgeSelection() error {
	s.currentSubStep = 1
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_age_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "20代前半",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:age:20early:%s", s.userID),
				},
				discordgo.Button{
					Label:    "20代後半",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:age:20late:%s", s.userID),
				},
				discordgo.Button{
					Label:    "30代前半",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:age:30early:%s", s.userID),
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "30代後半",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:age:30late:%s", s.userID),
				},
				discordgo.Button{
					Label:    "40代前半",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:age:40early:%s", s.userID),
				},
				discordgo.Button{
					Label:    "40代後半",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:age:40late:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send age selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowVoiceTypeSelection displays voice type selection buttons.
func (s *OnboardingSession) ShowVoiceTypeSelection() error {
	s.currentSubStep = 2
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_voice_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "高音",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:voice:high:%s", s.userID),
				},
				discordgo.Button{
					Label:    "中高音",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:voice:midhigh:%s", s.userID),
				},
				discordgo.Button{
					Label:    "中音",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:voice:mid:%s", s.userID),
				},
				discordgo.Button{
					Label:    "中低音",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:voice:midlow:%s", s.userID),
				},
				discordgo.Button{
					Label:    "低音",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:voice:low:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send voice selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowEroipuSelection displays eroipu OK/NG buttons.
func (s *OnboardingSession) ShowEroipuSelection() error {
	s.currentSubStep = 3
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_eroipu_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "エロイプOK",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("onboarding:eroipu:ok:%s", s.userID),
				},
				discordgo.Button{
					Label:    "エロイプNG",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("onboarding:eroipu:ng:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send eroipu selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowNeochiOkNgSelection displays neochi OK/NG buttons.
func (s *OnboardingSession) ShowNeochiOkNgSelection() error {
	s.currentSubStep = 4
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_neochi_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "寝落ちOK",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("onboarding:neochi:ok:%s", s.userID),
				},
				discordgo.Button{
					Label:    "寝落ちNG",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("onboarding:neochi:ng:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send neochi ok/ng selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowNeochiHandlingSelection displays neochi handling buttons.
func (s *OnboardingSession) ShowNeochiHandlingSelection() error {
	s.currentSubStep = 5
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_neochi_handling_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "寝落ち部屋",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:neochi_handling:room:%s", s.userID),
				},
				discordgo.Button{
					Label:    "寝落ち切断",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:neochi_handling:disconnect:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send neochi handling selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowDMSelection displays DM OK/NG buttons.
func (s *OnboardingSession) ShowDMSelection() error {
	s.currentSubStep = 6
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_dm_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "DMOK",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("onboarding:dm:ok:%s", s.userID),
				},
				discordgo.Button{
					Label:    "DMNG",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("onboarding:dm:ng:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send dm selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowFriendSelection displays friend OK/NG buttons.
func (s *OnboardingSession) ShowFriendSelection() error {
	s.currentSubStep = 7
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_friend_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "フレンド OK",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("onboarding:friend:ok:%s", s.userID),
				},
				discordgo.Button{
					Label:    "フレンド NG",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("onboarding:friend:ng:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send friend selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowEventSelection displays event role buttons (users can select both).
func (s *OnboardingSession) ShowEventSelection() error {
	s.currentSubStep = 8
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_event_prompt"),
		Color:       0x9b59b6,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "BunnyClub イベント",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:event:bunnyclub:%s", s.userID),
				},
				discordgo.Button{
					Label:    "ユーザーイベント",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:event:user:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send event selection: %w", err)
	}

	return s.saveSessionToCache()
}

// ShowStep3Completion shows the final message of step 3 with next button.
func (s *OnboardingSession) ShowStep3Completion() error {
	s.currentSubStep = 9
	s.UpdateActivity()

	embed := &discordgo.MessageEmbed{
		Description: s.i18n.T(s.ctx, s.guildID, "onboarding.step3_completion"),
		Color:       0x2ecc71,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_next"),
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("onboarding:step3_next:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send step 3 completion: %w", err)
	}

	return s.saveSessionToCache()
}

// StartStep4 begins step 4 of the onboarding tutorial.
func (s *OnboardingSession) StartStep4() error {
	s.currentStep = 4
	s.UpdateActivity()

	// Message 1: First part of text
	part1 := s.i18n.T(s.ctx, s.guildID, "onboarding.step4_description_part1")
	_, err := s.session.ChannelMessageSend(s.vcChannelID, part1)
	if err != nil {
		return fmt.Errorf("send step 4 part 1: %w", err)
	}

	// Message 2: Image
	imagePath := "assets/images/onboarding/step4.png"
	file, err := os.Open(imagePath)
	if err != nil {
		s.logger.Warn("failed to open step 4 image", "error", err, "path", imagePath)
	} else {
		defer file.Close()
		_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
			Files: []*discordgo.File{
				{
					Name:   "step4.png",
					Reader: file,
				},
			},
		})
		if err != nil {
			s.logger.Warn("failed to send step 4 image", "error", err)
		} else {
			s.logger.Info("sent step 4 guide image")
		}
	}

	// Message 3: Second part of text with buttons
	part2 := s.i18n.T(s.ctx, s.guildID, "onboarding.step4_description_part2")
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_next"),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:step4_next:%s", s.userID),
				},
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_replay"),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("onboarding:step4_replay:%s", s.userID),
				},
			},
		},
	}

	_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Content:    part2,
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send step 4 part 2: %w", err)
	}

	// Save updated session state
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Play step 4 point audio
	if err := s.playAudioFile(s.selectedGuide, "4-point.dca"); err != nil {
		s.logger.Error("failed to play step 4 audio", "error", err)
		return fmt.Errorf("play step 4 audio: %w", err)
	}

	return nil
}

// StartStep5 begins step 5 of the onboarding tutorial.
func (s *OnboardingSession) StartStep5() error {
	s.currentStep = 5
	s.UpdateActivity()

	// Send plain markdown message with buttons
	content := s.i18n.T(s.ctx, s.guildID, "onboarding.step5_description")
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_next"),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:step5_next:%s", s.userID),
				},
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_replay"),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("onboarding:step5_replay:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Content:    content,
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send step 5 message: %w", err)
	}

	// Save updated session state
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Play step 5 club audio
	if err := s.playAudioFile(s.selectedGuide, "5-club.dca"); err != nil {
		s.logger.Error("failed to play step 5 audio", "error", err)
		return fmt.Errorf("play step 5 audio: %w", err)
	}

	return nil
}

// StartStep6 begins step 6 of the onboarding tutorial.
func (s *OnboardingSession) StartStep6() error {
	s.currentStep = 6
	s.UpdateActivity()

	// Message 1: First part of text
	part1 := s.i18n.T(s.ctx, s.guildID, "onboarding.step6_description_part1")
	_, err := s.session.ChannelMessageSend(s.vcChannelID, part1)
	if err != nil {
		return fmt.Errorf("send step 6 part 1: %w", err)
	}

	// Message 2: First image
	imagePath1 := "assets/images/onboarding/step6-1.png"
	file1, err := os.Open(imagePath1)
	if err != nil {
		s.logger.Warn("failed to open step 6 image 1", "error", err, "path", imagePath1)
	} else {
		defer file1.Close()
		_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
			Files: []*discordgo.File{
				{
					Name:   "step6-1.png",
					Reader: file1,
				},
			},
		})
		if err != nil {
			s.logger.Warn("failed to send step 6 image 1", "error", err)
		} else {
			s.logger.Info("sent step 6 guide image 1")
		}
	}

	// Message 3: Second part of text
	part2 := s.i18n.T(s.ctx, s.guildID, "onboarding.step6_description_part2")
	_, err = s.session.ChannelMessageSend(s.vcChannelID, part2)
	if err != nil {
		return fmt.Errorf("send step 6 part 2: %w", err)
	}

	// Message 4: Second image
	imagePath2 := "assets/images/onboarding/step6-2.png"
	file2, err := os.Open(imagePath2)
	if err != nil {
		s.logger.Warn("failed to open step 6 image 2", "error", err, "path", imagePath2)
	} else {
		defer file2.Close()
		_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
			Files: []*discordgo.File{
				{
					Name:   "step6-2.png",
					Reader: file2,
				},
			},
		})
		if err != nil {
			s.logger.Warn("failed to send step 6 image 2", "error", err)
		} else {
			s.logger.Info("sent step 6 guide image 2")
		}
	}

	// Message 5: Buttons
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_next"),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("onboarding:step6_next:%s", s.userID),
				},
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_replay"),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("onboarding:step6_replay:%s", s.userID),
				},
			},
		},
	}

	_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send step 6 buttons: %w", err)
	}

	// Save updated session state
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Play step 6 membership audio
	if err := s.playAudioFile(s.selectedGuide, "6-membership.dca"); err != nil {
		s.logger.Error("failed to play step 6 audio", "error", err)
		return fmt.Errorf("play step 6 audio: %w", err)
	}

	return nil
}

// StartStep7 begins step 7 of the onboarding tutorial (final step).
func (s *OnboardingSession) StartStep7() error {
	s.currentStep = 7
	s.UpdateActivity()

	// Send plain markdown message with buttons
	content := s.i18n.T(s.ctx, s.guildID, "onboarding.step7_description")
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_complete"),
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("onboarding:step7_complete:%s", s.userID),
				},
				discordgo.Button{
					Label:    s.i18n.T(s.ctx, s.guildID, "onboarding.button_replay"),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("onboarding:step7_replay:%s", s.userID),
				},
			},
		},
	}

	_, err := s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Content:    content,
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send step 7 message: %w", err)
	}

	// Save updated session state
	if err := s.saveSessionToCache(); err != nil {
		s.logger.Warn("failed to save session to cache", "error", err)
	}

	// Play step 7 end audio
	if err := s.playAudioFile(s.selectedGuide, "7-end.dca"); err != nil {
		s.logger.Error("failed to play step 7 audio", "error", err)
		return fmt.Errorf("play step 7 audio: %w", err)
	}

	return nil
}

// sendGuideImage sends a guide image to the voice channel.
// This is a helper method to send images from the assets/images/onboarding directory.
func (s *OnboardingSession) sendGuideImage(filename string) error {
	imagePath := fmt.Sprintf("assets/images/onboarding/%s", filename)
	s.logger.Info("sending guide image", "path", imagePath)

	// Check if file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		s.logger.Warn("guide image not found", "path", imagePath)
		return nil // Don't fail the step if image is missing
	}

	// Open image file
	file, err := os.Open(imagePath)
	if err != nil {
		s.logger.Error("failed to open guide image", "error", err, "path", imagePath)
		return nil // Don't fail the step if image can't be opened
	}
	defer file.Close()

	// Send image as attachment
	_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
		Files: []*discordgo.File{
			{
				Name:   filename,
				Reader: file,
			},
		},
	})
	if err != nil {
		s.logger.Error("failed to send guide image", "error", err, "path", imagePath)
		return nil // Don't fail the step if image send fails
	}

	s.logger.Info("guide image sent successfully", "path", imagePath)
	return nil
}

