package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) registerSlashCommands() error {
	moderationAutoComplete := model.NewAutocompleteData("moderation", "", "Manage content moderation settings")
	channelAutoComplete := model.NewAutocompleteData("channel", "", "Manage content moderation settings for channel")
	channelAutoComplete.AddStaticListArgument("action", true, []model.AutocompleteListItem{
		{Item: "disable", HelpText: "Disable moderation for channel"},
		{Item: "enable", HelpText: "Enable moderation for channel"},
		{Item: "status", HelpText: "Print moderation status for this channel"},
		{Item: "list", HelpText: "List all excluded channels you have permission to manage"},
	})
	moderationAutoComplete.AddCommand(channelAutoComplete)

	command := model.Command{
		Trigger:          "moderation",
		DisplayName:      "Content Moderation",
		Description:      "Manage content moderation settings",
		AutoComplete:     true,
		AutocompleteData: moderationAutoComplete,
	}
	return p.API.RegisterCommand(&command)
}

// ExecuteCommand executes a given slash command
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	parts := strings.Fields(args.Command)
	if len(parts) == 0 || parts[0] != "/moderation" {
		return &model.CommandResponse{}, nil
	}

	if len(parts) < 3 || parts[1] != "channel" {
		return &model.CommandResponse{
			Text: "Error: invalid moderation command",
		}, nil
	}

	switch parts[2] {
	case "disable":
		return p.executeDisableCommand(args)
	case "enable":
		return p.executeEnableCommand(args)
	case "status":
		return p.executeStatusCommand(args)
	case "list":
		return p.executeListCommand(args)
	default:
		return &model.CommandResponse{
			Text: "Error: invalid moderation command",
		}, nil
	}
}

// executeDisableCommand handles the disable_channel subcommand
func (p *Plugin) executeDisableCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	auditRecord := plugin.MakeAuditRecord(auditEventTypeManageChannelModeration, model.AuditStatusAttempt)
	auditRecord.AddMeta(auditMetaKeyChannelID, args.ChannelId)
	auditRecord.AddMeta(auditMetaKeyUserID, args.UserId)
	auditRecord.AddMeta(auditMetaKeyAction, "disable")

	if p.getConfiguration().AuditLoggingEnabled {
		defer p.API.LogAuditRec(auditRecord)
	}

	if !p.hasChannelPermission(args.UserId, args.ChannelId) {
		return &model.CommandResponse{
			Text: "You must be a channel admin or system admin to disable moderation for channels.",
		}, nil
	}

	err := p.excludedChannelStore.SetExcluded(args.ChannelId, true)
	if err != nil {
		p.API.LogError("Failed to disable channel", "channel_id", args.ChannelId, "user_id", args.UserId, "err", err)
		auditRecord.AddErrorDesc(err.Error())
		auditRecord.Fail()

		return &model.CommandResponse{
			Text: "Failed to disable moderation for this channel.",
		}, nil
	}

	p.API.LogInfo("Channel moderation disabled", "channel_id", args.ChannelId, "user_id", args.UserId)
	auditRecord.Success()

	return &model.CommandResponse{
		Text: "Content moderation has been disabled for this channel.",
	}, nil
}

func (p *Plugin) executeEnableCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	auditRecord := plugin.MakeAuditRecord(auditEventTypeManageChannelModeration, model.AuditStatusAttempt)
	auditRecord.AddMeta(auditMetaKeyChannelID, args.ChannelId)
	auditRecord.AddMeta(auditMetaKeyUserID, args.UserId)
	auditRecord.AddMeta(auditMetaKeyAction, "enable")

	if p.getConfiguration().AuditLoggingEnabled {
		defer p.API.LogAuditRec(auditRecord)
	}

	if !p.hasChannelPermission(args.UserId, args.ChannelId) {
		return &model.CommandResponse{
			Text: "You must be a channel admin or system admin to enable moderation for channels.",
		}, nil
	}

	err := p.excludedChannelStore.SetExcluded(args.ChannelId, false)
	if err != nil {
		p.API.LogError("Failed to enable channel", "channel_id", args.ChannelId, "user_id", args.UserId, "err", err)
		auditRecord.AddErrorDesc(err.Error())
		auditRecord.Fail()

		return &model.CommandResponse{
			Text: "Failed to enable moderation for this channel.",
		}, nil
	}

	p.API.LogInfo("Channel moderation enabled", "channel_id", args.ChannelId, "user_id", args.UserId)
	auditRecord.Success()

	return &model.CommandResponse{
		Text: "Content moderation has been enabled for this channel.",
	}, nil
}

func (p *Plugin) executeStatusCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !p.hasChannelPermission(args.UserId, args.ChannelId) {
		return &model.CommandResponse{
			Text: "You must be a channel admin or system admin to see moderation status.",
		}, nil
	}

	excluded, err := p.excludedChannelStore.IsExcluded(args.ChannelId)
	if err != nil {
		p.API.LogError("Failed to get channel status",
			"channel_id", args.ChannelId, "user_id", args.UserId, "err", err)
		return &model.CommandResponse{
			Text: "Failed to get moderation status of channel.",
		}, nil
	}

	var statusMessage string
	if excluded {
		statusMessage = "This channel is not actively moderated."
	} else {
		statusMessage = "This channel is actively moderated."
	}

	return &model.CommandResponse{Text: statusMessage}, nil
}

func (p *Plugin) executeListCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	excludedChannels := p.excludedChannelStore.ListExcluded()

	var filteredChannels []ExcludedChannelInfo
	for _, channelInfo := range excludedChannels {
		if !p.hasChannelPermission(args.UserId, channelInfo.ID) {
			continue
		}
		filteredChannels = append(filteredChannels, channelInfo)
	}

	if len(filteredChannels) == 0 {
		return &model.CommandResponse{
			Text: "There are no excluded channels that you have permission to manage.",
		}, nil
	}

	var channelNames []string
	for _, channel := range filteredChannels {
		channelNames = append(channelNames, "~"+channel.Name)
	}

	response := fmt.Sprintf("The following channels are excluded from moderation:\n%s",
		strings.Join(channelNames, ", "))

	return &model.CommandResponse{Text: response}, nil
}

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
