// movie_stats.go — movie stats
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var statsFormat string

var movieStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show library statistics",
	Long: `Display total counts, top genres, and average ratings.

Use --format json to output stats as JSON to stdout for piping.
Use --format table to output stats as a formatted table.`,
	Run: runMovieStats,
}

func init() {
	movieStatsCmd.Flags().StringVar(&statsFormat, "format", "default",
		"output format: default, table, or json")
}

// statsJSONOutput is the JSON structure for --format json.
type statsJSONOutput struct {
	TotalMovies int           `json:"total_movies"`
	TotalTV     int           `json:"total_tv_shows"`
	Total       int           `json:"total"`
	Storage     *statsStorage `json:"storage,omitempty"`
	TopGenres   []statsGenre  `json:"top_genres,omitempty"`
	AvgImdb     float64       `json:"avg_imdb_rating,omitempty"`
	AvgTmdb     float64       `json:"avg_tmdb_rating,omitempty"`
}

type statsStorage struct {
	TotalSize    int64  `json:"total_bytes"`
	TotalHuman   string `json:"total_human"`
	LargestFile  int64  `json:"largest_file_bytes"`
	SmallestFile int64  `json:"smallest_file_bytes"`
	AverageSize  int64  `json:"average_bytes"`
}

type statsGenre struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func runMovieStats(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	totalMovies, _ := database.CountMedia(string(db.MediaTypeMovie))
	totalTV, _ := database.CountMedia(string(db.MediaTypeTV))
	total, err := database.CountMedia("")
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}

	if total == 0 {
		printEmptyStats()
		return
	}

	switch statsFormat {
	case string(db.OutputFormatJSON):
		printStatsJSON(database, totalMovies, totalTV, total)
	case string(db.OutputFormatTable):
		printStatsTable(database, totalMovies, totalTV, total)
	default:
		printStatsDefault(database, totalMovies, totalTV, total)
	}
}

func printEmptyStats() {
	if statsFormat == "json" {
		fmt.Println("{}")
	} else {
		fmt.Println("📭 No media in library. Run 'movie scan <folder>' first.")
	}
}

func printStatsJSON(database *db.DB, totalMovies, totalTV, total int) {
	out := statsJSONOutput{
		TotalMovies: totalMovies, TotalTV: totalTV, Total: total,
	}
	out.Storage = buildStatsStorageJSON(database, total)
	out.TopGenres = buildStatsGenresJSON(database)
	out.AvgImdb, out.AvgTmdb = computeAvgRatings(database)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(out); encErr != nil {
		errlog.Error("JSON encode error: %v", encErr)
	}
}

func buildStatsStorageJSON(database *db.DB, total int) *statsStorage {
	totalSize, largestSize, smallestSize, sizeErr := database.FileSizeStats()
	if sizeErr != nil || totalSize <= 0 {
		return nil
	}
	return &statsStorage{
		TotalSize:    int64(totalSize * 1024 * 1024),
		TotalHuman:   db.HumanSize(totalSize),
		LargestFile:  int64(largestSize * 1024 * 1024),
		SmallestFile: int64(smallestSize * 1024 * 1024),
		AverageSize:  int64(totalSize*1024*1024) / int64(total),
	}
}

func buildStatsGenresJSON(database *db.DB) []statsGenre {
	genres, genreErr := database.TopGenres(10)
	if genreErr != nil || len(genres) == 0 {
		return nil
	}
	var out []statsGenre
	for n, c := range genres {
		out = append(out, statsGenre{Name: n, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

func printStatsDefault(database *db.DB, totalMovies, totalTV, total int) {
	printStatsDefaultCounts(totalMovies, totalTV, total)
	printStatsDefaultStorage(database, total)
	printStatsDefaultGenres(database)
	printStatsDefaultRatings(database)
}

func printStatsDefaultCounts(totalMovies, totalTV, total int) {
	fmt.Println("📊 Library Statistics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  🎬 Total Movies:    %d\n", totalMovies)
	fmt.Printf("  📺 Total TV Shows:  %d\n", totalTV)
	fmt.Printf("  📁 Total:           %d\n", total)
	fmt.Println()
}

func printStatsDefaultStorage(database *db.DB, total int) {
	totalSize, largestSize, smallestSize, sizeErr := database.FileSizeStats()
	if sizeErr != nil {
		errlog.Warn("File size stats error: %v", sizeErr)
		return
	}
	if totalSize <= 0 {
		return
	}
	fmt.Println("  💾 Storage:")
	fmt.Printf("     Total Size:    %s\n", db.HumanSize(totalSize))
	fmt.Printf("     Largest File:  %s\n", db.HumanSize(largestSize))
	fmt.Printf("     Smallest File: %s\n", db.HumanSize(smallestSize))
	if total > 0 {
		fmt.Printf("     Average Size:  %s\n", db.HumanSize(totalSize/float64(total)))
	}
	fmt.Println()
}

func printStatsDefaultGenres(database *db.DB) {
	sorted := sortedGenreCounts(database, 10)
	if len(sorted) == 0 {
		return
	}
	fmt.Println("  🎭 Top Genres:")
	for _, g := range sorted {
		bar := strings.Repeat("█", minInt(g.count, 30))
		fmt.Printf("     %-20s %s %d\n", g.name, bar, g.count)
	}
	fmt.Println()
}

func printStatsDefaultRatings(database *db.DB) {
	avgImdb, avgTmdb := computeAvgRatings(database)
	if avgImdb > 0 {
		fmt.Printf("  ⭐ Avg IMDb Rating: %.1f\n", avgImdb)
	}
	if avgTmdb > 0 {
		fmt.Printf("  ⭐ Avg TMDb Rating: %.1f\n", avgTmdb)
	}
}
