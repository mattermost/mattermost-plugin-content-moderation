package main

import (
	"context"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const notificationWaitForResultTimeout = 10 * time.Second

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	if p.moderationProcessor != nil {
		p.moderationProcessor.queueMessage(p.API, post.Message)
	}
	return nil, ""
}

func (p *Plugin) MessageWillBeUpdated(c *plugin.Context, post, _ *model.Post) (*model.Post, string) {
	if p.moderationProcessor != nil {
		p.moderationProcessor.queueMessage(p.API, post.Message)
	}
	return post, ""
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if p.postProcessor != nil {
		p.postProcessor.queuePost(p.API, post)
	}
}

func (p *Plugin) MessageHasBeenUpdated(c *plugin.Context, post *model.Post) {
	if p.postProcessor != nil {
		p.postProcessor.queuePost(p.API, post)
	}
}

func (p *Plugin) NotificationsWillBeSent(post *model.Post) string {
	if len(post.Id) == 0 || p.postProcessor == nil ||
		!p.postProcessor.shouldModerateUser(post.UserId) ||
		!p.postProcessor.shouldModerateChannel(post.ChannelId) {
		return ""
	}
	result := p.postProcessor.resultsCache.waitForResult(post.Message, notificationWaitForResultTimeout)
	if result == nil {
		errMsg := "Failed to complete content moderation before notification deadline"
		p.API.LogError(errMsg, "post_id", post.Id, "err", context.DeadlineExceeded)
		return ""
	}
	if result.code == moderationResultFlagged {
		return "Can not send notifications for flagged post"
	}
	return ""
}
