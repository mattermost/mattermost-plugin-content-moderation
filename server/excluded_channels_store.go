package main

import (
	"encoding/json"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const excludedChannelsKVKey = "excluded_channels_list"

type ExcludedChannelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ExcludedChannelsStore interface {
	SetExcluded(channelID string, excluded bool) error
	IsExcluded(channelID string) (bool, error)
	ListExcluded() []ExcludedChannelInfo
}

type excludedChannelsStore struct {
	cacheLock sync.Mutex
	cache     map[string]ExcludedChannelInfo
	loaded    bool
	api       plugin.API
}

func newExcludedChannelsStore(api plugin.API) (*excludedChannelsStore, error) {
	s := &excludedChannelsStore{
		cache: make(map[string]ExcludedChannelInfo),
		api:   api,
	}
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	if err := s.loadCacheWithoutLock(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *excludedChannelsStore) SetExcluded(channelID string, excluded bool) error {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()

	if excluded {
		channel, err := s.api.GetChannel(channelID)
		if err != nil {
			return err
		}
		s.cache[channelID] = ExcludedChannelInfo{
			ID:   channelID,
			Name: channel.Name,
		}
	} else {
		delete(s.cache, channelID)
	}

	if err := s.saveCacheWithoutLock(); err != nil {
		return err
	}

	return nil
}

func (s *excludedChannelsStore) IsExcluded(channelID string) (bool, error) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	_, excluded := s.cache[channelID]
	return excluded, nil
}

func (s *excludedChannelsStore) ListExcluded() []ExcludedChannelInfo {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	var excludedChannels []ExcludedChannelInfo
	for _, channelInfo := range s.cache {
		excludedChannels = append(excludedChannels, channelInfo)
	}
	return excludedChannels
}

func (s *excludedChannelsStore) loadCacheWithoutLock() error {
	if s.loaded {
		return nil
	}

	var appErr *model.AppError
	data, appErr := s.api.KVGet(excludedChannelsKVKey)
	if appErr != nil {
		return appErr
	}
	if data == nil {
		s.loaded = true
		return nil
	}

	var excludedChannels []ExcludedChannelInfo
	if err := json.Unmarshal(data, &excludedChannels); err != nil {
		return err
	}

	for _, channelInfo := range excludedChannels {
		s.cache[channelInfo.ID] = channelInfo
	}

	s.loaded = true
	return nil
}

func (s *excludedChannelsStore) saveCacheWithoutLock() error {
	var excludedChannels []ExcludedChannelInfo
	for _, channelInfo := range s.cache {
		excludedChannels = append(excludedChannels, channelInfo)
	}

	data, err := json.Marshal(excludedChannels)
	if err != nil {
		return err
	}

	if appErr := s.api.KVSet(excludedChannelsKVKey, data); appErr != nil {
		return appErr
	}

	return nil
}
