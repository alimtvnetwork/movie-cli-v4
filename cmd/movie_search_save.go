// movie_search_save.go — save selected search result to database and print summary
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// saveSearchResult builds a Media record from the selected TMDb result,
// fetches full details, downloads the thumbnail, and persists to the database.
func saveSearchResult(client *tmdb.Client, database *db.DB, selected tmdb.SearchResult) {
	title := selected.GetDisplayTitle()
	year := selected.GetYear()
	yearInt := 0
	if year != "" {
		yearInt, _ = strconv.Atoi(year)
	}

	fmt.Printf("\n⏳ Fetching full details for: %s...\n", title)

	m := &db.Media{
		Title:       title,
		CleanTitle:  title,
		Year:        yearInt,
		TmdbID:      selected.ID,
		TmdbRating:  selected.VoteAvg,
		Popularity:  selected.Popularity,
		Description: selected.Overview,
		Genre:       tmdb.GenreNames(selected.GenreIDs),
	}

	m.Type = resolveMediaType(selected.MediaType)
	fetchDetailsByType(client, selected.ID, m)

	downloadSearchThumbnail(client, database, selected, m)
	persistMedia(database, m)
	printSavedSummary(m)
}

// downloadSearchThumbnail downloads the poster image for a search result.
func downloadSearchThumbnail(client *tmdb.Client, database *db.DB, selected tmdb.SearchResult, m *db.Media) {
	if selected.PosterPath == "" {
		return
	}

	slug := cleaner.ToSlug(m.CleanTitle)
	if m.Year > 0 {
		slug += "-" + strconv.Itoa(m.Year)
	}

	thumbDir := filepath.Join(database.BasePath, "thumbnails", slug)
	if mkdirErr := os.MkdirAll(thumbDir, 0755); mkdirErr != nil {
		errlog.Warn("Cannot create thumbnail dir: %v", mkdirErr)
	}

	thumbPath := filepath.Join(thumbDir, slug+".jpg")
	if dlErr := client.DownloadPoster(selected.PosterPath, thumbPath); dlErr != nil {
		errlog.Warn("Thumbnail download failed: %v", dlErr)
		return
	}
	m.ThumbnailPath = thumbPath
	fmt.Println("🖼️  Thumbnail saved")
}

// resolveMediaType maps a TMDb media type string to the internal type.
func resolveMediaType(mediaType string) string {
	if mediaType == string(db.MediaTypeTV) {
		return string(db.MediaTypeTV)
	}
	return string(db.MediaTypeMovie)
}

// fetchDetailsByType fetches full details based on media type.
func fetchDetailsByType(client *tmdb.Client, tmdbID int, m *db.Media) {
	if m.Type == string(db.MediaTypeTV) {
		fetchTVDetails(client, tmdbID, m)
		return
	}
	fetchMovieDetails(client, tmdbID, m)
}

// persistMedia inserts (or updates) the media record and links genres.
func persistMedia(database *db.DB, m *db.Media) {
	jsonDir := filepath.Join(database.BasePath, "json", m.Type)
	if mkdirErr := os.MkdirAll(jsonDir, 0755); mkdirErr != nil {
		errlog.Warn("Cannot create JSON dir: %v", mkdirErr)
	}

	mediaID, insertErr := database.InsertMedia(m)
	if insertErr != nil {
		persistMediaUpdate(database, m, insertErr)
		return
	}
	linkGenresOnInsert(database, mediaID, m.Genre)
}

func persistMediaUpdate(database *db.DB, m *db.Media, originalErr error) {
	if m.TmdbID <= 0 {
		errlog.Error("DB error: %v", originalErr)
		return
	}
	updateErr := database.UpdateMediaByTmdbID(m)
	if updateErr != nil {
		errlog.Error("DB error: %v", updateErr)
		return
	}
	fmt.Printf("🔄 Updated existing record for: %s\n", m.Title)
	replaceGenresOnUpdate(database, m.TmdbID, m.Genre)
}

func replaceGenresOnUpdate(database *db.DB, tmdbID int, genre string) {
	if genre == "" {
		return
	}
	existing, _ := database.GetMediaByTmdbID(tmdbID)
	if existing == nil {
		return
	}
	database.ReplaceMediaGenres(existing.ID, genre)
}

func linkGenresOnInsert(database *db.DB, mediaID int64, genre string) {
	if mediaID <= 0 || genre == "" {
		return
	}
	if linkErr := database.LinkMediaGenres(mediaID, genre); linkErr != nil {
		errlog.Warn("Genre link error: %v", linkErr)
	}
}

// printSavedSummary prints the saved media summary to stdout.
func printSavedSummary(m *db.Media) {
	typeIcon := db.TypeIcon(m.Type)
	typeLabel := db.TypeLabel(m.Type)
	folder := db.JSONSubDir(m.Type)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ Saved to database!\n\n")
	fmt.Printf("  %s  %s (%s)\n", typeIcon, m.Title, typeLabel)
	fmt.Printf("  📅  Year: %d\n", m.Year)
	fmt.Printf("  ⭐  Rating: %.1f\n", m.TmdbRating)
	fmt.Printf("  🎭  Genre: %s\n", m.Genre)

	if m.Director != "" {
		fmt.Printf("  🎬  Director: %s\n", m.Director)
	}
	if m.CastList != "" {
		fmt.Printf("  👥  Cast: %s\n", m.CastList)
	}
	if m.Description != "" {
		desc := m.Description
		if len(desc) > 150 {
			desc = desc[:147] + "..."
		}
		fmt.Printf("  📝  %s\n", desc)
	}

	fmt.Printf("  📁  Stored in: %s/ folder\n", folder)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
