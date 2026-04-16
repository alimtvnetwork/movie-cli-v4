// movie_suggest_helpers.go — helper functions for movie suggest command.
package cmd

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// genreCount holds a genre name and its frequency.
type genreCount struct {
	name  string
	count int
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
		results, discErr := client.DiscoverByGenre(mediaType, genreID, 1)
		suggestions = appendUniqueResults(results, discErr, suggestions, existingIDs, count)
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
