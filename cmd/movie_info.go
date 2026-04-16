// movie_info.go — movie info <id-or-title>
//
// Accepts a numeric ID (from library) or a title string.
// Checks local DB first; if not found by title, falls back to TMDb API,
// fetches full details, stores in DB, then displays.
//
// Shared TMDb fetch helpers (fetchMovieDetails, fetchTVDetails) live in
// movie_fetch_details.go.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/cleaner"
	"github.com/alimtvnetwork/movie-cli-v3/apperror"
	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
	"github.com/alimtvnetwork/movie-cli-v3/tmdb"
)

var infoFormat string

var movieInfoCmd = &cobra.Command{
	Use:   "info [id or title]",
	Short: "Show detailed info for a movie or TV show",
	Long: `Display full metadata for a media item.

If a numeric ID is given, it looks up the item from your local library.
If a title is given, it first searches the local database. If not found,
it queries the TMDb API, saves the result, and then displays it.

Use --format json to output the result as JSON to stdout.
Use --format table to output the result as a formatted table.`,
	Args: cobra.MinimumNArgs(1),
	Run:  runMovieInfo,
}

func init() {
	movieInfoCmd.Flags().StringVar(&infoFormat, "format", "", "Output format: json, table")
}



func runMovieInfo(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	query := strings.Join(args, " ")

	// 1) Try local DB first (by ID or title)
	m, resolveErr := resolveMediaByQuery(database, query)
	if resolveErr == nil {
		printInfoResult(m, "local")
		return
	}

	// 2) Not in DB — fall back to TMDb API
	m = infoFetchFromTMDb(database, query)
	if m == nil {
		return
	}

	infoSaveAndDisplay(database, m)
}

// printInfoResult outputs a media item in the requested format.
func printInfoResult(m *db.Media, source string) {
	switch db.OutputFormat(infoFormat) {
	case db.OutputFormatJSON:
		printMediaDetailJSON(m, source)
	case db.OutputFormatTable:
		printMediaDetailTable(m)
	default:
		if source == "local" {
			fmt.Println("📚 Found in local library:")
		} else {
			fmt.Println("✅ Saved to your library!")
		}
		fmt.Println()
		printMediaDetail(m)
	}
}

// infoFetchFromTMDb searches TMDb, fetches details, and returns a populated Media.
func infoFetchFromTMDb(database *db.DB, query string) *db.Media {
	fmt.Printf("🔎 Not found locally. Searching TMDb for: %s\n\n", query)

	client, clientErr := buildTMDbClient(database)
	if clientErr != nil {
		errlog.Error("%v", clientErr)
		return nil
	}

	tmdbResults, searchErr := client.SearchMulti(query)
	if searchErr != nil {
		errlog.Error("TMDb search error: %v", searchErr)
		return nil
	}
	if len(tmdbResults) == 0 {
		fmt.Println("📭 No results found on TMDb either.")
		return nil
	}

	selected := tmdbResults[0]

	// Check if already in DB by TMDb ID
	existing, existErr := database.GetMediaByTmdbID(selected.ID)
	if existErr != nil && existErr.Error() != "sql: no rows in result set" {
		errlog.Warn("DB lookup error: %v", existErr)
	}
	if existing != nil {
		printInfoResult(existing, "local")
		return nil
	}

	return buildMediaFromTMDb(client, database, &selected)
}

// buildTMDbClient creates a TMDb client from config or env.
func buildTMDbClient(database *db.DB) (*tmdb.Client, error) {
	apiKey, cfgErr := database.GetConfig("TmdbApiKey")
	if cfgErr != nil && cfgErr.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error: %v", cfgErr)
	}
	if apiKey == "" {
		apiKey = os.Getenv("TMDB_API_KEY")
	}
	if apiKey == "" {
		return nil, apperror.New("No TMDb API key configured. Set it with: movie config set tmdb_api_key YOUR_KEY")
	}
	return tmdb.NewClient(apiKey), nil
}

// buildMediaFromTMDb creates a Media from a TMDb search result with full details.
func buildMediaFromTMDb(client *tmdb.Client, database *db.DB, selected *tmdb.SearchResult) *db.Media {
	title := selected.GetDisplayTitle()
	year := selected.GetYear()
	yearInt := 0
	if year != "" {
		yearInt, _ = strconv.Atoi(year)
	}

	fmt.Printf("⏳ Fetching details for: %s (%s)...\n", title, year)

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

	if selected.MediaType == string(db.MediaTypeMovie) || selected.MediaType == "" {
		m.Type = string(db.MediaTypeMovie)
		fetchMovieDetails(client, selected.ID, m)
	} else if selected.MediaType == string(db.MediaTypeTV) {
		m.Type = string(db.MediaTypeTV)
		fetchTVDetails(client, selected.ID, m)
	}

	downloadInfoThumbnail(client, database, selected, m)
	return m
}

// downloadInfoThumbnail downloads poster for info command context.
func downloadInfoThumbnail(client *tmdb.Client, database *db.DB, selected *tmdb.SearchResult, m *db.Media) {
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
}

// infoSaveAndDisplay persists a media record and displays it.
func infoSaveAndDisplay(database *db.DB, m *db.Media) {
	mediaID, insertErr := database.InsertMedia(m)
	if insertErr != nil {
		if m.TmdbID > 0 {
			insertErr = database.UpdateMediaByTmdbID(m)
			if insertErr == nil && m.Genre != "" {
				existing, _ := database.GetMediaByTmdbID(m.TmdbID)
				if existing != nil {
					database.ReplaceMediaGenres(existing.ID, m.Genre)
				}
			}
		}
		if insertErr != nil {
			errlog.Error("DB error: %v", insertErr)
			return
		}
	} else if mediaID > 0 && m.Genre != "" {
		database.LinkMediaGenres(mediaID, m.Genre)
	}

	printInfoResult(m, "tmdb")
}
