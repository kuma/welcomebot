package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handlePreviewButton handles guide preview button clicks.
func (w *Worker) handlePreviewButton(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract guide name from customID: onboarding:preview:{guide}:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid preview customID", "custom_id", customID)
		return
	}

	guide := parts[2]
	userID := parts[3]

	// Verify user is the one who started onboarding
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Acknowledge interaction
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Get session from cache
	sessionKey := fmt.Sprintf("welcomebot:session:%s:%s", i.GuildID, userID)
	var sessionData map[string]interface{}
	if err := w.cache.GetJSON(ctx, sessionKey, &sessionData); err != nil {
		w.logger.Error("session not found", "error", err)
		return
	}

	vcChannelID, _ := sessionData["vc_channel_id"].(string)
	if vcChannelID == "" {
		w.logger.Error("vc_channel_id not found in session")
		return
	}

	w.logger.Info("preview button clicked", "guide", guide, "user_id", userID, "vc_channel_id", vcChannelID)
	
	// Get the active session
	sessionKey = fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found", "session_key", sessionKey)
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Send a message in the VC to indicate audio is playing
	previewMessage := w.i18n.T(ctx, i.GuildID, "onboarding.preview_playing")
	_, err = s.ChannelMessageSend(vcChannelID, previewMessage)
	if err != nil {
		w.logger.Warn("failed to send preview message", "error", err)
	}

	// Play the preview audio (0-voice-select.dca)
	go func() {
		if err := activeSession.PlayAudioFile(guide, "0-voice-select.dca"); err != nil {
			w.logger.Error("failed to play preview audio", "error", err)
		}
	}()
}

// handleGuideSelection handles guide dropdown selection.
func (w *Worker) handleGuideSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:select_guide:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid select_guide customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This selection is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get selected guide
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		w.logger.Error("no guide selected")
		return
	}

	selectedGuide := values[0]

	// Update session with selected guide
	sessionKey := fmt.Sprintf("welcomebot:session:%s:%s", i.GuildID, userID)
	var sessionData map[string]interface{}
	if err := w.cache.GetJSON(ctx, sessionKey, &sessionData); err != nil {
		w.logger.Error("session not found", "error", err)
		return
	}

	sessionData["selected_guide"] = selectedGuide
	sessionData["current_step"] = 0 // Still at step 0 (confirmation pending)

	if err := w.cache.SetJSON(ctx, sessionKey, sessionData, 10*time.Minute); err != nil {
		w.logger.Error("failed to update session", "error", err)
		return
	}

	// Respond with confirmation prompt
	guideName := w.i18n.T(ctx, i.GuildID, fmt.Sprintf("onboarding.guides.%s.name", selectedGuide))
	confirmationText := w.i18n.TWithArgs(ctx, i.GuildID, "onboarding.guide_selected", map[string]string{
		"guide": guideName,
	})

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: confirmationText,
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    w.i18n.T(ctx, i.GuildID, "onboarding.confirm_guide"),
							Style:    discordgo.SuccessButton,
							CustomID: fmt.Sprintf("onboarding:confirm_guide:%s:%s", selectedGuide, userID),
						},
						discordgo.Button{
							Label:    w.i18n.T(ctx, i.GuildID, "onboarding.button_back"),
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("onboarding:back_to_guide_selection:%s", userID),
						},
					},
				},
			},
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	w.logger.Info("guide selected, awaiting confirmation", "guide", selectedGuide, "user_id", userID)
}

// handleGuideConfirmation handles the confirmation button after guide selection.
func (w *Worker) handleGuideConfirmation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract guide and userID from customID: onboarding:confirm_guide:{guide}:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid confirm_guide customID", "custom_id", customID)
		return
	}

	guide := parts[2]
	userID := parts[3]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Acknowledge the button click
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    w.i18n.T(ctx, i.GuildID, "onboarding.starting_tutorial"),
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{}, // Clear button
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Get the active session and start step 1
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for guide confirmation", "session_key", sessionKey)
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	w.logger.Info("guide confirmed, starting tutorial", "guide", guide, "user_id", userID)

	// Start step 1 of the tutorial
	go func() {
		// Small delay to let the user see the confirmation message
		time.Sleep(1 * time.Second)

		// Start Step 1 (removes entrance role, shows UI, plays audio)
		if err := activeSession.StartStep1(guide); err != nil {
			w.logger.Error("failed to start step 1", "error", err)
			return
		}

		w.logger.Info("step 1 started", "guide", guide)
	}()
}

// handleBackToGuideSelection handles the [戻る] (Back) button click from guide confirmation.
func (w *Worker) handleBackToGuideSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:back_to_guide_selection:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid back_to_guide_selection customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get the active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for back to guide selection", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Rebuild the guide selection UI
	title := w.i18n.T(ctx, i.GuildID, "onboarding.session_started_title")
	description := w.i18n.TWithArgs(ctx, i.GuildID, "onboarding.session_started_description", map[string]string{
		"user": fmt.Sprintf("<@%s>", userID),
	})

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x5865F2, // Discord blurple
	}

	// Rebuild guide selection components using the session's method
	components := activeSession.BuildGuideSelectionComponents()

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "",
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	w.logger.Info("user went back to guide selection", "user_id", userID)
}

// handleStep1Next handles the [次へ] (Next) button click in Step 1.
func (w *Worker) handleStep1Next(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step1_next:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step1_next customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step1 next", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Stop current audio
	activeSession.StopCurrentAudio()

	// Acknowledge button click
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    w.i18n.T(ctx, i.GuildID, "onboarding.moving_to_step2"),
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{}, // Clear buttons
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	w.logger.Info("user clicked next, moving to step 2", "user_id", userID)
	
	// Start Step 2
	go func() {
		// Small delay to let the user see the transition message
		time.Sleep(1 * time.Second)

		if err := activeSession.StartStep2(); err != nil {
			w.logger.Error("failed to start step 2", "error", err)
			return
		}

		w.logger.Info("step 2 started", "user_id", userID)
	}()
}

// handleStep1Replay handles the [もう一度聞く] (Play Again) button click in Step 1.
func (w *Worker) handleStep1Replay(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step1_replay:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step1_replay customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step1 replay", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Acknowledge button click (but keep the same UI)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Replay the audio
	if err := activeSession.ReplayCurrentAudio(); err != nil {
		w.logger.Error("failed to replay audio", "error", err)
		return
	}

	w.logger.Info("replaying step 1 audio", "user_id", userID)
}

// handleStep2Next handles the [次へ] (Next) button click in Step 2.
func (w *Worker) handleStep2Next(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step2_next:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step2_next customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step2 next", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Stop current audio
	activeSession.StopCurrentAudio()

	// Check if user already has 説明会③ role (skip Step 3 if they do)
	skipStep3 := false
	if activeSession.Setsumeikai3RoleID != "" {
		// Get guild member to check roles
		member, err := s.GuildMember(i.GuildID, userID)
		if err == nil {
			// Check if user has the setsumeikai3 role
			for _, roleID := range member.Roles {
				if roleID == activeSession.Setsumeikai3RoleID {
					skipStep3 = true
					w.logger.Info("user already has setsumeikai3 role, skipping step 3", "user_id", userID)
					break
				}
			}
		}
	}

	// Acknowledge button click
	var responseContent string
	if skipStep3 {
		responseContent = "⏭️ ステップ4に進んでいます..."
	} else {
		responseContent = w.i18n.T(ctx, i.GuildID, "onboarding.moving_to_step3")
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    responseContent,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{}, // Clear buttons
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	if skipStep3 {
		w.logger.Info("skipping step 3, moving directly to step 4", "user_id", userID)
		
		// Start Step 4
		if err := activeSession.StartStep4(); err != nil {
			w.logger.Error("failed to start step 4", "error", err)
			return
		}
	} else {
		w.logger.Info("user clicked next, moving to step 3", "user_id", userID)
		
		// Start Step 3
		if err := activeSession.StartStep3(); err != nil {
			w.logger.Error("failed to start step 3", "error", err)
			return
		}
	}
}

// handleStep2Replay handles the [もう一度聞く] (Play Again) button click in Step 2.
func (w *Worker) handleStep2Replay(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step2_replay:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step2_replay customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step2 replay", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Acknowledge button click (but keep the same UI)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Replay the audio
	if err := activeSession.ReplayCurrentAudio(); err != nil {
		w.logger.Error("failed to replay audio", "error", err)
		return
	}

	w.logger.Info("replaying step 2 audio", "user_id", userID)
}

// handleStep3AgeSelection handles age range button clicks in step 3.
func (w *Worker) handleStep3AgeSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract age type and userID from customID: onboarding:age:{ageType}:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid age customID", "custom_id", customID)
		return
	}

	ageType := parts[2]
	userID := parts[3]

	// Verify user is the one who started onboarding
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Map age type to role ID
	var roleID string
	var roleName string
	switch ageType {
	case "20early":
		roleID = activeSession.Age20EarlyRoleID
		roleName = "20代前半"
	case "20late":
		roleID = activeSession.Age20LateRoleID
		roleName = "20代後半"
	case "30early":
		roleID = activeSession.Age30EarlyRoleID
		roleName = "30代前半"
	case "30late":
		roleID = activeSession.Age30LateRoleID
		roleName = "30代後半"
	case "40early":
		roleID = activeSession.Age40EarlyRoleID
		roleName = "40代前半"
	case "40late":
		roleID = activeSession.Age40LateRoleID
		roleName = "40代後半"
	}

	// Assign role if configured
	if roleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
			w.logger.Error("failed to add age role", "error", err, "role_id", roleID)
		}
	}

	// Acknowledge interaction with confirmation
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s のロールを付与しました", roleName),
		},
	})

	// Wait before showing next selection
	time.Sleep(1500 * time.Millisecond)

	// Show voice type selection
	if err := activeSession.ShowVoiceTypeSelection(); err != nil {
		w.logger.Error("failed to show voice selection", "error", err)
	}
}

// handleStep3VoiceSelection handles voice type button clicks in step 3.
func (w *Worker) handleStep3VoiceSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid voice customID", "custom_id", customID)
		return
	}

	voiceType := parts[2]
	userID := parts[3]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	var roleID string
	var roleName string
	switch voiceType {
	case "high":
		roleID = activeSession.HighVoiceRoleID
		roleName = "高音"
	case "midhigh":
		roleID = activeSession.MidHighVoiceRoleID
		roleName = "中高音"
	case "mid":
		roleID = activeSession.MidVoiceRoleID
		roleName = "中音"
	case "midlow":
		roleID = activeSession.MidLowVoiceRoleID
		roleName = "中低音"
	case "low":
		roleID = activeSession.LowVoiceRoleID
		roleName = "低音"
	}

	if roleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
			w.logger.Error("failed to add voice role", "error", err, "role_id", roleID)
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s のロールを付与しました", roleName),
		},
	})

	time.Sleep(1500 * time.Millisecond)

	if err := activeSession.ShowEroipuSelection(); err != nil {
		w.logger.Error("failed to show eroipu selection", "error", err)
	}
}

// handleStep3EroipuSelection handles eroipu OK/NG button clicks.
func (w *Worker) handleStep3EroipuSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid eroipu customID", "custom_id", customID)
		return
	}

	choice := parts[2]
	userID := parts[3]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	var roleID string
	var roleName string
	switch choice {
	case "ok":
		roleID = activeSession.EroOkRoleID
		roleName = "エロイプOK"
	case "ng":
		roleID = activeSession.EroNgRoleID
		roleName = "エロイプNG"
	}

	if roleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
			w.logger.Error("failed to add eroipu role", "error", err, "role_id", roleID)
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s のロールを付与しました", roleName),
		},
	})

	time.Sleep(1500 * time.Millisecond)

	if err := activeSession.ShowNeochiOkNgSelection(); err != nil {
		w.logger.Error("failed to show neochi selection", "error", err)
	}
}

// handleStep3NeochiOkNgSelection handles neochi OK/NG button clicks.
func (w *Worker) handleStep3NeochiOkNgSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid neochi customID", "custom_id", customID)
		return
	}

	choice := parts[2]
	userID := parts[3]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	var roleID string
	var roleName string
	switch choice {
	case "ok":
		roleID = activeSession.NeochiOkRoleID
		roleName = "寝落ちOK"
	case "ng":
		roleID = activeSession.NeochiNgRoleID
		roleName = "寝落ちNG"
	}

	if roleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
			w.logger.Error("failed to add neochi role", "error", err, "role_id", roleID)
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s のロールを付与しました", roleName),
		},
	})

	time.Sleep(1500 * time.Millisecond)

	if err := activeSession.ShowNeochiHandlingSelection(); err != nil {
		w.logger.Error("failed to show neochi handling selection", "error", err)
	}
}

// handleStep3NeochiHandlingSelection handles neochi handling button clicks.
func (w *Worker) handleStep3NeochiHandlingSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid neochi_handling customID", "custom_id", customID)
		return
	}

	choice := parts[2]
	userID := parts[3]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	var roleName string
	if choice == "disconnect" {
		// Give disconnect role
		if activeSession.NeochiDisconnectRoleID != "" {
			if err := s.GuildMemberRoleAdd(i.GuildID, userID, activeSession.NeochiDisconnectRoleID); err != nil {
				w.logger.Error("failed to add neochi disconnect role", "error", err)
			}
		}
		roleName = "寝落ち切断"
	} else {
		// Room choice - no role given
		roleName = "寝落ち部屋"
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s を選択しました", roleName),
		},
	})

	time.Sleep(1500 * time.Millisecond)

	if err := activeSession.ShowDMSelection(); err != nil {
		w.logger.Error("failed to show DM selection", "error", err)
	}
}

// handleStep3DMSelection handles DM OK/NG button clicks.
func (w *Worker) handleStep3DMSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid dm customID", "custom_id", customID)
		return
	}

	choice := parts[2]
	userID := parts[3]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	var roleID string
	var roleName string
	switch choice {
	case "ok":
		roleID = activeSession.DmOkRoleID
		roleName = "DMOK"
	case "ng":
		roleID = activeSession.DmNgRoleID
		roleName = "DMNG"
	}

	if roleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
			w.logger.Error("failed to add dm role", "error", err, "role_id", roleID)
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s のロールを付与しました", roleName),
		},
	})

	time.Sleep(1500 * time.Millisecond)

	if err := activeSession.ShowFriendSelection(); err != nil {
		w.logger.Error("failed to show friend selection", "error", err)
	}
}

// handleStep3FriendSelection handles friend OK/NG button clicks.
func (w *Worker) handleStep3FriendSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid friend customID", "custom_id", customID)
		return
	}

	choice := parts[2]
	userID := parts[3]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	var roleID string
	var roleName string
	switch choice {
	case "ok":
		roleID = activeSession.FriendOkRoleID
		roleName = "フレンド OK"
	case "ng":
		roleID = activeSession.FriendNgRoleID
		roleName = "フレンド NG"
	}

	if roleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
			w.logger.Error("failed to add friend role", "error", err, "role_id", roleID)
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s のロールを付与しました", roleName),
		},
	})

	time.Sleep(1500 * time.Millisecond)

	if err := activeSession.ShowEventSelection(); err != nil {
		w.logger.Error("failed to show event selection", "error", err)
	}
}

// handleStep3EventSelection handles event role button clicks (users can select both).
func (w *Worker) handleStep3EventSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		w.logger.Error("invalid event customID", "custom_id", customID)
		return
	}

	eventType := parts[2]
	userID := parts[3]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	var roleID string
	var roleName string
	switch eventType {
	case "bunnyclub":
		roleID = activeSession.BunnyclubEventRoleID
		roleName = "BunnyClub イベント"
	case "user":
		roleID = activeSession.UserEventRoleID
		roleName = "ユーザーイベント"
	}

	if roleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
			w.logger.Error("failed to add event role", "error", err, "role_id", roleID)
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s のロールを付与しました", roleName),
		},
	})

	// Don't auto-progress since users can select both event roles
	// They'll need to wait for both selections to complete, then we show completion
	// To handle this, we check if this is the first or second event role selection
	// For simplicity, we'll show completion after a delay
	go func() {
		time.Sleep(2 * time.Second)
		if err := activeSession.ShowStep3Completion(); err != nil {
			w.logger.Error("failed to show step 3 completion", "error", err)
		}
	}()
}

// handleStep3Next handles the next button at the end of step 3.
func (w *Worker) handleStep3Next(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step3_next customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This button is not for you!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: アクティブなセッションが見つかりません",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Acknowledge interaction
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ステップ4に進んでいます...",
		},
	})

	// Add "説明会③" role if configured
	if activeSession.Setsumeikai3RoleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, activeSession.Setsumeikai3RoleID); err != nil {
			w.logger.Warn("failed to add setsumeikai3 role", "error", err, "role_id", activeSession.Setsumeikai3RoleID)
		} else {
			w.logger.Info("added setsumeikai3 role", "user_id", userID, "role_id", activeSession.Setsumeikai3RoleID)
		}
	}

	// Start Step 4
	if err := activeSession.StartStep4(); err != nil {
		w.logger.Error("failed to start step 4", "error", err)
		return
	}
	
	w.logger.Info("step 3 completed, moving to step 4", "user_id", userID)
}

// handleStep4Next handles the [次へ] (Next) button click in Step 4.
func (w *Worker) handleStep4Next(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step4_next:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step4_next customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step4 next", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Stop current audio
	activeSession.StopCurrentAudio()

	// Acknowledge button click
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "⏭️ ステップ5へ移動中...",
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{}, // Clear buttons
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	w.logger.Info("user clicked next, moving to step 5", "user_id", userID)
	
	// Start Step 5
	if err := activeSession.StartStep5(); err != nil {
		w.logger.Error("failed to start step 5", "error", err)
		return
	}
}

// handleStep4Replay handles the [もう一度聞く] (Play Again) button click in Step 4.
func (w *Worker) handleStep4Replay(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step4_replay:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step4_replay customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step4 replay", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Acknowledge button click (but keep the same UI)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Replay the audio
	if err := activeSession.ReplayCurrentAudio(); err != nil {
		w.logger.Error("failed to replay audio", "error", err)
		return
	}

	w.logger.Info("replaying step 4 audio", "user_id", userID)
}

// handleStep5Next handles the [次へ] (Next) button click in Step 5.
func (w *Worker) handleStep5Next(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step5_next:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step5_next customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step5 next", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Stop current audio
	activeSession.StopCurrentAudio()

	// Acknowledge button click
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "⏭️ ステップ6へ移動中...",
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{}, // Clear buttons
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	w.logger.Info("user clicked next, moving to step 6", "user_id", userID)
	
	// Start Step 6
	if err := activeSession.StartStep6(); err != nil {
		w.logger.Error("failed to start step 6", "error", err)
		return
	}
}

// handleStep5Replay handles the [もう一度聞く] (Play Again) button click in Step 5.
func (w *Worker) handleStep5Replay(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step5_replay:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step5_replay customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step5 replay", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Acknowledge button click (but keep the same UI)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Replay the audio
	if err := activeSession.ReplayCurrentAudio(); err != nil {
		w.logger.Error("failed to replay audio", "error", err)
		return
	}

	w.logger.Info("replaying step 5 audio", "user_id", userID)
}

// handleStep6Next handles the [次へ] (Next) button click in Step 6.
func (w *Worker) handleStep6Next(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step6_next:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step6_next customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step6 next", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Stop current audio
	activeSession.StopCurrentAudio()

	// Acknowledge button click
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "⏭️ ステップ7へ移動中...",
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{}, // Clear buttons
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	w.logger.Info("user clicked next, moving to step 7", "user_id", userID)
	
	// Start Step 7
	if err := activeSession.StartStep7(); err != nil {
		w.logger.Error("failed to start step 7", "error", err)
		return
	}
}

// handleStep6Replay handles the [もう一度聞く] (Play Again) button click in Step 6.
func (w *Worker) handleStep6Replay(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step6_replay:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step6_replay customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step6 replay", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Acknowledge button click (but keep the same UI)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Replay the audio
	if err := activeSession.ReplayCurrentAudio(); err != nil {
		w.logger.Error("failed to replay audio", "error", err)
		return
	}

	w.logger.Info("replaying step 6 audio", "user_id", userID)
}

// handleStep7Complete handles the [BunnyClubへ] (Complete) button click in Step 7.
func (w *Worker) handleStep7Complete(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step7_complete:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step7_complete customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step7 complete", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Stop current audio
	activeSession.StopCurrentAudio()

	// Acknowledge button click
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "🎉 説明会完了！BunnyClubへようこそ！",
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{}, // Clear buttons
		},
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	w.logger.Info("user completed onboarding, applying final roles", "user_id", userID)

	// Add "visitor" role
	if activeSession.VisitorRoleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, activeSession.VisitorRoleID); err != nil {
			w.logger.Error("failed to add visitor role", "error", err, "role_id", activeSession.VisitorRoleID)
		} else {
			w.logger.Info("added visitor role", "user_id", userID, "role_id", activeSession.VisitorRoleID)
		}
	}

	// Add "会員" (member) role
	if activeSession.MemberRoleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, activeSession.MemberRoleID); err != nil {
			w.logger.Error("failed to add member role", "error", err, "role_id", activeSession.MemberRoleID)
		} else {
			w.logger.Info("added member role", "user_id", userID, "role_id", activeSession.MemberRoleID)
		}
	}

	// Remove "説明会" role (setsumeikai1)
	if activeSession.Setsumeikai1RoleID != "" {
		if err := s.GuildMemberRoleRemove(i.GuildID, userID, activeSession.Setsumeikai1RoleID); err != nil {
			w.logger.Error("failed to remove setsumeikai1 role", "error", err, "role_id", activeSession.Setsumeikai1RoleID)
		} else {
			w.logger.Info("removed setsumeikai1 role", "user_id", userID, "role_id", activeSession.Setsumeikai1RoleID)
		}
	}

	// Remove "説明会②" role (setsumeikai2)
	if activeSession.Setsumeikai2RoleID != "" {
		if err := s.GuildMemberRoleRemove(i.GuildID, userID, activeSession.Setsumeikai2RoleID); err != nil {
			w.logger.Error("failed to remove setsumeikai2 role", "error", err, "role_id", activeSession.Setsumeikai2RoleID)
		} else {
			w.logger.Info("removed setsumeikai2 role", "user_id", userID, "role_id", activeSession.Setsumeikai2RoleID)
		}
	}

	// Remove "説明会③" role (setsumeikai3)
	if activeSession.Setsumeikai3RoleID != "" {
		if err := s.GuildMemberRoleRemove(i.GuildID, userID, activeSession.Setsumeikai3RoleID); err != nil {
			w.logger.Error("failed to remove setsumeikai3 role", "error", err, "role_id", activeSession.Setsumeikai3RoleID)
		} else {
			w.logger.Info("removed setsumeikai3 role", "user_id", userID, "role_id", activeSession.Setsumeikai3RoleID)
		}
	}

	// Complete the session (this will delete the VC and cleanup)
	activeSession.Complete()

	// Remove from active sessions map
	w.sessionsMutex.Lock()
	delete(w.activeSessions, sessionKey)
	w.sessionsMutex.Unlock()

	w.logger.Info("onboarding completed successfully", "user_id", userID)
}

// handleStep7Replay handles the [もう一度聞く] (Play Again) button click in Step 7.
func (w *Worker) handleStep7Replay(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Extract userID from customID: onboarding:step7_replay:{userID}
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		w.logger.Error("invalid step7_replay customID", "custom_id", customID)
		return
	}

	userID := parts[2]

	// Verify user
	if i.Member.User.ID != userID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.not_your_button"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get active session
	sessionKey := fmt.Sprintf("%s:%s", i.GuildID, userID)
	w.sessionsMutex.RLock()
	activeSession, exists := w.activeSessions[sessionKey]
	w.sessionsMutex.RUnlock()

	if !exists {
		w.logger.Error("active session not found for step7 replay", "session_key", sessionKey)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: w.i18n.T(ctx, i.GuildID, "onboarding.session_not_found"),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update activity timestamp
	activeSession.UpdateActivity()

	// Acknowledge button click (but keep the same UI)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		w.logger.Error("failed to respond to interaction", "error", err)
		return
	}

	// Replay the audio
	if err := activeSession.ReplayCurrentAudio(); err != nil {
		w.logger.Error("failed to replay audio", "error", err)
		return
	}

	w.logger.Info("replaying step 7 audio", "user_id", userID)
}

