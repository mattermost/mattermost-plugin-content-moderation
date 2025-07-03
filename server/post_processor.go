package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
)

const channelCacheTTL = 1 * time.Minute

const (
	channelNotificationTemplate = "_A post with potentially offensive content was flagged and removed._"
	dmNotificationTemplate      = "_Your post with the following content was flagged and removed:_\n\n%s"
)

const (
	maxPostProcessingQueueSize = maxModerationProcessingQueueSize
	waitForResultTimeout       = 1 * time.Minute
)

const (
	auditEventTypeContentModeration = "contentModeration"
	auditMetaKeyFlagged             = "flagged"
	auditMetaKeyResult              = "result"
	auditMetaKeyThreshold           = "threshold"
	auditMetaKeyExcluded            = "exclusion_reason"
	auditMetaKeyPost                = "post"
)

var (
	ErrModerationRejection   = errors.New("potentially inappropriate content detected")
	ErrModerationUnavailable = errors.New("moderation service is not available")
)

type PostProcessor struct {
	botID           string
	auditLogEnabled bool

	excludedUsers          map[string]struct{}
	excludedChannels       map[string]struct{}
	channelInfoCache       sync.Map // channel ID -> channelInfoCacheEntry
	excludeDirectMessages  bool
	excludePrivateChannels bool

	resultsCache *moderationResultsCache
	postsCh      chan *model.Post
	done         chan struct{}
}

type channelInfoCacheEntry struct {
	channelType  model.ChannelType
	creationTime time.Time
}

func newPostProcessor(
	botID string,
	auditLogEnabled bool,
	moderationResultsCache *moderationResultsCache,
	excludedUsers map[string]struct{},
	excludedChannels map[string]struct{},
	excludeDirectMessages bool,
	excludePrivateChannels bool,
) (*PostProcessor, error) {
	return &PostProcessor{
		botID:                  botID,
		resultsCache:           moderationResultsCache,
		auditLogEnabled:        auditLogEnabled,
		excludedUsers:          excludedUsers,
		excludedChannels:       excludedChannels,
		excludeDirectMessages:  excludeDirectMessages,
		excludePrivateChannels: excludePrivateChannels,
		postsCh:                make(chan *model.Post, maxPostProcessingQueueSize),
		done:                   make(chan struct{}),
	}, nil
}

func (p *PostProcessor) start(api plugin.API) {
	go p.processPostsLoop(api)
}

func (p *PostProcessor) processPostsLoop(api plugin.API) {
	for {
		var post *model.Post

		select {
		case post = <-p.postsCh:
		case <-p.done:
			return
		}

		record := plugin.MakeAuditRecord(auditEventTypeContentModeration, model.AuditStatusAttempt)
		model.AddEventParameterAuditableToAuditRec(record, auditMetaKeyPost, post)

		if !p.shouldModerateUser(post.UserId, record) ||
			!p.shouldModerateChannel(api, post.ChannelId, record) {
			continue
		}

		result := p.resultsCache.waitForResult(post.Message, waitForResultTimeout)
		if result == nil {
			errMsg := "Failed to complete content moderation"
			api.LogError(errMsg, "post_id", post.Id, "err", context.DeadlineExceeded)
			p.logAuditFail(api, record, errMsg, context.DeadlineExceeded)
			continue
		}

		record.AddMeta(auditMetaKeyResult, result.result)

		switch result.code {
		case moderationResultProcessed:
			record.AddMeta(auditMetaKeyFlagged, false)
			p.logAuditSuccess(api, record)
			continue
		case moderationResultPending:
			errMsg := "Failed to complete content moderation"
			err := errors.New("moderation result from cache is still pending")
			api.LogError(errMsg, "post_id", post.Id, "err", err)
			p.logAuditFail(api, record, errMsg, err)
			continue
		case moderationResultFlagged:
			record.AddMeta(auditMetaKeyFlagged, true)
			if err := api.DeletePost(post.Id); err != nil {
				errMsg := "Failed to delete post flagged by content moderation"
				api.LogError(errMsg, "post_id", post.Id, "err", err)
				p.logAuditFail(api, record, errMsg, err)
				continue
			}
			if err := p.reportModerationEvent(api, post); err != nil {
				errMsg := "Failed report content moderation event"
				api.LogError(errMsg, "post_id", post.Id, "err", err)
				p.logAuditFail(api, record, errMsg, err)
				continue
			}
			p.logAuditSuccess(api, record)
			continue
		case moderationResultError:
			errMsg := "Content moderation error"
			api.LogError(errMsg, "err", result.err, "post_id", post.Id, "user_id", post.UserId)
			p.logAuditFail(api, record, errMsg, result.err)
			continue
		}
	}
}

func (p *PostProcessor) stop() {
	close(p.done)
}

func (p *PostProcessor) queuePost(api plugin.API, post *model.Post) {
	if post.Message == "" {
		return
	}

	select {
	case p.postsCh <- post:
		return
	default:
		api.LogError("Content moderation unable to analyze post: exceeded maximum post queue size")
		return
	}
}

func (p *PostProcessor) shouldModerateUser(userID string, auditRecord *model.AuditRecord) bool {
	if userID == p.botID {
		auditRecord.AddMeta(auditMetaKeyExcluded, "excluded_plugin_bot")
		return false
	}

	if len(p.excludedUsers) == 0 {
		return true
	}

	if _, excluded := p.excludedUsers[userID]; excluded {
		auditRecord.AddMeta(auditMetaKeyExcluded, "excluded_user_list")
		return false
	}

	return true
}

func (p *PostProcessor) shouldModerateChannel(api plugin.API, channelID string, auditRecord *model.AuditRecord) bool {
	if len(p.excludedChannels) > 0 {
		if _, excluded := p.excludedChannels[channelID]; excluded {
			auditRecord.AddMeta(auditMetaKeyExcluded, "excluded_channel_list")
			return false
		}
	}

	channelType := p.getChannelType(api, channelID)

	if p.excludeDirectMessages &&
		(channelType == model.ChannelTypeDirect ||
			channelType == model.ChannelTypeGroup) {
		auditRecord.AddMeta(auditMetaKeyExcluded, "excluded_private_messages")
		return false
	}

	if p.excludePrivateChannels && channelType == model.ChannelTypePrivate {
		auditRecord.AddMeta(auditMetaKeyExcluded, "excluded_private_channels")
		return false
	}

	return true
}

func (p *PostProcessor) getChannelType(api plugin.API, channelID string) model.ChannelType {
	var entry channelInfoCacheEntry
	entryObj, ok := p.channelInfoCache.Load(channelID)
	if ok {
		entry = entryObj.(channelInfoCacheEntry)
	}
	if !ok || time.Since(entry.creationTime) > channelCacheTTL {
		channel, err := api.GetChannel(channelID)
		if err != nil {
			api.LogError("Failed to get channel type for moderation check",
				"channel_id", channelID, "err", err)
			// Default to open channel if we can't determine the type
			return model.ChannelTypeOpen
		}
		entry = channelInfoCacheEntry{
			channelType:  channel.Type,
			creationTime: time.Now(),
		}
		p.channelInfoCache.Store(channelID, entry)
	}
	return entry.channelType
}

func (p *PostProcessor) reportModerationEvent(api plugin.API, post *model.Post) error {
	if _, err := api.CreatePost(&model.Post{
		UserId:    p.botID,
		ChannelId: post.ChannelId,
		RootId:    post.RootId,
		Message:   channelNotificationTemplate,
	}); err != nil {
		return errors.Wrap(err, "failed to post channel notification")
	}

	dmChannel, err := api.GetDirectChannel(p.botID, post.UserId)
	if err != nil {
		return errors.Wrap(err, "failed to create DM channel")
	}

	if _, err := api.CreatePost(&model.Post{
		UserId:    p.botID,
		ChannelId: dmChannel.Id,
		Message:   fmt.Sprintf(dmNotificationTemplate, post.Message),
	}); err != nil {
		return errors.Wrap(err, "failed to send DM notification")
	}

	return nil
}

func (p *PostProcessor) logAuditSuccess(api plugin.API, auditRecord *model.AuditRecord) {
	if !p.auditLogEnabled {
		return
	}
	auditRecord.Success()
	api.LogAuditRec(auditRecord)
}

func (p *PostProcessor) logAuditFail(api plugin.API, auditRecord *model.AuditRecord, errDesc string, err error) {
	if !p.auditLogEnabled {
		return
	}
	auditRecord.Fail()
	auditRecord.AddErrorDesc(errors.Wrap(err, errDesc).Error())
	api.LogAuditRec(auditRecord)
}
