package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostProcessor_shouldModerateUser(t *testing.T) {
	t.Run("should not moderate bot user", func(t *testing.T) {
		processor := &PostProcessor{
			botID:         "bot123",
			excludedUsers: map[string]struct{}{},
		}

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateUser("bot123", auditRecord)

		assert.False(t, result)
	})

	t.Run("should moderate user when no exclusions", func(t *testing.T) {
		processor := &PostProcessor{
			botID:         "bot123",
			excludedUsers: map[string]struct{}{},
		}

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateUser("user456", auditRecord)

		assert.True(t, result)
	})

	t.Run("should not moderate excluded user", func(t *testing.T) {
		processor := &PostProcessor{
			botID: "bot123",
			excludedUsers: map[string]struct{}{
				"user456": {},
				"user789": {},
			},
		}

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateUser("user456", auditRecord)

		assert.False(t, result)
	})

	t.Run("should moderate non-excluded user when exclusions exist", func(t *testing.T) {
		processor := &PostProcessor{
			botID: "bot123",
			excludedUsers: map[string]struct{}{
				"user456": {},
			},
		}

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateUser("user999", auditRecord)

		assert.True(t, result)
	})
}

func TestPostProcessor_shouldModerateChannel(t *testing.T) {
	t.Run("should not moderate excluded channel", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels: map[string]struct{}{
				"channel123": {},
				"channel456": {},
			},
		}

		api := &plugintest.API{}
		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "channel123", auditRecord)

		assert.False(t, result)
	})

	t.Run("should not moderate direct messages when excluded", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels:      map[string]struct{}{},
			excludeDirectMessages: true,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "dm_channel").Return(&model.Channel{
			Id:   "dm_channel",
			Type: model.ChannelTypeDirect,
		}, nil)

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "dm_channel", auditRecord)

		assert.False(t, result)
		api.AssertExpectations(t)
	})

	t.Run("should not moderate group messages when direct messages excluded", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels:      map[string]struct{}{},
			excludeDirectMessages: true,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "group_channel").Return(&model.Channel{
			Id:   "group_channel",
			Type: model.ChannelTypeGroup,
		}, nil)

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "group_channel", auditRecord)

		assert.False(t, result)
		api.AssertExpectations(t)
	})

	t.Run("should not moderate private channels when excluded", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels:       map[string]struct{}{},
			excludePrivateChannels: true,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "private_channel").Return(&model.Channel{
			Id:   "private_channel",
			Type: model.ChannelTypePrivate,
		}, nil)

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "private_channel", auditRecord)

		assert.False(t, result)
		api.AssertExpectations(t)
	})

	t.Run("should moderate open channels", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels:       map[string]struct{}{},
			excludeDirectMessages:  false,
			excludePrivateChannels: false,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "open_channel").Return(&model.Channel{
			Id:   "open_channel",
			Type: model.ChannelTypeOpen,
		}, nil)

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "open_channel", auditRecord)

		assert.True(t, result)
		api.AssertExpectations(t)
	})

	t.Run("should moderate direct messages when not excluded", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels:      map[string]struct{}{},
			excludeDirectMessages: false,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "dm_channel").Return(&model.Channel{
			Id:   "dm_channel",
			Type: model.ChannelTypeDirect,
		}, nil)

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "dm_channel", auditRecord)

		assert.True(t, result)
		api.AssertExpectations(t)
	})

	t.Run("should moderate private channels when not excluded", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels:       map[string]struct{}{},
			excludePrivateChannels: false,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "private_channel").Return(&model.Channel{
			Id:   "private_channel",
			Type: model.ChannelTypePrivate,
		}, nil)

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "private_channel", auditRecord)

		assert.True(t, result)
		api.AssertExpectations(t)
	})

	t.Run("should handle channel fetch error gracefully", func(t *testing.T) {
		processor := &PostProcessor{
			excludedChannels: map[string]struct{}{},
		}

		api := &plugintest.API{}
		api.On("GetChannel", "error_channel").Return(nil, &model.AppError{Message: "channel not found"})
		api.On("LogError", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		auditRecord := plugin.MakeAuditRecord("test", model.AuditStatusAttempt)
		result := processor.shouldModerateChannel(api, "error_channel", auditRecord)

		// Should default to moderating (treats as open channel)
		assert.True(t, result)
		api.AssertExpectations(t)
	})
}
