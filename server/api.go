package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// ServeHTTP handles HTTP requests to the plugin
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	if !p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/channels/search", p.searchChannels).Methods(http.MethodGet)
	router.ServeHTTP(w, r)
}

// searchChannels handles the channel search API endpoint
func (p *Plugin) searchChannels(w http.ResponseWriter, r *http.Request) {
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
	if prefix == "" {
		http.Error(w, "missing search prefix", http.StatusBadRequest)
		return
	}

	channels, err := p.sqlStore.SearchChannelsByPrefix(prefix)
	if err != nil {
		http.Error(w, "failed to search channels", http.StatusInternalServerError)
		p.API.LogError("failed to search channels", "error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(channels); err != nil {
		p.API.LogError("failed to write http response", "error", err.Error())
	}
}
