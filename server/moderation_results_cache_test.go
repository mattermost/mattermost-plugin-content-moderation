package main

import (
	"testing"
	"time"
)

func TestModerationResultsCache_cleanup(t *testing.T) {
	t.Run("cleanup expired entries", func(t *testing.T) {
		cache := newModerationResultsCache()
		cache.cacheTTL = 100 * time.Millisecond

		// Add some entries
		cache.setResultPending("message1")
		cache.setResultPending("message2")
		cache.setModerationResultNotFlagged("message3")

		// Verify entries exist
		if len(cache.cache) != 3 {
			t.Errorf("Expected 3 cache entries, got %d", len(cache.cache))
		}

		// Wait for entries to expire
		time.Sleep(150 * time.Millisecond)

		// Cleanup expired entries
		cache.cleanup(false)

		// Verify entries are removed
		if len(cache.cache) != 0 {
			t.Errorf("Expected 0 cache entries after cleanup, got %d", len(cache.cache))
		}
	})

	t.Run("cleanup with ignoreExpiry true", func(t *testing.T) {
		cache := newModerationResultsCache()
		cache.cacheTTL = 5 * time.Minute // Long TTL

		// Add some entries
		cache.setResultPending("message1")
		cache.setModerationResultNotFlagged("message2")

		// Verify entries exist
		if len(cache.cache) != 2 {
			t.Errorf("Expected 2 cache entries, got %d", len(cache.cache))
		}

		// Cleanup ignoring expiry
		cache.cleanup(true)

		// Verify all entries are removed
		if len(cache.cache) != 0 {
			t.Errorf("Expected 0 cache entries after cleanup with ignoreExpiry=true, got %d", len(cache.cache))
		}
	})

	t.Run("cleanup preserves non-expired entries", func(t *testing.T) {
		cache := newModerationResultsCache()
		cache.cacheTTL = 100 * time.Millisecond

		// Add first entry and let it expire
		cache.setResultPending("expired_message")
		time.Sleep(150 * time.Millisecond)

		// Add second entry (fresh)
		cache.setResultPending("fresh_message")

		// Verify both entries exist
		if len(cache.cache) != 2 {
			t.Errorf("Expected 2 cache entries, got %d", len(cache.cache))
		}

		// Cleanup expired entries
		cache.cleanup(false)

		// Verify only fresh entry remains
		if len(cache.cache) != 1 {
			t.Errorf("Expected 1 cache entry after cleanup, got %d", len(cache.cache))
		}

		if _, ok := cache.cache["fresh_message"]; !ok {
			t.Error("Expected fresh_message to remain in cache")
		}

		if _, ok := cache.cache["expired_message"]; ok {
			t.Error("Expected expired_message to be removed from cache")
		}
	})

	t.Run("cleanup closes listener channels", func(t *testing.T) {
		cache := newModerationResultsCache()
		cache.cacheTTL = 100 * time.Millisecond

		// Add entry and create listener
		cache.setResultPending("message1")
		ch := make(chan *moderationResult, 1)
		cache.cacheLock.Lock()
		cache.listeners["message1"] = []chan *moderationResult{ch}
		cache.cacheLock.Unlock()

		// Wait for entry to expire
		time.Sleep(150 * time.Millisecond)

		// Cleanup expired entries
		cache.cleanup(false)

		// Verify listener channel is closed
		select {
		case _, ok := <-ch:
			if ok {
				t.Error("Expected listener channel to be closed")
			}
		default:
			t.Error("Expected listener channel to be closed and readable")
		}

		// Verify listeners map is cleaned up
		if len(cache.listeners) != 0 {
			t.Errorf("Expected 0 listeners after cleanup, got %d", len(cache.listeners))
		}
	})
}

func TestModerationResultsCache_waitForResult(t *testing.T) {
	t.Run("returns cached non-pending result immediately", func(t *testing.T) {
		cache := newModerationResultsCache()

		// Set a processed result
		cache.setModerationResultNotFlagged("message1")

		// Wait for result should return immediately
		start := time.Now()
		result := cache.waitForResult("message1", 1*time.Second)
		duration := time.Since(start)

		if result == nil {
			t.Error("Expected result, got nil")
		}
		if result.code != moderationResultProcessed {
			t.Errorf("Expected moderationResultProcessed, got %v", result.code)
		}
		if duration > 50*time.Millisecond {
			t.Errorf("Expected immediate return, took %v", duration)
		}
	})

	t.Run("waits for pending result and receives notification", func(t *testing.T) {
		cache := newModerationResultsCache()

		// Set pending result
		cache.setResultPending("message1")

		// Start waiting in goroutine
		resultCh := make(chan *moderationResult, 1)
		go func() {
			result := cache.waitForResult("message1", 1*time.Second)
			resultCh <- result
		}()

		// Give waitForResult time to set up listener
		time.Sleep(10 * time.Millisecond)

		// Complete the moderation
		cache.setModerationResultNotFlagged("message1")

		// Should receive the result
		select {
		case result := <-resultCh:
			if result == nil {
				t.Error("Expected result, got nil")
			}
			if result.code != moderationResultProcessed {
				t.Errorf("Expected moderationResultProcessed, got %v", result.code)
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for result")
		}
	})

	t.Run("times out waiting for result", func(t *testing.T) {
		cache := newModerationResultsCache()

		// Set pending result but never complete it
		cache.setResultPending("message1")

		start := time.Now()
		result := cache.waitForResult("message1", 100*time.Millisecond)
		duration := time.Since(start)

		if result != nil {
			t.Errorf("Expected nil result due to timeout, got %v", result.code)
		}
		if duration < 90*time.Millisecond || duration > 150*time.Millisecond {
			t.Errorf("Expected timeout around 100ms, got %v", duration)
		}
	})

	t.Run("waits for non-existent message", func(t *testing.T) {
		cache := newModerationResultsCache()

		// Don't add any message to cache
		start := time.Now()
		result := cache.waitForResult("nonexistent", 100*time.Millisecond)
		duration := time.Since(start)

		if result != nil {
			t.Errorf("Expected nil result due to timeout, got %v", result.code)
		}
		if duration < 90*time.Millisecond || duration > 150*time.Millisecond {
			t.Errorf("Expected timeout around 100ms, got %v", duration)
		}
	})

	t.Run("multiple waiters receive same result", func(t *testing.T) {
		cache := newModerationResultsCache()

		// Set pending result
		cache.setResultPending("message1")

		// Start multiple waiters
		const numWaiters = 3
		resultChs := make([]chan *moderationResult, numWaiters)
		for i := 0; i < numWaiters; i++ {
			resultChs[i] = make(chan *moderationResult, 1)
			go func(ch chan *moderationResult) {
				result := cache.waitForResult("message1", 1*time.Second)
				ch <- result
			}(resultChs[i])
		}

		// Give waiters time to set up listeners
		time.Sleep(10 * time.Millisecond)

		// Complete the moderation
		cache.setModerationResultNotFlagged("message1")

		// All waiters should receive the result
		for i := 0; i < numWaiters; i++ {
			select {
			case result := <-resultChs[i]:
				if result == nil {
					t.Errorf("Waiter %d: Expected result, got nil", i)
				}
				if result.code != moderationResultProcessed {
					t.Errorf("Waiter %d: Expected moderationResultProcessed, got %v", i, result.code)
				}
			case <-time.After(500 * time.Millisecond):
				t.Errorf("Waiter %d: Timeout waiting for result", i)
			}
		}
	})
}
