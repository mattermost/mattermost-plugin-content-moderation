package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

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
