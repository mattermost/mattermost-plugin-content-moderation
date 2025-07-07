package main

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/pkg/errors"
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

func TestPostProcessor_processPostsLoop(t *testing.T) {
	t.Run("stops when done channel is closed", func(t *testing.T) {
		processor := &PostProcessor{
			botID:         "bot123",
			excludedUsers: map[string]struct{}{},
			postsCh:       make(chan *model.Post, 1),
			done:          make(chan struct{}),
		}

		api := &plugintest.API{}

		// Call stop to close the done channel
		processor.stop()

		// This should return immediately without blocking
		start := time.Now()
		processor.processPostsLoop(api)
		duration := time.Since(start)

		// Should return very quickly since done channel is already closed
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("skips moderation when user should not be moderated", func(t *testing.T) {
		cache := newModerationResultsCache()
		processor := &PostProcessor{
			botID:            "bot123",
			excludedUsers:    map[string]struct{}{"user456": {}},
			excludedChannels: map[string]struct{}{},
			resultsCache:     cache,
			postsCh:          make(chan *model.Post, 1),
			done:             make(chan struct{}),
		}

		api := &plugintest.API{}
		post := &model.Post{
			Id:        "post123",
			UserId:    "user456", // excluded user
			ChannelId: "channel123",
			Message:   "test message",
		}

		// Start processing loop in goroutine
		done := make(chan struct{})
		go func() {
			defer close(done)
			processor.processPostsLoop(api)
		}()

		// Queue post and give it time to process
		processor.queuePost(api, post)
		time.Sleep(50 * time.Millisecond)

		// Now stop the processor
		processor.stop()

		// Wait for loop to finish
		<-done

		// Verify no API calls were made (user was excluded)
		api.AssertExpectations(t)
	})

	t.Run("skips moderation when channel should not be moderated", func(t *testing.T) {
		cache := newModerationResultsCache()
		processor := &PostProcessor{
			botID:            "bot123",
			excludedUsers:    map[string]struct{}{},
			excludedChannels: map[string]struct{}{"channel123": {}},
			resultsCache:     cache,
			postsCh:          make(chan *model.Post, 1),
			done:             make(chan struct{}),
		}

		api := &plugintest.API{}

		post := &model.Post{
			Id:        "post123",
			UserId:    "user456",
			ChannelId: "channel123", // excluded channel
			Message:   "test message",
		}

		// Start processing loop in goroutine
		done := make(chan struct{})
		go func() {
			defer close(done)
			processor.processPostsLoop(api)
		}()

		// Queue post and give it time to process
		processor.queuePost(api, post)
		time.Sleep(50 * time.Millisecond)

		// Now stop the processor
		processor.stop()

		// Wait for loop to finish
		<-done

		// Verify no API calls were made (channel was excluded)
		api.AssertExpectations(t)
	})

	t.Run("handles processed moderation result", func(t *testing.T) {
		cache := newModerationResultsCache()
		processor := &PostProcessor{
			botID:            "bot123",
			excludedUsers:    map[string]struct{}{},
			excludedChannels: map[string]struct{}{},
			resultsCache:     cache,
			postsCh:          make(chan *model.Post, 1),
			done:             make(chan struct{}),
			auditLogEnabled:  false,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "channel123").Return(&model.Channel{
			Id:   "channel123",
			Type: model.ChannelTypeOpen,
		}, nil)

		post := &model.Post{
			Id:        "post123",
			UserId:    "user456",
			ChannelId: "channel123",
			Message:   "test message",
		}

		// Set processed result
		cache.setModerationResultNotFlagged("test message", moderation.Result{})

		// Start processing loop in goroutine
		done := make(chan struct{})
		go func() {
			defer close(done)
			processor.processPostsLoop(api)
		}()

		// Queue post and give it time to process
		processor.queuePost(api, post)
		time.Sleep(50 * time.Millisecond)

		// Now stop the processor
		processor.stop()

		// Wait for loop to finish
		<-done

		api.AssertExpectations(t)
	})

	t.Run("handles flagged content and deletes post", func(t *testing.T) {
		cache := newModerationResultsCache()
		processor := &PostProcessor{
			botID:            "bot123",
			excludedUsers:    map[string]struct{}{},
			excludedChannels: map[string]struct{}{},
			resultsCache:     cache,
			postsCh:          make(chan *model.Post, 1),
			done:             make(chan struct{}),
			auditLogEnabled:  false,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "channel123").Return(&model.Channel{
			Id:   "channel123",
			Type: model.ChannelTypeOpen,
		}, nil)
		api.On("DeletePost", "post123").Return(nil)
		api.On("GetDirectChannel", "bot123", "user456").Return(&model.Channel{
			Id: "dm_channel",
		}, nil)
		api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.ChannelId == "channel123" && post.UserId == "bot123"
		})).Return(&model.Post{}, nil)
		api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.ChannelId == "dm_channel" && post.UserId == "bot123"
		})).Return(&model.Post{}, nil)

		post := &model.Post{
			Id:        "post123",
			UserId:    "user456",
			ChannelId: "channel123",
			Message:   "test message",
		}

		// Set flagged result
		cache.setModerationResultFlagged("test message", map[string]int{"hate": 7})

		// Start processing loop in goroutine
		done := make(chan struct{})
		go func() {
			defer close(done)
			processor.processPostsLoop(api)
		}()

		// Queue post and give it time to process
		processor.queuePost(api, post)
		time.Sleep(50 * time.Millisecond)

		// Now stop the processor
		processor.stop()

		// Wait for loop to finish
		<-done

		api.AssertExpectations(t)
	})

	t.Run("handles moderation error", func(t *testing.T) {
		cache := newModerationResultsCache()
		processor := &PostProcessor{
			botID:            "bot123",
			excludedUsers:    map[string]struct{}{},
			excludedChannels: map[string]struct{}{},
			resultsCache:     cache,
			postsCh:          make(chan *model.Post, 1),
			done:             make(chan struct{}),
			auditLogEnabled:  false,
		}

		api := &plugintest.API{}
		api.On("GetChannel", "channel123").Return(&model.Channel{
			Id:   "channel123",
			Type: model.ChannelTypeOpen,
		}, nil)
		api.On("LogError", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		post := &model.Post{
			Id:        "post123",
			UserId:    "user456",
			ChannelId: "channel123",
			Message:   "test message",
		}

		// Set error result
		cache.setModerationResultError("test message", errors.New("moderation API error"))

		// Start processing loop in goroutine
		done := make(chan struct{})
		go func() {
			defer close(done)
			processor.processPostsLoop(api)
		}()

		// Queue post and give it time to process
		processor.queuePost(api, post)
		time.Sleep(50 * time.Millisecond)

		// Now stop the processor
		processor.stop()

		// Wait for loop to finish
		<-done

		api.AssertExpectations(t)
	})
}
