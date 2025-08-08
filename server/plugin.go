package main

import (
	"fmt"
	"sync"

	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation"
	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation/agents"
	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation/azure"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex
	configuration     *configuration

	postProcessor        *PostProcessor
	moderationProcessor  *ModerationProcessor
	excludedChannelStore ExcludedChannelsStore
}

func (p *Plugin) OnActivate() error {
	if !pluginapi.IsEnterpriseLicensedOrDevelopment(
		p.API.GetConfig(),
		p.API.GetLicense(),
	) {
		err := fmt.Errorf("this plugin requires an Enterprise license")
		p.API.LogError("Cannot initialize plugin", "err", err)
		return err
	}

	var err error
	p.excludedChannelStore, err = newExcludedChannelsStore(p.API)
	if err != nil {
		p.API.LogError("Failed to create excluded channel store", "err", err)
		return err
	}

	if err := p.registerSlashCommands(); err != nil {
		p.API.LogError("Failed to register slash commands", "err", err)
		return err
	}

	config := p.getConfiguration()
	if err := p.initialize(config); err != nil {
		p.API.LogError("Cannot initialize plugin", "err", err)
		return nil
	}

	return nil
}

func (p *Plugin) initialize(config *configuration) error {
	if p.postProcessor != nil {
		p.postProcessor.stop()
		p.postProcessor = nil
	}

	if p.moderationProcessor != nil {
		p.moderationProcessor.stop()
		p.moderationProcessor = nil
	}

	if !config.Enabled {
		p.API.LogInfo("Content moderation is disabled")
		return nil
	}

	excludedUsers := config.ExcludedUserSet()

	pluginBotID, err := p.API.EnsureBotUser(&model.Bot{
		Username:    config.BotUsername,
		DisplayName: config.BotDisplayName,
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize bot user")
	}

	// We need a user ID to interact with agent plugin based backends, but
	// we use the bot ID instead of user ID to ensure consistent access control.
	// The bot account only needs to be granted agent access once, rather than
	// requiring every user whose content is moderated to have agent permissions.
	moderator, err := initModerator(p.API, config, pluginBotID)
	if err != nil {
		return errors.Wrap(err, "failed to initialize moderator")
	}

	thresholdValue, err := config.ThresholdValue()
	if err != nil {
		return errors.Wrap(err, "failed to load moderation threshold")
	}

	moderationResultsCache := newModerationResultsCache()
	rateLimitPerMinute := config.RateLimitValue()
	moderationProcessor, err := newModerationProcessor(moderationResultsCache, moderator, thresholdValue, rateLimitPerMinute)
	if err != nil {
		return errors.Wrap(err, "failed to create post moderation processor")
	}
	p.moderationProcessor = moderationProcessor
	p.moderationProcessor.start(p.API)

	postCache := newPostCache()
	processor, err := newPostProcessor(
		pluginBotID, config.AuditLoggingEnabled, moderationResultsCache,
		postCache, excludedUsers, p.excludedChannelStore,
		config.ExcludeDirectMessages, config.ExcludePrivateChannels)
	if err != nil {
		return errors.Wrap(err, "failed to create post processor")
	}
	p.postProcessor = processor
	p.postProcessor.start(p.API)

	return nil
}

func initModerator(api plugin.API, config *configuration, pluginBotID string) (moderation.Moderator, error) {
	switch config.ModeratorConfig.Type {
	case "azure":
		azureConfig := &moderation.Config{
			Endpoint: config.ModeratorConfig.AzureEndpoint,
			APIKey:   config.ModeratorConfig.AzureAPIKey,
		}

		mod, err := azure.New(azureConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create Azure moderator")
		}

		api.LogInfo("Azure AI Content Safety moderator initialized")
		return mod, nil
	case "agents":
		mod, err := agents.New(api, config.ModeratorConfig.AgentsSystemPrompt, pluginBotID, config.ModeratorConfig.AgentsBotUsername)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create agents moderator")
		}

		api.LogInfo("Agents plugin moderator initialized")
		return mod, nil
	default:
		return nil, errors.Errorf("unknown moderator type: %s", config.ModeratorConfig.Type)
	}
}
