// movie_search.go — movie search <name>
// Searches TMDb API, fetches full details, and saves to local database.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

var searchFormat string

var movieSearchCmd = &cobra.Command{
	Use:   "search [name]",
	Short: "Search TMDb for a movie or TV show and save to database",
	Long: `Searches the TMDb API for movies/TV shows matching the query.
Fetches full metadata (rating, genres, cast, crew, poster) and saves
to the local database. Categorizes as Movie or TV Show automatically.
Does NOT require the file to exist in your library.

Use --format json to output search results as JSON (no interactive prompt).
Use --format table to output search results as a formatted table (no interactive prompt).`,
	Args: cobra.MinimumNArgs(1),
	Run:  runMovieSearch,
}

func init() {
	movieSearchCmd.Flags().StringVar(&searchFormat, "format", "", "Output format: json, table")
}

func runMovieSearch(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	client := resolveSearchTMDbClient(database)
	if client == nil {
		return
	}

	query := strings.Join(args, " ")
	results := executeSearch(client, query)
	if results == nil {
		return
	}

	if searchFormat == string(db.OutputFormatJSON) {
		printSearchResultsJSON(results)
		return
	}
	if searchFormat == string(db.OutputFormatTable) {
		printSearchResultsTable(results)
		return
	}

	selected := promptSearchSelection(results)
	if selected == nil {
		return
	}
	saveSearchResult(client, database, *selected)
}

func resolveSearchTMDbClient(database *db.DB) *tmdb.Client {
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

func executeSearch(client *tmdb.Client, query string) []tmdb.SearchResult {
	if searchFormat != string(db.OutputFormatJSON) && searchFormat != string(db.OutputFormatTable) {
		fmt.Printf("🔎 Searching TMDb for: %s\n\n", query)
	}

	results, searchErr := client.SearchMulti(query)
	if searchErr != nil {
		errlog.Error("TMDb search error: %v", searchErr)
		return nil
	}
	if len(results) == 0 {
		if searchFormat == string(db.OutputFormatJSON) {
			fmt.Println("[]")
		} else {
			fmt.Println("📭 No results found on TMDb.")
		}
		return nil
	}
	return results
}

func promptSearchSelection(results []tmdb.SearchResult) *tmdb.SearchResult {
	fmt.Printf("Found %d results:\n\n", len(results))
	printSearchResultsList(results)

	fmt.Println()
	fmt.Print("Enter number to save (0 to cancel): ")

	var choice int
	_, scanErr := fmt.Scan(&choice)
	if scanErr != nil || choice < 1 || choice > len(results) || choice > 15 {
		fmt.Println("❌ Canceled.")
		return nil
	}
	return &results[choice-1]
}

func printSearchResultsList(results []tmdb.SearchResult) {
	for i := range results {
		if i >= 15 {
			break
		}
		printSearchResultRow(i+1, &results[i])
	}
}

func printSearchResultRow(num int, r *tmdb.SearchResult) {
	title := r.GetDisplayTitle()
	year := r.GetYear()
	typeIcon := db.TypeIcon(r.MediaType)
	typeLabel := db.TypeLabel(r.MediaType)

	rating := "N/A"
	if r.VoteAvg > 0 {
		rating = fmt.Sprintf("%.1f", r.VoteAvg)
	}
	yearStr := ""
	if year != "" {
		yearStr = fmt.Sprintf("(%s)", year)
	}
	fmt.Printf("  %d. %s %-35s %-6s  ⭐ %-4s  [%s]\n",
		num, typeIcon, title, yearStr, rating, typeLabel)
}
