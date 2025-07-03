package main

import (
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) registerSlashCommands() error {
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "moderation",
		DisplayName:      "Content Moderation",
		Description:      "Manage content moderation settings",
		AutoComplete:     true,
		AutoCompleteDesc: "Exclude or include channels from content moderation",
		AutoCompleteHint: "[exclude_channel|include_channel]",
	})
}

// ExecuteCommand executes a given slash command
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	parts := strings.Fields(args.Command)
	if len(parts) == 0 || parts[0] != "/moderation" {
		return &model.CommandResponse{}, nil
	}

	if len(parts) < 2 {
		return &model.CommandResponse{
			Text: "Usage: `/moderation exclude_channel` or `/moderation include_channel`",
		}, nil
	}

	switch parts[1] {
	case "exclude_channel":
		return p.executeExcludeCommand(args)
	case "include_channel":
		return p.executeIncludeCommand(args)
	default:
		return &model.CommandResponse{
			Text: "Usage: `/moderation exclude_channel` or `/moderation include_channel`",
		}, nil
	}
}

// executeExcludeCommand handles the exclude_channel subcommand
func (p *Plugin) executeExcludeCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !p.hasChannelPermission(args.UserId, args.ChannelId) {
		return &model.CommandResponse{
			Text: "You must be a channel admin or system admin to exclude channels from moderation.",
		}, nil
	}

	err := p.excludedChannelStore.SetExcluded(args.ChannelId, true)
	if err != nil {
		p.API.LogError("Failed to exclude channel", "channel_id", args.ChannelId, "user_id", args.UserId, "err", err)
		return &model.CommandResponse{
			Text: "Failed to exclude this channel from moderation.",
		}, nil
	}

	return &model.CommandResponse{
		Text: "This channel has been excluded from content moderation.",
	}, nil
}

// executeIncludeCommand handles the include_channel subcommand
func (p *Plugin) executeIncludeCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !p.hasChannelPermission(args.UserId, args.ChannelId) {
		return &model.CommandResponse{
			Text: "You must be a channel admin or system admin to include channels in moderation.",
		}, nil
	}

	err := p.excludedChannelStore.SetExcluded(args.ChannelId, false)
	if err != nil {
		p.API.LogError("Failed to include channel", "channel_id", args.ChannelId, "user_id", args.UserId, "err", err)
		return &model.CommandResponse{
			Text: "Failed to include this channel in moderation.",
		}, nil
	}

	return &model.CommandResponse{
		Text: "This channel has been included in content moderation.",
	}, nil
}

// hasChannelPermission checks if user has permission to manage channel moderation
func (p *Plugin) hasChannelPermission(userID, channelID string) bool {
	if p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
		return true
	}

	channel, err := p.API.GetChannel(channelID)
	if err != nil {
		return false
	}

	if channel.Type == model.ChannelTypeOpen {
		return p.API.HasPermissionToChannel(userID, channelID, model.PermissionManagePublicChannelProperties)
	}

	return p.API.HasPermissionToChannel(userID, channelID, model.PermissionManagePrivateChannelProperties)
}
