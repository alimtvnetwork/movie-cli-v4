// movie_suggest.go — movie suggest [N]
package cmd

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
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

// genreCount holds a genre name and its frequency.
type genreCount struct {
	name  string
	count int
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

func analyzeTopGenres(database *db.DB) []genreCount {
	genres, err := database.TopGenres(5)
	if err != nil {
		errlog.Warn("Genre analysis error: %v", err)
		fmt.Println("⚠️  Showing trending instead.")
		return nil
	}
	if len(genres) == 0 {
		fmt.Println("⚠️  Not enough data. Showing trending instead.")
		return nil
	}

	var sorted []genreCount
	for name, cnt := range genres {
		sorted = append(sorted, genreCount{name, cnt})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})
	return sorted
}

func printTopGenres(sorted []genreCount) {
	fmt.Printf("📊 Your top genres: ")
	for i, g := range sorted {
		if i >= 3 {
			break
		}
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%s (%d)", g.name, g.count)
	}
	fmt.Println()
	fmt.Println()
}

func collectExistingIDs(database *db.DB, mediaType string) map[int]bool {
	existing, existErr := database.MediaByType(mediaType, 1000)
	if existErr != nil {
		errlog.Warn("DB error: %v", existErr)
	}
	ids := make(map[int]bool)
	for i := range existing {
		ids[existing[i].TmdbID] = true
	}
	return ids
}

func discoverByGenres(client *tmdb.Client, sorted []genreCount, mediaType, typeName string, existingIDs map[int]bool, count int) []tmdb.SearchResult {
	var suggestions []tmdb.SearchResult
	genreNameToID := tmdb.GenreNameToID()

	for _, g := range sorted {
		if len(suggestions) >= count {
			break
		}
		genreID, ok := genreNameToID[g.name]
		if !ok {
			continue
		}
		fmt.Printf("  🎭 Discovering %s %s...\n", g.name, typeName)
		suggestions = appendUniqueResults(client.DiscoverByGenre(mediaType, genreID, 1), suggestions, existingIDs, count)
	}
	return suggestions
}

func fillFromRecommendations(client *tmdb.Client, database *db.DB, mediaType string, existingIDs map[int]bool, suggestions []tmdb.SearchResult, count int) []tmdb.SearchResult {
	if len(suggestions) >= count {
		return suggestions
	}
	existing, _ := database.MediaByType(mediaType, 1000)
	if len(existing) == 0 {
		return suggestions
	}
	indices := rand.Perm(len(existing))
	for _, idx := range indices {
		if len(suggestions) >= count {
			break
		}
		recs, recErr := client.GetRecommendations(existing[idx].TmdbID, mediaType, 1)
		if recErr != nil {
			continue
		}
		suggestions = appendUnique(recs, suggestions, existingIDs, count)
	}
	return suggestions
}

func fillFromTrending(client *tmdb.Client, mediaType string, existingIDs map[int]bool, suggestions []tmdb.SearchResult, count int) []tmdb.SearchResult {
	if len(suggestions) >= count {
		return suggestions
	}
	trending, trendErr := client.Trending(mediaType)
	if trendErr != nil {
		errlog.Warn("Trending fetch error: %v", trendErr)
		return suggestions
	}
	return appendUnique(trending, suggestions, existingIDs, count)
}

func appendUniqueResults(results []tmdb.SearchResult, discErr error, suggestions []tmdb.SearchResult, existingIDs map[int]bool, count int) []tmdb.SearchResult {
	if discErr != nil {
		errlog.Warn("Discover error: %v", discErr)
		return suggestions
	}
	return appendUnique(results, suggestions, existingIDs, count)
}

func appendUnique(results, suggestions []tmdb.SearchResult, existingIDs map[int]bool, count int) []tmdb.SearchResult {
	for i := range results {
		if len(suggestions) >= count {
			break
		}
		if !existingIDs[results[i].ID] {
			suggestions = append(suggestions, results[i])
			existingIDs[results[i].ID] = true
		}
	}
	return suggestions
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
	rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

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

func printSuggestions(suggestions []tmdb.SearchResult, category string) {
	if len(suggestions) == 0 {
		fmt.Println("📭 No suggestions available.")
		return
	}
	fmt.Printf("✨ Suggested %s (%d):\n\n", category, len(suggestions))
	for i := range suggestions {
		printSuggestionItem(i, &suggestions[i])
	}
}

func printSuggestionItem(idx int, s *tmdb.SearchResult) {
	title := s.GetDisplayTitle()
	year := s.GetYear()
	rating := fmt.Sprintf("%.1f", s.VoteAvg)
	genres := tmdb.GenreNames(s.GenreIDs)

	fmt.Printf("  %2d. %s", idx+1, title)
	if year != "" {
		fmt.Printf(" (%s)", year)
	}
	fmt.Printf("  ⭐ %s", rating)
	if genres != "" {
		fmt.Printf("  [%s]", genres)
	}
	fmt.Println()
}
