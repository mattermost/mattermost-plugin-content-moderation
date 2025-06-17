package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if p.processor != nil {
		p.processor.queuePostForProcessing(post)
	}
}

func (p *Plugin) MessageHasBeenUpdated(c *plugin.Context, post, _ *model.Post) {
	if p.processor != nil {
		p.processor.queuePostForProcessing(post)
	}
}
