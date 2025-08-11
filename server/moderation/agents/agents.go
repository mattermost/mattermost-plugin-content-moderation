package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/interpluginclient"
	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
)

const (
	CategoryHate     = "Hate"
	CategorySexual   = "Sexual"
	CategoryViolence = "Violence"
	CategorySelfHarm = "SelfHarm"
)

var _ moderation.Moderator = (*Moderator)(nil)

type Moderator struct {
	client           *interpluginclient.Client
	systemPrompt     string
	api              plugin.API
	pluginBotID      string
	agentBotUsername string
}

type CategoryAnalysis struct {
	Category string `json:"category"`
	Severity int    `json:"severity"`
}

type LLMResponse struct {
	CategoriesAnalysis []CategoryAnalysis `json:"categoriesAnalysis"`
}

func New(api plugin.API, systemPrompt string, pluginBotID string, agentBotUsername string) (*Moderator, error) {
	client := interpluginclient.NewClient(&plugin.MattermostPlugin{API: api})
	return &Moderator{
		client:           client,
		systemPrompt:     systemPrompt,
		api:              api,
		pluginBotID:      pluginBotID,
		agentBotUsername: agentBotUsername,
	}, nil
}

func (m *Moderator) ModerateText(ctx context.Context, text string) (moderation.Result, error) {
	req := interpluginclient.SimpleCompletionRequest{
		SystemPrompt:    m.systemPrompt,
		UserPrompt:      fmt.Sprintf("Message: %q", text),
		RequesterUserID: m.pluginBotID,
		BotUsername:     m.agentBotUsername,
	}

	response, err := m.client.SimpleCompletionWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get LLM response from agents plugin")
	}

	result, err := m.parseStructuredResponse(response)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to parse LLM response: '%s'", response))
	}

	return result, nil
}

func (m *Moderator) parseStructuredResponse(response string) (moderation.Result, error) {
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, errors.New("no JSON block found in response")
	}
	jsonStr := response[jsonStart : jsonEnd+1]

	// sometimes the LLM returns comments in the json, so we strip those out
	commentRe := regexp.MustCompile(`//.*\n`)
	jsonStr = commentRe.ReplaceAllString(jsonStr, "")

	var llmResponse LLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &llmResponse); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal JSON response")
	}

	if len(llmResponse.CategoriesAnalysis) == 0 {
		return nil, errors.New("received empty analysis in JSON response")
	}

	result := make(moderation.Result)
	for _, categoryResult := range llmResponse.CategoriesAnalysis {
		if err := m.validateSeverity(categoryResult.Severity); err != nil {
			return nil, fmt.Errorf("%w: category=%s, severity=%d",
				err, categoryResult.Category, categoryResult.Severity)
		}
		result[categoryResult.Category] = categoryResult.Severity
	}

	return result, nil
}

func (m *Moderator) validateSeverity(severity int) error {
	if severity < 0 || severity > 6 {
		return errors.New("invalid severity value")
	}
	return nil
}
