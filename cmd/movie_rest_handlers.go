// movie_rest_handlers.go — additional REST API handlers for tags, similar, and watched.
package cmd

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

func handleTags(w http.ResponseWriter, r *http.Request, database *db.DB) {
	switch r.Method {
	case http.MethodGet:
		handleTagsGet(w, r, database)
	case http.MethodPost:
		handleTagsPost(w, r, database)
	case http.MethodDelete:
		handleTagsDelete(w, r, database)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTagsGet(w http.ResponseWriter, r *http.Request, database *db.DB) {
	idStr := r.URL.Query().Get("media_id")
	if idStr == "" {
		counts, cErr := database.GetAllTagCounts()
		if cErr != nil {
			http.Error(w, cErr.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, counts)
		return
	}
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
}

func handleTagsPost(w http.ResponseWriter, r *http.Request, database *db.DB) {
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
}

func handleTagsDelete(w http.ResponseWriter, r *http.Request, database *db.DB) {
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
}

// handleSimilar handles GET /api/media/{id}/similar — fetches TMDb recommendations.
func handleSimilar(w http.ResponseWriter, r *http.Request, database *db.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := parseMediaSubpath(r.URL.Path, "similar")
	if id <= 0 {
		http.Error(w, "invalid path", http.StatusBadRequest)
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

	results := fetchSimilarFromTMDb(database, m)
	if results == nil {
		http.Error(w, "TMDb error", http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]interface{}{"media_id": id, "title": m.Title, "similar": results})
}

func fetchSimilarFromTMDb(database *db.DB, m *db.Media) []tmdb.SearchResult {
	apiKey, _ := database.GetConfig("TmdbApiKey")
	token, _ := database.GetConfig("TmdbToken")
	client := tmdb.NewClientWithToken(apiKey, token)

	results, recErr := client.GetRecommendations(m.TmdbID, m.Type, 1)
	if recErr != nil {
		errlog.Error("TMDb recommendations error: %v", recErr)
		return nil
	}
	return results
}

func parseMediaSubpath(urlPath, suffix string) int64 {
	path := strings.TrimPrefix(urlPath, "/api/media/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != suffix {
		return 0
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// handleWatched handles PATCH /api/media/{id}/watched.
func handleWatched(w http.ResponseWriter, r *http.Request, database *db.DB) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := parseMediaSubpath(r.URL.Path, "watched")
	if id <= 0 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	m, getErr := database.GetMediaByID(id)
	if getErr != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	markMediaWatched(database, id, m.TmdbID)
	writeJSON(w, map[string]interface{}{"status": "marked_watched", "media_id": id, "title": m.Title})
}

func markMediaWatched(database *db.DB, id int64, tmdbID int) {
	if _, wlErr := database.Exec("UPDATE watchlist SET status = 'watched', watched_at = CURRENT_TIMESTAMP WHERE tmdb_id = ?", tmdbID); wlErr != nil {
		errlog.Error("watchlist update error for media %d: %v", id, wlErr)
	}
	if tagErr := database.AddTag(int(id), "watched"); tagErr != nil {
		errlog.Warn("could not add watched tag for media %d: %v", id, tagErr)
	}
}

// handleLogs handles GET /api/logs.
func handleLogs(w http.ResponseWriter, r *http.Request, database *db.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := parseLogsLimit(r)
	entries, err := database.RecentErrorLogs(limit)
	if err != nil {
		errlog.Error("Failed to read error logs: %v", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	entries = filterRESTLogEntries(r, entries)
	writeJSON(w, map[string]interface{}{"total": len(entries), "entries": entries})
}

func parseLogsLimit(r *http.Request) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return 50
	}
	parsed, err := strconv.Atoi(limitStr)
	if err != nil || parsed <= 0 {
		return 50
	}
	if parsed > 500 {
		return 500
	}
	return parsed
}

func filterRESTLogEntries(r *http.Request, entries []map[string]string) []map[string]string {
	level := strings.ToUpper(r.URL.Query().Get("level"))
	search := strings.ToLower(r.URL.Query().Get("search"))
	if level == "" && search == "" {
		return entries
	}
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
	return filtered
}

// splitGenres splits a comma-separated genre string.
func splitGenres(genres string) []string {
	var out []string
	for _, g := range strings.Split(genres, ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			out = append(out, g)
		}
	}
	return out
}
