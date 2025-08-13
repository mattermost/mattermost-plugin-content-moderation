package main

import (
	"context"
	"time"

	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// The current Azure rate limit is 1000 posts per minute.
// Using half of that to give us some wiggle room:
// https://learn.microsoft.com/en-us/azure/ai-services/content-safety/faq
const (
	moderationAPITimeout             = 15 * time.Second
	maxModerationProcessingQueueSize = 10000
)

type ModerationProcessor struct {
	moderator              moderation.Moderator
	thresholdValue         int
	moderationResultsCache *moderationResultsCache
	messagesCh             chan string
	done                   chan struct{}
	cleanupTicker          *time.Ticker
	rateLimitPerMinute     int
	processingInterval     time.Duration
}

func newModerationProcessor(
	moderationResultsCache *moderationResultsCache,
	moderator moderation.Moderator,
	thresholdValue int,
	rateLimitPerMinute int,
) (*ModerationProcessor, error) {
	if moderator == nil {
		return nil, ErrModerationUnavailable
	}
	processingInterval := time.Duration(float64(time.Minute) / float64(rateLimitPerMinute))
	return &ModerationProcessor{
		moderator:              moderator,
		thresholdValue:         thresholdValue,
		moderationResultsCache: moderationResultsCache,
		messagesCh:             make(chan string, maxModerationProcessingQueueSize),
		done:                   make(chan struct{}),
		cleanupTicker:          time.NewTicker(5 * time.Minute),
		rateLimitPerMinute:     rateLimitPerMinute,
		processingInterval:     processingInterval,
	}, nil
}

func (p *ModerationProcessor) start(api plugin.API) {
	go func() {
		for {
			select {
			case message := <-p.messagesCh:
				p.moderateMessage(message)
				time.Sleep(p.processingInterval)
			case <-p.cleanupTicker.C:
				p.moderationResultsCache.cleanup()
				continue
			case <-p.done:
				return
			}
		}
	}()
}

func (p *ModerationProcessor) stop() {
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		p.cleanupTicker = nil
	}
	close(p.done)
}

func (p *ModerationProcessor) queueMessage(api plugin.API, message string) {
	if message == "" {
		return
	}

	shouldQueue := p.moderationResultsCache.setResultPending(message)
	if !shouldQueue {
		return
	}

	select {
	case p.messagesCh <- message:
		return
	default:
		api.LogError("Content moderation unable to analyze post: exceeded maximum post queue size")
		return
	}
}

func (p *ModerationProcessor) moderateMessage(message string) {
	ctx, cancel := context.WithTimeout(context.Background(), moderationAPITimeout)
	defer cancel()

	result, err := p.moderator.ModerateText(ctx, message)
	if err != nil {
		p.moderationResultsCache.setModerationResultError(message, err)
		return
	}

	if p.resultSeverityAboveThreshold(result) {
		p.moderationResultsCache.setModerationResultFlagged(message, result)
		return
	}

	p.moderationResultsCache.setModerationResultNotFlagged(message, result)
}

func (p *ModerationProcessor) resultSeverityAboveThreshold(result moderation.Result) bool {
	for _, severity := range result {
		if severity >= p.thresholdValue {
			return true
		}
	}
	return false
}
