package main

import (
	"fmt"
	"sync"

	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation"
	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation/azure"
	"github.com/mattermost/mattermost-plugin-content-moderation/server/store/sqlstore"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex
	configuration     *configuration

	sqlStore            *sqlstore.SQLStore
	postProcessor       *PostProcessor
	moderationProcessor *ModerationProcessor
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

	client := pluginapi.NewClient(p.API, p.Driver)
	SQLStore, err := sqlstore.New(client.Store, &client.Log)
	if err != nil {
		p.API.LogError("Cannot create SQLStore", "err", err)
		return err
	}
	p.sqlStore = SQLStore

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

	moderator, err := initModerator(p.API, config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize moderator")
	}

	thresholdValue, err := config.ThresholdValue()
	if err != nil {
		return errors.Wrap(err, "failed to load moderation threshold")
	}

	moderationResultsCache := newModerationResultsCache()
	moderationProcessor, err := newModerationProcessor(moderationResultsCache, moderator, thresholdValue)
	if err != nil {
		return errors.Wrap(err, "failed to create post moderation processor")
	}
	p.moderationProcessor = moderationProcessor
	p.moderationProcessor.start(p.API)

	excludedUsers := config.ExcludedUserSet()
	excludedChannels := config.ExcludedChannelSet()

	botID, err := p.API.EnsureBotUser(&model.Bot{Username: config.BotUsername})
	if err != nil {
		return errors.Wrap(err, "could not initialize bot user")
	}

	processor, err := newPostProcessor(
		botID, config.AuditLoggingEnabled, moderationResultsCache,
		excludedUsers, excludedChannels,
		config.ExcludeDirectMessages, config.ExcludePrivateChannels)
	if err != nil {
		return errors.Wrap(err, "failed to create post processor")
	}
	p.postProcessor = processor
	p.postProcessor.start(p.API)

	return nil
}

func initModerator(api plugin.API, config *configuration) (moderation.Moderator, error) {
	switch config.Type {
	case "azure":
		azureConfig := &moderation.Config{
			Endpoint: config.Endpoint,
			APIKey:   config.APIKey,
		}

		mod, err := azure.New(azureConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create Azure moderator")
		}

		api.LogInfo("Azure AI Content Safety moderator initialized")
		return mod, nil
	default:
		return nil, errors.Errorf("unknown moderator type: %s", config.Type)
	}
}
