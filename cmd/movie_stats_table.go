// movie_stats_table.go — table-formatted output for movie stats
package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v3/db"
)

// printStatsTable outputs library statistics as a formatted table.
func printStatsTable(database *db.DB, totalMovies, totalTV, total int) {
	labelWidth := 20
	valueWidth := 40

	fmt.Println()
	fmt.Printf("  %-*s │ %-*s\n", labelWidth, "Metric", valueWidth, "Value")
	fmt.Printf("  %s─┼─%s\n",
		strings.Repeat("─", labelWidth),
		strings.Repeat("─", valueWidth))

	fmt.Printf("  %-*s │ %-*d\n", labelWidth, "Total Movies", valueWidth, totalMovies)
	fmt.Printf("  %-*s │ %-*d\n", labelWidth, "Total TV Shows", valueWidth, totalTV)
	fmt.Printf("  %-*s │ %-*d\n", labelWidth, "Total", valueWidth, total)

	// Storage
	totalSize, largestSize, smallestSize, sizeErr := database.FileSizeStats()
	if sizeErr == nil && totalSize > 0 {
		fmt.Printf("  %-*s │ %-*s\n", labelWidth, "Total Size", valueWidth, db.HumanSize(totalSize))
		largestTitle, _, lgErr := database.LargestMediaBySize()
		if lgErr == nil && largestTitle != "" {
			fmt.Printf("  %-*s │ %-*s\n", labelWidth, "Largest File", valueWidth,
				fmt.Sprintf("%s (%s)", db.HumanSize(largestSize), largestTitle))
		} else {
			fmt.Printf("  %-*s │ %-*s\n", labelWidth, "Largest File", valueWidth, db.HumanSize(largestSize))
		}
		fmt.Printf("  %-*s │ %-*s\n", labelWidth, "Smallest File", valueWidth, db.HumanSize(smallestSize))
		if total > 0 {
			fmt.Printf("  %-*s │ %-*s\n", labelWidth, "Average Size", valueWidth, db.HumanSize(totalSize/float64(total)))
		}
	}

	// Top genres
	genres, genreErr := database.TopGenres(10)
	if genreErr == nil && len(genres) > 0 {
		type gc struct {
			name  string
			count int
		}
		var sorted []gc
		for n, c := range genres {
			sorted = append(sorted, gc{n, c})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})

		fmt.Printf("  %s─┼─%s\n",
			strings.Repeat("─", labelWidth),
			strings.Repeat("─", valueWidth))

		for i, g := range sorted {
			if i >= 10 {
				break
			}
			fmt.Printf("  %-*s │ %-*d\n", labelWidth, g.name, valueWidth, g.count)
		}
	}

	// Average ratings
	allMedia, listErr := database.ListMedia(0, 100000)
	if listErr == nil {
		var sumImdb, sumTmdb float64
		var cntImdb, cntTmdb int
		for i := range allMedia {
			if allMedia[i].ImdbRating > 0 {
				sumImdb += allMedia[i].ImdbRating
				cntImdb++
			}
			if allMedia[i].TmdbRating > 0 {
				sumTmdb += allMedia[i].TmdbRating
				cntTmdb++
			}
		}

		fmt.Printf("  %s─┼─%s\n",
			strings.Repeat("─", labelWidth),
			strings.Repeat("─", valueWidth))

		if cntImdb > 0 {
			fmt.Printf("  %-*s │ %-*.1f\n", labelWidth, "Avg IMDb Rating", valueWidth, sumImdb/float64(cntImdb))
		}
		if cntTmdb > 0 {
			fmt.Printf("  %-*s │ %-*.1f\n", labelWidth, "Avg TMDb Rating", valueWidth, sumTmdb/float64(cntTmdb))
		}
	}

	fmt.Printf("  %s─┴─%s\n",
		strings.Repeat("─", labelWidth),
		strings.Repeat("─", valueWidth))
	fmt.Println()
}
