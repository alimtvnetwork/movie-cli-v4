// movie_suggest.go — movie suggest [N]
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

var movieSuggestCmd = &cobra.Command{
	Use:   "suggest [N]",
	Short: "Get movie or TV show suggestions",
	Long: `Suggests movies or TV shows based on your library.
Choose Movie, TV, or Random (Empty).`,
	Args: cobra.MaximumNArgs(1),
	Run:  runMovieSuggest,
}

func runMovieSuggest(cmd *cobra.Command, args []string) {
	count := parseSuggestCount(args)

	database, client := initSuggestDeps()
	if database == nil {
		return
	}
	defer database.Close()

	choice := promptSuggestCategory()

	switch choice {
	case "1":
		suggestByType(database, client, string(db.MediaTypeMovie), count)
	case "2":
		suggestByType(database, client, string(db.MediaTypeTV), count)
	case "3":
		suggestRandom(client, count)
	default:
		fmt.Println("❌ Invalid choice")
	}
}

func parseSuggestCount(args []string) int {
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			return n
		}
	}
	return 10
}

func initSuggestDeps() (*db.DB, *tmdb.Client) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return nil, nil
	}

	apiKey, err := database.GetConfig("TmdbApiKey")
	if err != nil && err.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error: %v", err)
	}
	if apiKey == "" {
		apiKey = os.Getenv("TMDB_API_KEY")
	}
	if apiKey == "" {
		errlog.Error("TMDb API key required for suggestions. Set with: movie config set tmdb_api_key YOUR_KEY")
		database.Close()
		return nil, nil
	}
	return database, tmdb.NewClient(apiKey)
}

func promptSuggestCategory() string {
	fmt.Println("🎯 Movie Suggest")
	fmt.Println()
	fmt.Println("  Select category:")
	fmt.Println("  1. 🎬 Movie")
	fmt.Println("  2. 📺 TV")
	fmt.Println("  3. 🎲 Empty (Random)")
	fmt.Println()
	fmt.Print("  Choose [1/2/3]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return ""
	}
	choice := strings.TrimSpace(scanner.Text())
	fmt.Println()
	return choice
}

func suggestByType(database *db.DB, client *tmdb.Client, mediaType string, count int) {
	typeName := db.TypeLabelPlural(mediaType)
	fmt.Printf("🔍 Analyzing your %s library...\n\n", typeName)

	sorted := analyzeTopGenres(database)
	if sorted == nil {
		showTrending(client, mediaType, count)
		return
	}

	printTopGenres(sorted)

	existingIDs := collectExistingIDs(database, mediaType)
	var suggestions []tmdb.SearchResult

	suggestions = discoverByGenres(client, sorted, mediaType, typeName, existingIDs, count)
	suggestions = fillFromRecommendations(client, database, mediaType, existingIDs, suggestions, count)
	suggestions = fillFromTrending(client, mediaType, existingIDs, suggestions, count)

	fmt.Println()
	printSuggestions(suggestions, typeName)
}

func suggestRandom(client *tmdb.Client, count int) {
	fmt.Println("🎲 Fetching random suggestions...")

	movieTrending, err := client.Trending(string(db.MediaTypeMovie))
	if err != nil {
		errlog.Warn("Movie trending error: %v", err)
	}
	tvTrending, err := client.Trending(string(db.MediaTypeTV))
	if err != nil {
		errlog.Warn("TV trending error: %v", err)
	}

	all := make([]tmdb.SearchResult, 0, len(movieTrending)+len(tvTrending))
	all = append(all, movieTrending...)
	all = append(all, tvTrending...)

	seenIDs := make(map[int]bool)
	suggestions := appendUnique(all, nil, seenIDs, count)
	printSuggestions(suggestions, "Movies & TV Shows")
}

func showTrending(client *tmdb.Client, mediaType string, count int) {
	trending, err := client.Trending(mediaType)
	if err != nil {
		errlog.Error("TMDb error: %v", err)
		return
	}
	if len(trending) > count {
		trending = trending[:count]
	}
	printSuggestions(trending, db.TypeLabelPlural(mediaType))
}
