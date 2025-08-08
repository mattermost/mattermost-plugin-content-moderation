package main

import (
	"context"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const emailNotificationWaitForResultTimeout = 15 * time.Second

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

func (p *Plugin) EmailNotificationWillBeSent(emailNotification *model.EmailNotification) (*model.EmailNotificationContent, string) {
	if p.postProcessor == nil {
		return nil, ""
	}

	post, err := p.postProcessor.postCache.getPost(p.API, emailNotification.PostId)
	if err != nil {
		p.API.LogError("Cannot retrieve post before sending email notification",
			"post_id", emailNotification.PostId, "err", err)
		return nil, ""
	}

	result := p.postProcessor.resultsCache.waitForResult(
		post.Message, emailNotificationWaitForResultTimeout)
	if result == nil {
		p.API.LogError(
			"Failed to complete content moderation before email notification timeout",
			"post_id", post.Id, "err", context.DeadlineExceeded)
		return nil, ""
	}

	if result.code == moderationResultFlagged {
		return nil, "content flagged by moderation plugin"
	}

	return nil, ""
}
