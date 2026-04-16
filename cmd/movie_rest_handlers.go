// movie_rest_handlers.go — additional REST API handlers for tags, similar, and watched.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// handleTags handles POST /api/tags (add/remove tags).
//
//	POST body: {"media_id": 1, "tag": "favorite"}
//	DELETE body: {"media_id": 1, "tag": "favorite"}
//	GET /api/tags?media_id=1 — list tags for a media item
//	GET /api/tags — list all tags with counts
func handleTags(w http.ResponseWriter, r *http.Request, database *db.DB) {
	switch r.Method {
	case http.MethodGet:
		idStr := r.URL.Query().Get("media_id")
		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid media_id", http.StatusBadRequest)
				return
			}
			tags, tagErr := database.GetTagsByMediaID(id)
			if tagErr != nil {
				http.Error(w, tagErr.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"media_id": id, "tags": tags})
		} else {
			counts, cErr := database.GetAllTagCounts()
			if cErr != nil {
				http.Error(w, cErr.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, counts)
		}

	case http.MethodPost:
		var req struct {
			MediaID int    `json:"media_id"`
			Tag     string `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.MediaID == 0 || req.Tag == "" {
			http.Error(w, "media_id and tag are required", http.StatusBadRequest)
			return
		}
		if err := database.AddTag(req.MediaID, req.Tag); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, map[string]string{"status": "added", "tag": req.Tag})

	case http.MethodDelete:
		var req struct {
			MediaID int    `json:"media_id"`
			Tag     string `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		removed, err := database.RemoveTag(req.MediaID, req.Tag)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !removed {
			http.Error(w, "tag not found", http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]string{"status": "removed", "tag": req.Tag})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSimilar handles GET /api/media/{id}/similar — fetches TMDb recommendations.
func handleSimilar(w http.ResponseWriter, r *http.Request, database *db.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse ID from path: /api/media/{id}/similar
	path := strings.TrimPrefix(r.URL.Path, "/api/media/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "similar" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	m, getErr := database.GetMediaByID(id)
	if getErr != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if m.TmdbID == 0 {
		writeJSON(w, map[string]interface{}{"media_id": id, "similar": []interface{}{}, "message": "no TMDb ID available"})
		return
	}

	// Get TMDb credentials — missing keys are not fatal (empty string = no auth)
	apiKey, keyErr := database.GetConfig("TmdbApiKey")
	if keyErr != nil {
		errlog.Warn("Could not read tmdb_api_key: %v", keyErr)
	}
	token, tokErr := database.GetConfig("TmdbToken")
	if tokErr != nil {
		errlog.Warn("Could not read tmdb_token: %v", tokErr)
	}
	client := tmdb.NewClientWithToken(apiKey, token)

	results, recErr := client.GetRecommendations(m.TmdbID, m.Type, 1)
	if recErr != nil {
		http.Error(w, fmt.Sprintf("TMDb error: %v", recErr), http.StatusBadGateway)
		return
	}

	writeJSON(w, map[string]interface{}{
		"media_id": id,
		"title":    m.Title,
		"similar":  results,
	})
}

// handleWatched handles PATCH /api/media/{id}/watched — marks a media item as watched.
func handleWatched(w http.ResponseWriter, r *http.Request, database *db.DB) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse ID from path: /api/media/{id}/watched
	path := strings.TrimPrefix(r.URL.Path, "/api/media/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "watched" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Check media exists
	m, getErr := database.GetMediaByID(id)
	if getErr != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Update watchlist status if in watchlist
	if _, wlErr := database.Exec("UPDATE watchlist SET status = 'watched', watched_at = CURRENT_TIMESTAMP WHERE tmdb_id = ?", m.TmdbID); wlErr != nil {
		errlog.Error("watchlist update error for media %d: %v", id, wlErr)
	}

	// Also add a "watched" tag
	if tagErr := database.AddTag(int(id), "watched"); tagErr != nil {
		errlog.Warn("could not add watched tag for media %d: %v", id, tagErr)
	}

	writeJSON(w, map[string]interface{}{
		"status":   "marked_watched",
		"media_id": id,
		"title":    m.Title,
	})
}

// handleLogs handles GET /api/logs — returns error logs with optional filtering.
//
//	Query params:
//	  level  — filter by level: ERROR, WARN, INFO (optional)
//	  limit  — max entries to return (default 50, max 500)
//	  search — substring match on message (optional)
func handleLogs(w http.ResponseWriter, r *http.Request, database *db.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 500 {
		limit = 500
	}

	entries, err := database.RecentErrorLogs(limit)
	if err != nil {
		errlog.Error("Failed to read error logs: %v", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	// Filter by level
	level := strings.ToUpper(r.URL.Query().Get("level"))
	search := strings.ToLower(r.URL.Query().Get("search"))

	if level != "" || search != "" {
		var filtered []map[string]string
		for _, e := range entries {
			if level != "" && e["level"] != level {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(e["message"]), search) {
				continue
			}
			filtered = append(filtered, e)
		}
		entries = filtered
	}

	writeJSON(w, map[string]interface{}{
		"total":   len(entries),
		"entries": entries,
	})
}
