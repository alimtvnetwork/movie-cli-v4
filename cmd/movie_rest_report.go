// movie_rest_report.go — HTML report rendering and media CRUD handlers.
package cmd

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/templates"
)

func handleMediaByID(w http.ResponseWriter, r *http.Request, database *db.DB) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/media/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleMediaGet(w, database, id)
	case http.MethodDelete:
		handleMediaDelete(w, database, id)
	case http.MethodPatch:
		handleMediaPatch(w, r, database, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleMediaGet(w http.ResponseWriter, database *db.DB, id int64) {
	m, getErr := database.GetMediaByID(id)
	if getErr != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, m)
}

func handleMediaDelete(w http.ResponseWriter, database *db.DB, id int64) {
	if delErr := database.DeleteMedia(id); delErr != nil {
		http.Error(w, delErr.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "deleted"})
}

func handleMediaPatch(w http.ResponseWriter, r *http.Request, database *db.DB, id int64) {
	var updates map[string]interface{}
	if decErr := json.NewDecoder(r.Body).Decode(&updates); decErr != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	for key, val := range updates {
		applyMediaUpdate(database, id, key, val)
	}
	m, _ := database.GetMediaByID(id)
	writeJSON(w, m)
}

func applyMediaUpdate(database *db.DB, id int64, key string, val interface{}) {
	switch key {
	case "genre":
		if genreStr, ok := val.(string); ok {
			database.ReplaceMediaGenres(id, genreStr)
		}
	case "title", "director", "description", "tagline":
		if _, execErr := database.Exec("UPDATE Media SET "+key+" = ?, UpdatedAt = datetime('now') WHERE MediaId = ?", val, id); execErr != nil {
			errlog.Error("DB update error for media %d field %s: %v", id, key, execErr)
		}
	}
}

// serveHTMLReport renders the HTML report template with live data from the database.
func serveHTMLReport(w http.ResponseWriter, database *db.DB, port int) {
	tmplBytes, err := templates.FS.ReadFile("report.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	tmpl, parseErr := template.New("report").Parse(string(tmplBytes))
	if parseErr != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		return
	}

	items, listErr := database.ListMedia(0, 10000)
	if listErr != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	data := buildReportData(items, port)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		errlog.Error("template render error: %v", err)
	}
}

func buildReportData(items []db.Media, port int) htmlReportData {
	movies, tv := 0, 0
	reportItems := make([]htmlReportItem, 0, len(items))
	for _, m := range items {
		if m.Type == string(db.MediaTypeMovie) {
			movies++
		} else {
			tv++
		}
		reportItems = append(reportItems, buildHTMLReportItem(m))
	}

	return htmlReportData{
		ScannedFolder: "Library",
		ScannedAt:     "Live",
		TotalFiles:    len(items),
		Movies:        movies,
		TVShows:       tv,
		Port:          port,
		Items:         reportItems,
	}
}

func buildHTMLReportItem(m db.Media) htmlReportItem {
	var genres []string
	if m.Genre != "" {
		for _, g := range splitGenres(m.Genre) {
			genres = append(genres, g)
		}
	}
	thumbSrc := ""
	if m.ThumbnailPath != "" {
		thumbSrc = "/thumbnails/" + filepath.Base(m.ThumbnailPath)
	}
	return htmlReportItem{
		ID:            m.ID,
		Title:         m.Title,
		Year:          m.Year,
		Type:          m.Type,
		Genre:         m.Genre,
		GenreList:     genres,
		Director:      m.Director,
		CastList:      m.CastList,
		Description:   m.Description,
		Tagline:       m.Tagline,
		TmdbRating:    m.TmdbRating,
		ImdbRating:    m.ImdbRating,
		Runtime:       m.Runtime,
		ThumbnailPath: thumbSrc,
	}
}

// logMiddleware wraps an http.Handler and logs every request via errlog.
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)
		duration := time.Since(start)
		errlog.Info("[REST] %s %s → %d (%s)", r.Method, r.URL.Path, lrw.statusCode, duration)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
