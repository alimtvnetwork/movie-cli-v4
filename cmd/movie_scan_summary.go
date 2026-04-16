// movie_scan_summary.go — writes .movie-output/summary.json after a scan.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"github.com/alimtvnetwork/movie-cli-v4/db"
)

// scanSummary is the top-level structure written to summary.json.
type scanSummary struct {
	ScannedFolder string              `json:"scanned_folder"`
	ScannedAt     string              `json:"scanned_at"`
	TotalFiles    int                 `json:"total_files"`
	Movies        int                 `json:"movies"`
	TVShows       int                 `json:"tv_shows"`
	Skipped       int                 `json:"skipped"`
	Categories    map[string][]string `json:"categories"`
	Items         []scanSummaryItem   `json:"items"`
}

// scanSummaryItem is per-media metadata in the summary.
type scanSummaryItem struct {
	Title       string  `json:"title"`
	Year        int     `json:"year,omitempty"`
	Type        string  `json:"type"`
	Genre       string  `json:"genre,omitempty"`
	Director    string  `json:"director,omitempty"`
	CastList    string  `json:"cast_list,omitempty"`
	Description string  `json:"description,omitempty"`
	TmdbID      int     `json:"tmdb_id,omitempty"`
	ImdbID      string  `json:"imdb_id,omitempty"`
	TmdbRating  float64 `json:"tmdb_rating,omitempty"`
	ImdbRating  float64 `json:"imdb_rating,omitempty"`
	Runtime     int     `json:"runtime,omitempty"`
	Language    string  `json:"language,omitempty"`
	Tagline     string  `json:"tagline,omitempty"`
	TrailerURL  string  `json:"trailer_url,omitempty"`
	FilePath    string  `json:"file_path"`
	FileName    string  `json:"file_name"`
	FileSize    int64   `json:"file_size,omitempty"`
}

// writeScanSummary writes .movie-output/summary.json with the full scan report.
func writeScanSummary(stats ScanStats) error {
	summaryItems, categories := buildSummaryItems(stats.Items)

	summary := scanSummary{
		ScannedFolder: stats.ScanDir,
		ScannedAt:     time.Now().Format(time.RFC3339),
		TotalFiles:    stats.Total,
		Movies:        stats.Movies,
		TVShows:       stats.TV,
		Skipped:       stats.Skipped,
		Categories:    categories,
		Items:         summaryItems,
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return apperror.Wrap("json encode", err)
	}

	outPath := filepath.Join(stats.OutputDir, "summary.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return apperror.Wrap("write file", err)
	}
	return nil
}

func buildSummaryItems(media []db.Media) ([]scanSummaryItem, map[string][]string) {
	categories := make(map[string][]string)
	items := make([]scanSummaryItem, 0, len(media))

	for _, m := range media {
		items = append(items, scanSummaryItem{
			Title: m.Title, Year: m.Year, Type: m.Type,
			Genre: m.Genre, Director: m.Director, CastList: m.CastList,
			Description: m.Description, TmdbID: m.TmdbID, ImdbID: m.ImdbID,
			TmdbRating: m.TmdbRating, ImdbRating: m.ImdbRating,
			Runtime: m.Runtime, Language: m.Language, Tagline: m.Tagline,
			TrailerURL: m.TrailerURL, FilePath: m.CurrentFilePath,
			FileName: m.OriginalFileName, FileSize: m.FileSize,
		})
		categorizeByGenre(categories, m)
	}
	return items, categories
}

func categorizeByGenre(categories map[string][]string, m db.Media) {
	if m.Genre == "" {
		return
	}
	for _, g := range splitGenres(m.Genre) {
		display := m.Title
		if m.Year > 0 {
			display = fmt.Sprintf("%s (%d)", m.Title, m.Year)
		}
		categories[g] = append(categories[g], display)
	}
}
