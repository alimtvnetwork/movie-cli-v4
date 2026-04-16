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
	count := 10
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			count = n
		}
	}

	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	apiKey, err := database.GetConfig("TmdbApiKey")
	if err != nil && err.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error: %v", err)
	}
	if apiKey == "" {
		apiKey = os.Getenv("TMDB_API_KEY")
	}
	if apiKey == "" {
		errlog.Error("TMDb API key required for suggestions. Set with: movie config set tmdb_api_key YOUR_KEY")
		return
	}

	client := tmdb.NewClient(apiKey)

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
		return
	}

	choice := strings.TrimSpace(scanner.Text())
	fmt.Println()

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

func suggestByType(database *db.DB, client *tmdb.Client, mediaType string, count int) {
	typeName := db.TypeLabelPlural(mediaType)

	fmt.Printf("🔍 Analyzing your %s library...\n\n", typeName)

	// Get top genres from library
	genres, err := database.TopGenres(5)
	if err != nil {
		errlog.Warn("Genre analysis error: %v", err)
		fmt.Println("⚠️  Showing trending instead.")
		showTrending(client, mediaType, count)
		return
	}
	if len(genres) == 0 {
		fmt.Println("⚠️  Not enough data. Showing trending instead.")
		showTrending(client, mediaType, count)
		return
	}

	// Sort genres by frequency
	type genreCount struct {
		name  string
		count int
	}
	var sorted []genreCount
	for name, cnt := range genres {
		sorted = append(sorted, genreCount{name, cnt})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

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

	// Get existing media to avoid duplicates
	existing, existErr := database.MediaByType(mediaType, 1000)
	if existErr != nil {
		errlog.Warn("DB error: %v", existErr)
	}
	existingIDs := make(map[int]bool)
	for i := range existing {
		existingIDs[existing[i].TmdbID] = true
	}

	var suggestions []tmdb.SearchResult

	// Phase 1: DiscoverByGenre — use top genres for genre-based discovery
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
		discovered, discErr := client.DiscoverByGenre(mediaType, genreID, 1)
		if discErr != nil {
			errlog.Warn("Discover error: %v", discErr)
			continue
		}
		for i := range discovered {
			if !existingIDs[discovered[i].ID] && len(suggestions) < count {
				suggestions = append(suggestions, discovered[i])
				existingIDs[discovered[i].ID] = true
			}
		}
	}

	// Phase 2: Recommendations from random library items (fill gaps)
	if len(suggestions) < count && len(existing) > 0 {
		indices := rand.Perm(len(existing))
		for _, idx := range indices {
			if len(suggestions) >= count {
				break
			}
			recs, recErr := client.GetRecommendations(existing[idx].TmdbID, mediaType, 1)
			if recErr != nil {
				errlog.Warn("Recommendations error for TMDb ID %d: %v", existing[idx].TmdbID, recErr)
				continue
			}
			for i := range recs {
				if !existingIDs[recs[i].ID] && len(suggestions) < count {
					suggestions = append(suggestions, recs[i])
					existingIDs[recs[i].ID] = true
				}
			}
		}
	}

	// Phase 3: Fill remaining with trending
	if len(suggestions) < count {
		trending, trendErr := client.Trending(mediaType)
		if trendErr != nil {
			errlog.Warn("Trending fetch error: %v", trendErr)
		}
		for i := range trending {
			if !existingIDs[trending[i].ID] && len(suggestions) < count {
				suggestions = append(suggestions, trending[i])
				existingIDs[trending[i].ID] = true
			}
		}
	}

	fmt.Println()
	printSuggestions(suggestions, typeName)
}

func suggestRandom(client *tmdb.Client, count int) {
	fmt.Println("🎲 Fetching random suggestions...")

	var suggestions []tmdb.SearchResult
	seenIDs := make(map[int]bool)

	// Mix movie and TV trending
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

	for i := range all {
		if len(suggestions) >= count {
			break
		}
		if !seenIDs[all[i].ID] {
			suggestions = append(suggestions, all[i])
			seenIDs[all[i].ID] = true
		}
	}

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

	typeName := db.TypeLabelPlural(mediaType)
	printSuggestions(trending, typeName)
}

func printSuggestions(suggestions []tmdb.SearchResult, category string) {
	if len(suggestions) == 0 {
		fmt.Println("📭 No suggestions available.")
		return
	}

	fmt.Printf("✨ Suggested %s (%d):\n\n", category, len(suggestions))

	for i := range suggestions {
		title := suggestions[i].GetDisplayTitle()
		year := suggestions[i].GetYear()
		rating := fmt.Sprintf("%.1f", suggestions[i].VoteAvg)
		genres := tmdb.GenreNames(suggestions[i].GenreIDs)

		fmt.Printf("  %2d. %s", i+1, title)
		if year != "" {
			fmt.Printf(" (%s)", year)
		}
		fmt.Printf("  ⭐ %s", rating)
		if genres != "" {
			fmt.Printf("  [%s]", genres)
		}
		fmt.Println()
	}
}
