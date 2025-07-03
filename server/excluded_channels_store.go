package main

import (
	"strconv"
	"sync"

	"github.com/mattermost/mattermost/server/public/plugin"
)

const excludedChannelsKVKeyPrefix = "excluded_channels_"

type ExcludedChannelsStore interface {
	SetExcluded(channelID string, excluded bool) error
	IsExcluded(channelID string) (bool, error)
}

type excludedChannelsStore struct {
	cacheLock sync.Mutex
	cache     map[string]bool
	api       plugin.API
}

func newExcludedChannelsStore(api plugin.API) *excludedChannelsStore {
	return &excludedChannelsStore{
		cache: make(map[string]bool),
		api:   api,
	}
}

func (s *excludedChannelsStore) SetExcluded(channelID string, excluded bool) error {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()

	key := excludedChannelsKVKeyPrefix + channelID
	err := s.api.KVSet(key, []byte(strconv.FormatBool(excluded)))
	if err != nil {
		return err
	}
	s.cache[key] = excluded
	return nil
}

func (s *excludedChannelsStore) IsExcluded(channelID string) (bool, error) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()

	key := excludedChannelsKVKeyPrefix + channelID
	// check the cache first
	excluded, ok := s.cache[key]
	if ok {
		return excluded, nil
	}
	// fall back to the KV store
	excludedBytes, appErr := s.api.KVGet(key)
	if appErr != nil {
		return false, appErr
	}
	// if that's empty, return false (and cache that)
	if excludedBytes == nil {
		s.cache[key] = false
		return false, nil
	}
	// if it's not empty, cache and return the value
	excluded, err := strconv.ParseBool(string(excludedBytes))
	if err != nil {
		return false, err
	}
	s.cache[key] = excluded
	return excluded, nil
}
