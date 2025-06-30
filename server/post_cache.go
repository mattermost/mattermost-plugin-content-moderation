package main

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
)

const postCacheTTL = 5 * time.Minute

type postCache struct {
	cache     map[string]cachedPost
	cacheTTL  time.Duration
	cacheLock sync.Mutex
}

type cachedPost struct {
	post      *model.Post
	timestamp time.Time
}

func newPostCache() *postCache {
	return &postCache{
		cache:    make(map[string]cachedPost),
		cacheTTL: postCacheTTL,
	}
}

func (pc *postCache) setPost(post *model.Post) {
	if post.Id == "" {
		return
	}

	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	pc.cache[post.Id] = cachedPost{
		post:      post,
		timestamp: time.Now(),
	}
}

func (pc *postCache) getPost(api plugin.API, postID string) (*model.Post, error) {
	if postID == "" {
		return nil, errors.New("post ID cannot be empty")
	}

	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	if cached, ok := pc.cache[postID]; ok {
		return cached.post, nil
	}

	post, err := api.GetPost(postID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve post from API")
	}

	if post == nil {
		return nil, errors.New("post not found")
	}

	pc.cache[postID] = cachedPost{
		post:      post,
		timestamp: time.Now(),
	}

	return post, nil
}

func (pc *postCache) cleanup(ignoreExpiry bool) {
	pc.cacheLock.Lock()
	defer pc.cacheLock.Unlock()

	now := time.Now()
	for id, post := range pc.cache {
		if ignoreExpiry || now.Sub(post.timestamp) > pc.cacheTTL {
			delete(pc.cache, id)
		}
	}
}
