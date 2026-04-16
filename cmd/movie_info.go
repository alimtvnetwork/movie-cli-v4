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
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
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

	m, resolveErr := resolveMediaByQuery(database, query)
	if resolveErr == nil {
		displayMediaInfo(m, "local")
		return
	}

	m = fetchAndStoreFromTMDb(database, query)
	if m != nil {
		displayMediaInfo(m, "tmdb")
	}
}

func displayMediaInfo(m *db.Media, source string) {
	switch infoFormat {
	case string(db.OutputFormatJSON):
		printMediaDetailJSON(m, source)
	case string(db.OutputFormatTable):
		printMediaDetailTable(m)
	default:
		label := "📚 Found in local library:"
		if source == "tmdb" {
			label = "✅ Saved to your library!"
		}
		fmt.Println(label)
		fmt.Println()
		printMediaDetail(m)
	}
}

func fetchAndStoreFromTMDb(database *db.DB, query string) *db.Media {
	fmt.Printf("🔎 Not found locally. Searching TMDb for: %s\n\n", query)

	client := resolveInfoTMDbClient(database)
	if client == nil {
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

	existing := checkExistingByTmdbID(database, selected.ID)
	if existing != nil {
		return existing
	}

	m := buildInfoMedia(selected)
	enrichInfoMedia(client, m, selected)
	downloadInfoThumbnail(client, database, m, selected.PosterPath)
	saveInfoMedia(database, m)

	return m
}

func resolveInfoTMDbClient(database *db.DB) *tmdb.Client {
	apiKey, cfgErr := database.GetConfig("TmdbApiKey")
	if cfgErr != nil && cfgErr.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error: %v", cfgErr)
	}
	if apiKey == "" {
		apiKey = os.Getenv("TMDB_API_KEY")
	}
	if apiKey == "" {
		errlog.Error("No TMDb API key configured. Set it with: movie config set tmdb_api_key YOUR_KEY")
		return nil
	}
	return tmdb.NewClient(apiKey)
}

func checkExistingByTmdbID(database *db.DB, tmdbID int) *db.Media {
	existing, existErr := database.GetMediaByTmdbID(tmdbID)
	if existErr != nil && existErr.Error() != "sql: no rows in result set" {
		errlog.Warn("DB lookup error: %v", existErr)
	}
	return existing
}

func buildInfoMedia(selected tmdb.SearchResult) *db.Media {
	yearInt := 0
	if year := selected.GetYear(); year != "" {
		yearInt, _ = strconv.Atoi(year)
	}
	return &db.Media{
		Title:       selected.GetDisplayTitle(),
		CleanTitle:  selected.GetDisplayTitle(),
		Year:        yearInt,
		TmdbID:      selected.ID,
		TmdbRating:  selected.VoteAvg,
		Popularity:  selected.Popularity,
		Description: selected.Overview,
		Genre:       tmdb.GenreNames(selected.GenreIDs),
	}
}

func enrichInfoMedia(client *tmdb.Client, m *db.Media, selected tmdb.SearchResult) {
	if selected.MediaType == string(db.MediaTypeTV) {
		m.Type = string(db.MediaTypeTV)
		fetchTVDetails(client, selected.ID, m)
		return
	}
	m.Type = string(db.MediaTypeMovie)
	fetchMovieDetails(client, selected.ID, m)
}

func downloadInfoThumbnail(input ThumbnailInput) {
	if input.PosterPath == "" {
		return
	}
	downloadThumbnailForMedia(input.Client, input.Database, input.Media, input.PosterPath)
}

func saveInfoMedia(database *db.DB, m *db.Media) {
	mediaID, insertErr := database.InsertMedia(m)
	if insertErr != nil {
		handleInfoInsertError(database, m, insertErr)
		return
	}
	if mediaID > 0 && m.Genre != "" {
		database.LinkMediaGenres(mediaID, m.Genre)
	}
}

func handleInfoInsertError(database *db.DB, m *db.Media, insertErr error) {
	if m.TmdbID <= 0 {
		errlog.Error("DB error: %v", insertErr)
		return
	}
	updateErr := database.UpdateMediaByTmdbID(m)
	if updateErr != nil {
		errlog.Error("DB error: %v", updateErr)
		return
	}
	if m.Genre == "" {
		return
	}
	existing, _ := database.GetMediaByTmdbID(m.TmdbID)
	if existing != nil {
		database.ReplaceMediaGenres(existing.ID, m.Genre)
	}
}
