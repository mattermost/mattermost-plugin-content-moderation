package main

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation"
	"github.com/stretchr/testify/assert"
)

func TestModerationProcessor_resultSeverityAboveThreshold(t *testing.T) {
	t.Run("returns false when no severities above threshold", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: 5,
		}

		result := moderation.Result{
			"hate":     2,
			"violence": 1,
			"sexual":   4,
			"selfharm": 0,
		}

		above := processor.resultSeverityAboveThreshold(result)
		assert.False(t, above)
	})

	t.Run("returns true when one severity above threshold", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: 5,
		}

		result := moderation.Result{
			"hate":     2,
			"violence": 6,
			"sexual":   4,
			"selfharm": 0,
		}

		above := processor.resultSeverityAboveThreshold(result)
		assert.True(t, above)
	})

	t.Run("returns true when multiple severities above threshold", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: 3,
		}

		result := moderation.Result{
			"hate":     7,
			"violence": 5,
			"sexual":   2,
			"selfharm": 4,
		}

		above := processor.resultSeverityAboveThreshold(result)
		assert.True(t, above)
	})

	t.Run("returns true when severity equals threshold", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: 5,
		}

		result := moderation.Result{
			"hate":     2,
			"violence": 5,
			"sexual":   1,
			"selfharm": 0,
		}

		above := processor.resultSeverityAboveThreshold(result)
		assert.True(t, above)
	})

	t.Run("returns false for empty result", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: 5,
		}

		result := moderation.Result{}

		above := processor.resultSeverityAboveThreshold(result)
		assert.False(t, above)
	})

	t.Run("returns true with threshold zero and all severities zero", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: 0,
		}

		result := moderation.Result{
			"hate":     0,
			"violence": 0,
			"sexual":   0,
			"selfharm": 0,
		}

		above := processor.resultSeverityAboveThreshold(result)
		assert.True(t, above)
	})

	t.Run("returns true with threshold zero and any positive severity", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: 0,
		}

		result := moderation.Result{
			"hate":     0,
			"violence": 1,
			"sexual":   0,
			"selfharm": 0,
		}

		above := processor.resultSeverityAboveThreshold(result)
		assert.True(t, above)
	})

	t.Run("returns false with negative threshold", func(t *testing.T) {
		processor := &ModerationProcessor{
			thresholdValue: -1,
		}

		result := moderation.Result{
			"hate":     0,
			"violence": 0,
			"sexual":   0,
			"selfharm": 0,
		}

		above := processor.resultSeverityAboveThreshold(result)
		assert.True(t, above)
	})
}
