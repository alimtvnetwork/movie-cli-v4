// movie_discover.go — movie discover [genre]
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
	"github.com/alimtvnetwork/movie-cli-v3/tmdb"
)

var discoverPage int
var discoverType string

var movieDiscoverCmd = &cobra.Command{
	Use:   "discover [genre]",
	Short: "Browse TMDb movies or TV shows by genre",
	Long: `Discover movies or TV shows filtered by genre from TMDb.

If no genre is given, an interactive genre picker is shown.
Use --type to filter by movie or tv (default: movie).
Use --page to paginate results.

Examples:
  movie discover Action
  movie discover Comedy --type tv
  movie discover --page 2`,
	Args: cobra.MaximumNArgs(1),
	Run:  runMovieDiscover,
}

func init() {
	movieDiscoverCmd.Flags().IntVar(&discoverPage, "page", 1, "result page number")
	movieDiscoverCmd.Flags().StringVar(&discoverType, "type", "movie", "media type: movie or tv")
}

func runMovieDiscover(cmd *cobra.Command, args []string) {
	database, dbErr := db.Open()
	if dbErr != nil {
		errlog.Error("Database error: %v", dbErr)
		return
	}
	defer database.Close()

	client := initTMDbClient(database)
	if client == nil {
		return
	}

	genreName := ""
	if len(args) > 0 {
		genreName = args[0]
	} else {
		genreName = promptGenre()
		if genreName == "" {
			return
		}
	}

	genreID := resolveGenreID(genreName)
	if genreID == 0 {
		errlog.Error("Unknown genre: %s", genreName)
		printAvailableGenres()
		return
	}

	mediaType := resolveDiscoverType()
	typeLabel := db.TypeLabelPlural(mediaType)

	fmt.Printf("\n🎭 Discovering %s %s (page %d)...\n\n", genreName, typeLabel, discoverPage)

	results, discErr := client.DiscoverByGenre(mediaType, genreID, discoverPage)
	if discErr != nil {
		errlog.Error("TMDb discover error: %v", discErr)
		return
	}

	if len(results) == 0 {
		fmt.Printf("📭 No %s found for genre: %s\n", typeLabel, genreName)
		return
	}

	printDiscoverResults(results, genreName, typeLabel)
}

// initTMDbClient creates a TMDb client from config or env.
func initTMDbClient(database *db.DB) *tmdb.Client {
	apiKey, cfgErr := database.GetConfig("TmdbApiKey")
	if cfgErr != nil && cfgErr.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error: %v", cfgErr)
	}
	if apiKey == "" {
		apiKey = os.Getenv("TMDB_API_KEY")
	}
	if apiKey == "" {
		errlog.Error("TMDb API key required. Set with: movie config set tmdb_api_key YOUR_KEY")
		return nil
	}
	return tmdb.NewClient(apiKey)
}

// promptGenre shows an interactive genre picker.
func promptGenre() string {
	genres := sortedGenreNames()

	fmt.Println("🎭 Select a genre:")
	fmt.Println()
	for i, name := range genres {
		fmt.Printf("  %2d. %s\n", i+1, name)
	}
	fmt.Println()
	fmt.Print("  Choose [number or name]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return ""
	}
	input := strings.TrimSpace(scanner.Text())

	// Try as number
	if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(genres) {
		return genres[idx-1]
	}

	// Try as name (case-insensitive match)
	for _, name := range genres {
		if strings.EqualFold(name, input) {
			return name
		}
	}

	return input
}

// sortedGenreNames returns genre names sorted alphabetically.
func sortedGenreNames() []string {
	nameToID := tmdb.GenreNameToID()
	names := make([]string, 0, len(nameToID))
	for name := range nameToID {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resolveGenreID finds the TMDb genre ID for a name (case-insensitive).
func resolveGenreID(name string) int {
	nameToID := tmdb.GenreNameToID()
	for gName, gID := range nameToID {
		if strings.EqualFold(gName, name) {
			return gID
		}
	}
	return 0
}

// resolveDiscoverType normalizes the --type flag.
func resolveDiscoverType() string {
	if strings.EqualFold(discoverType, "tv") {
		return string(db.MediaTypeTV)
	}
	return string(db.MediaTypeMovie)
}

// printAvailableGenres lists all known genres.
func printAvailableGenres() {
	fmt.Println("\nAvailable genres:")
	for i, name := range sortedGenreNames() {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(name)
	}
	fmt.Println()
}

// printDiscoverResults displays discover results in a formatted list.
func printDiscoverResults(results []tmdb.SearchResult, genre, typeLabel string) {
	fmt.Printf("✨ %s %s (%d results):\n\n", genre, typeLabel, len(results))

	for i := range results {
		title := results[i].GetDisplayTitle()
		year := results[i].GetYear()
		rating := fmt.Sprintf("%.1f", results[i].VoteAvg)
		genres := tmdb.GenreNames(results[i].GenreIDs)

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

	if discoverPage > 0 {
		fmt.Printf("\n  📄 Page %d — use --page %d for more\n", discoverPage, discoverPage+1)
	}
}
