package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type contextKey string

const (
	contextKeyUserID        contextKey = "userID"
	contextKeyChannelID     contextKey = "channelID"
	contextKeyPluginContext contextKey = "pluginContext"
)

type ModerationStatusResponse struct {
	Excluded bool `json:"excluded"`
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()

	apiRouter := router.PathPrefix("/channels/{channelId}/moderation").Subrouter()
	apiRouter.HandleFunc("/enable", p.requireChannelPermission(c, p.handleEnableChannelModeration)).Methods("POST")
	apiRouter.HandleFunc("/disable", p.requireChannelPermission(c, p.handleDisableChannelModeration)).Methods("POST")
	apiRouter.HandleFunc("/status", p.requireChannelPermission(c, p.handleGetChannelModerationStatus)).Methods("GET")

	router.ServeHTTP(w, r)
}

// requireChannelPermission is a middleware that handles authentication and authorization
func (p *Plugin) requireChannelPermission(pluginContext *plugin.Context, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		channelID := vars["channelId"]

		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !p.hasChannelPermission(userID, channelID) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Add userID, channelID, and pluginContext to context for use in handlers
		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		ctx = context.WithValue(ctx, contextKeyChannelID, channelID)
		ctx = context.WithValue(ctx, contextKeyPluginContext, pluginContext)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

func (p *Plugin) handleEnableChannelModeration(w http.ResponseWriter, r *http.Request) {
	// Get userID, channelID, and pluginContext from context (set by middleware)
	userID := r.Context().Value(contextKeyUserID).(string)
	channelID := r.Context().Value(contextKeyChannelID).(string)
	pluginContext := r.Context().Value(contextKeyPluginContext).(*plugin.Context)

	auditRecord := plugin.MakeAuditRecordWithContext(auditEventTypeManageChannelModeration, model.AuditStatusAttempt, pluginContext, userID, r.URL.Path)
	auditRecord.AddMeta(auditMetaKeyChannelID, channelID)
	auditRecord.AddMeta(auditMetaKeyUserID, userID)
	auditRecord.AddMeta(auditMetaKeyAction, "enable")

	if p.getConfiguration().AuditLoggingEnabled {
		defer p.API.LogAuditRec(auditRecord)
	}

	err := p.excludedChannelStore.SetExcluded(channelID, false)
	if err != nil {
		p.API.LogError("Failed to enable channel moderation", "channel_id", channelID, "user_id", userID, "err", err)
		auditRecord.AddErrorDesc(err.Error())
		auditRecord.Fail()
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	p.API.LogInfo("Channel moderation enabled via API", "channel_id", channelID, "user_id", userID)
	auditRecord.Success()

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) handleDisableChannelModeration(w http.ResponseWriter, r *http.Request) {
	// Get userID, channelID, and pluginContext from context (set by middleware)
	userID := r.Context().Value(contextKeyUserID).(string)
	channelID := r.Context().Value(contextKeyChannelID).(string)
	pluginContext := r.Context().Value(contextKeyPluginContext).(*plugin.Context)

	auditRecord := plugin.MakeAuditRecordWithContext(auditEventTypeManageChannelModeration, model.AuditStatusAttempt, pluginContext, userID, r.URL.Path)
	auditRecord.AddMeta(auditMetaKeyChannelID, channelID)
	auditRecord.AddMeta(auditMetaKeyUserID, userID)
	auditRecord.AddMeta(auditMetaKeyAction, "disable")

	if p.getConfiguration().AuditLoggingEnabled {
		defer p.API.LogAuditRec(auditRecord)
	}

	err := p.excludedChannelStore.SetExcluded(channelID, true)
	if err != nil {
		p.API.LogError("Failed to disable channel moderation", "channel_id", channelID, "user_id", userID, "err", err)
		auditRecord.AddErrorDesc(err.Error())
		auditRecord.Fail()
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	p.API.LogInfo("Channel moderation disabled via API", "channel_id", channelID, "user_id", userID)
	auditRecord.Success()

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) handleGetChannelModerationStatus(w http.ResponseWriter, r *http.Request) {
	// Get userID and channelID from context (set by middleware)
	userID := r.Context().Value(contextKeyUserID).(string)
	channelID := r.Context().Value(contextKeyChannelID).(string)

	excluded, err := p.excludedChannelStore.IsExcluded(channelID)
	if err != nil {
		p.API.LogError("Failed to get channel moderation status", "channel_id", channelID, "user_id", userID, "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ModerationStatusResponse{
		Excluded: excluded,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		p.API.LogError("Failed to encode response", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
