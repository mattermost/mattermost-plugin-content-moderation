package main

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-content-moderation/server/moderation"
)

type moderationResultCode int

const moderationResultsCacheTTL = 5 * time.Minute

const (
	moderationResultPending moderationResultCode = iota
	moderationResultProcessed
	moderationResultFlagged
	moderationResultError
)

type moderationResultsCache struct {
	cache     map[string]*moderationResult
	listeners map[string][]chan *moderationResult
	cacheTTL  time.Duration
	cacheLock sync.Mutex
}

type moderationResult struct {
	code      moderationResultCode
	result    moderation.Result
	err       error
	timestamp time.Time
}

func newModerationResultsCache() *moderationResultsCache {
	return &moderationResultsCache{
		cache:     make(map[string]*moderationResult),
		listeners: make(map[string][]chan *moderationResult),
		cacheTTL:  moderationResultsCacheTTL,
	}
}

// setResultPending will add a new message to the cache with a pending
// status code. If the message already exists in the cache, we'll just
// updated the timestamp to keep it around longer. We won't mark it
// pending - there's no reason to recompute the result. Returns true
// if we created a new entry in the pending state.
func (pc *moderationResultsCache) setResultPending(message string) bool {
	if message == "" {
		return false
	}

	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	// We may already have a result in the cache
	// so let's just refresh the timestamp
	if result, ok := pc.cache[message]; ok {
		result.timestamp = time.Now()
		return false
	}

	pc.cache[message] = &moderationResult{
		code:      moderationResultPending,
		timestamp: time.Now(),
	}

	return true
}

func (pc *moderationResultsCache) setModerationResultError(message string, err error) {
	if message == "" {
		return
	}

	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	result := &moderationResult{
		code:      moderationResultError,
		timestamp: time.Now(),
		err:       err,
	}
	pc.cache[message] = result
	pc.notifyListeners(message, result)
}

func (pc *moderationResultsCache) setModerationResultNotFlagged(message string, result moderation.Result) {
	if message == "" {
		return
	}

	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	moderationResult := &moderationResult{
		code:      moderationResultProcessed,
		result:    result,
		timestamp: time.Now(),
	}
	pc.cache[message] = moderationResult
	pc.notifyListeners(message, moderationResult)
}

func (pc *moderationResultsCache) setModerationResultFlagged(message string, result moderation.Result) {
	if message == "" {
		return
	}

	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	moderationResult := &moderationResult{
		code:      moderationResultFlagged,
		result:    result,
		timestamp: time.Now(),
	}
	pc.cache[message] = moderationResult
	pc.notifyListeners(message, moderationResult)
}

func (pc *moderationResultsCache) waitForResult(message string, timeout time.Duration) *moderationResult {
	pc.cacheLock.Lock()
	if cached, ok := pc.cache[message]; ok && cached.code != moderationResultPending {
		pc.cacheLock.Unlock()
		return cached
	}
	ch := make(chan *moderationResult, 1)
	pc.listeners[message] = append(pc.listeners[message], ch)
	pc.cacheLock.Unlock()

	select {
	case result := <-ch:
		return result
	case <-time.After(timeout):
		return nil
	}
}

// notifyListeners notifies all registered listeners for a message and cleans them up.
// IMPORTANT: This method assumes the caller already holds pc.cacheLock.
func (pc *moderationResultsCache) notifyListeners(message string, result *moderationResult) {
	listeners, ok := pc.listeners[message]
	if !ok {
		return
	}

	for _, ch := range listeners {
		select {
		case ch <- result:
		default:
		}
		close(ch)
	}

	delete(pc.listeners, message)
}

func (pc *moderationResultsCache) cleanup(ignoreExpiry bool) {
	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	now := time.Now()
	for message, result := range pc.cache {
		if ignoreExpiry || now.Sub(result.timestamp) > pc.cacheTTL {
			delete(pc.cache, message)
			for _, ch := range pc.listeners[message] {
				close(ch)
			}
			delete(pc.listeners, message)
		}
	}
}
