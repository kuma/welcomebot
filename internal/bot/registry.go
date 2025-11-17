package bot

import (
	"context"
	"errors"
	"fmt"

	"welcomebot/internal/core/logger"

	"github.com/bwmarrin/discordgo"
)

// Registry manages all bot features.
type Registry struct {
	features    map[string]Feature
	logger      logger.Logger
	eventRouter *EventRouter
}

// NewRegistry creates a new feature registry.
func NewRegistry(log logger.Logger) *Registry {
	return &Registry{
		features:    make(map[string]Feature),
		logger:      log,
		eventRouter: NewEventRouter(log),
	}
}

// EventRouter returns the event router.
func (r *Registry) EventRouter() *EventRouter {
	return r.eventRouter
}

// Register adds a feature to the registry.
func (r *Registry) Register(feature Feature) error {
	if feature == nil {
		return errors.New("feature cannot be nil")
	}

	name := feature.Name()
	if name == "" {
		return errors.New("feature name cannot be empty")
	}

	if _, exists := r.features[name]; exists {
		return fmt.Errorf("feature %s already registered", name)
	}

	r.features[name] = feature
	r.logger.Info("feature registered", "name", name)
	return nil
}

// HandleInteraction routes interactions to the appropriate feature.
func (r *Registry) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	commandName := r.extractCommandName(i)
	if commandName == "" {
		r.logger.Warn("cannot extract command name", "type", i.Type)
		return
	}

	// Try each feature until one handles it
	for name, feature := range r.features {
		if err := feature.HandleInteraction(ctx, s, i); err == nil {
			return // Feature handled it successfully
		} else if !errors.Is(err, ErrNotHandled) {
			r.logger.Error("feature error handling interaction",
				"feature", name,
				"command", commandName,
				"error", err,
			)
			return
		}
	}

	r.logger.Debug("no feature handled interaction", "command", commandName)
}

// HandleMessage routes messages using hybrid approach.
func (r *Registry) HandleMessage(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) {
	// 1. Route to indexed handlers (high-frequency)
	r.eventRouter.RouteMessageCreate(ctx, s, m)

	// 2. Route to filtered handlers (low-frequency)
	for name, feature := range r.features {
		if msgFeature, ok := feature.(MessageFeature); ok {
			if err := msgFeature.HandleMessage(ctx, s, m); err != nil {
				if !errors.Is(err, ErrNotHandled) {
					r.logger.Error("feature error handling message",
						"feature", name,
						"error", err,
					)
				}
			}
		}
	}
}

// HandleMessageDelete routes message deletion events.
func (r *Registry) HandleMessageDelete(ctx context.Context, s *discordgo.Session, m *discordgo.MessageDelete) {
	r.eventRouter.RouteMessageDelete(ctx, s, m)
}

// HandleReactionAdd routes reaction add events to features.
func (r *Registry) HandleReactionAdd(ctx context.Context, s *discordgo.Session, ra *discordgo.MessageReactionAdd) {
	for name, feature := range r.features {
		if reactFeature, ok := feature.(ReactionFeature); ok {
			if err := reactFeature.HandleReactionAdd(ctx, s, ra); err != nil {
				if !errors.Is(err, ErrNotHandled) {
					r.logger.Error("feature error handling reaction add",
						"feature", name,
						"error", err,
					)
				}
			}
		}
	}
}

// HandleVoiceStateUpdate routes voice state updates using hybrid approach.
func (r *Registry) HandleVoiceStateUpdate(ctx context.Context, s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// 1. Route to indexed handlers (high-frequency)
	r.eventRouter.RouteVoiceStateUpdate(ctx, s, v)

	// 2. Route to filtered handlers (low-frequency)
	for name, feature := range r.features {
		if voiceFeature, ok := feature.(VoiceFeature); ok {
			if err := voiceFeature.HandleVoiceStateUpdate(ctx, s, v); err != nil {
				if !errors.Is(err, ErrNotHandled) {
					r.logger.Error("feature error handling voice state",
						"feature", name,
						"error", err,
					)
				}
			}
		}
	}
}

// RegisterSlashCommands registers all feature commands with Discord.
func (r *Registry) RegisterSlashCommands(s *discordgo.Session) error {
	var commands []*discordgo.ApplicationCommand

	for _, feature := range r.features {
		commands = append(commands, feature.RegisterCommands()...)
	}

	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("create command %s: %w", cmd.Name, err)
		}
		r.logger.Info("command registered", "name", cmd.Name)
	}

	return nil
}

// extractCommandName extracts the command name from an interaction.
func (r *Registry) extractCommandName(i *discordgo.InteractionCreate) string {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		return i.ApplicationCommandData().Name
	case discordgo.InteractionMessageComponent:
		return i.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		return i.ModalSubmitData().CustomID
	default:
		return ""
	}
}

// GetAllFeatures returns all registered features.
func (r *Registry) GetAllFeatures() []Feature {
	features := make([]Feature, 0, len(r.features))
	for _, feature := range r.features {
		features = append(features, feature)
	}
	return features
}

// ErrNotHandled is returned when a feature doesn't handle an event.
var ErrNotHandled = errors.New("not handled by this feature")
