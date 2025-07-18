package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

const defaultSystemPrompt = `You are an AI system trained to evaluate user-generated content for safety and policy compliance.

Your task is to assess the message below for the following four categories:

- Hate: Language that expresses, incites, or promotes hatred based on race, religion, gender, sexuality, or similar group identity.
- SelfHarm: Mentions of self-injury, suicide, or encouragement of self-harm.
- Sexual: Sexually explicit or suggestive language, including innuendo, erotic descriptions, or references to sexual acts.
- Violence: Threats, encouragement, or descriptions of physical violence, abuse, or harm.

For each category, assign a severity score using the following scale:
- 0 — No content of this type
- 2 — Mild concern
- 4 — Moderate concern
- 6 — Severe concern

Respond with a JSON object in **exactly** the following format (no extra commentary):

{
  "categoriesAnalysis": [
    {"category": "Hate", "severity": 0},
    {"category": "SelfHarm", "severity": 0},
    {"category": "Sexual", "severity": 0},
    {"category": "Violence", "severity": 0}
  ]
}`

var _ moderation.Moderator = (*Moderator)(nil)

type Moderator struct {
	client       *interpluginclient.Client
	systemPrompt string
	api          plugin.API
	botID        string
	botUsername  string
}

type CategoryAnalysis struct {
	Category string `json:"category"`
	Severity int    `json:"severity"`
}

type LLMResponse struct {
	CategoriesAnalysis []CategoryAnalysis `json:"categoriesAnalysis"`
}

func New(api plugin.API, systemPrompt string, botID string, botUsername string) (*Moderator, error) {
	client := interpluginclient.NewClient(&plugin.MattermostPlugin{API: api})

	if err := validateAgentsPlugin(client, api, botID); err != nil {
		return nil, errors.Wrap(err, "agents plugin not available")
	}

	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = defaultSystemPrompt
	}

	return &Moderator{
		client:       client,
		systemPrompt: systemPrompt,
		api:          api,
		botID:        botID,
		botUsername:  botUsername,
	}, nil
}

func validateAgentsPlugin(client *interpluginclient.Client, api plugin.API, botID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.SimpleCompletionWithContext(ctx, interpluginclient.SimpleCompletionRequest{
		SystemPrompt:    "Test connection",
		UserPrompt:      "Respond with 'OK'",
		RequesterUserID: botID,
	})

	if response != "OK" || err != nil {
		return errors.Wrap(err, "agents plugin connection test failed")
	}

	return nil
}

func (m *Moderator) ModerateText(ctx context.Context, text string) (moderation.Result, error) {
	req := interpluginclient.SimpleCompletionRequest{
		SystemPrompt:    m.systemPrompt,
		UserPrompt:      fmt.Sprintf("Message: %q", text),
		RequesterUserID: m.botID,
		BotUsername:     m.botUsername,
	}

	response, err := m.client.SimpleCompletionWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get LLM response from agents plugin")
	}

	result, err := m.parseStructuredResponse(response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse LLM response")
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
	switch severity {
	case 0:
	case 2:
	case 4:
	case 6:
	default:
		return errors.New("invalid severity value")
	}
	return nil
}
