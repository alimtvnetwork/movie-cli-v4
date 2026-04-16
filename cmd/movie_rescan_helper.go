// movie_rescan_helper.go — shared rescan logic used by both scan and rescan commands
package cmd

import (
	"regexp"
	"strconv"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// mediaNeedsRescan returns true if the entry is missing genre, rating, or description.
// Genre is populated from the M:N Genre/MediaGenre tables via the compat field.
func mediaNeedsRescan(m *db.Media) bool {
	return m.Genre == "" || m.TmdbRating == 0 || m.Description == ""
}

// rescanMediaEntry re-fetches TMDb metadata for a single media entry.
// Returns true if the entry was updated successfully.
func rescanMediaEntry(database *db.DB, client *tmdb.Client, m *db.Media) bool {
	searchTitle := m.CleanTitle
	if m.Year > 0 {
		yearStr := strconv.Itoa(m.Year)
		re := regexp.MustCompile(`\s+` + regexp.QuoteMeta(yearStr) + `$`)
		searchTitle = re.ReplaceAllString(searchTitle, "")
	}
	searchQuery := searchTitle
	if m.Year > 0 {
		searchQuery += " " + strconv.Itoa(m.Year)
	}

	tmdbResults, tmdbErr := client.SearchMulti(searchQuery)
	if tmdbErr != nil {
		errlog.Warn("rescan TMDb search failed for '%s': %v", searchQuery, tmdbErr)
		return false
	}
	if len(tmdbResults) == 0 {
		return false
	}

	best := tmdbResults[0]
	m.TmdbID = best.ID
	m.TmdbRating = best.VoteAvg
	m.Popularity = best.Popularity
	m.Description = best.Overview
	m.Genre = tmdb.GenreNames(best.GenreIDs)

	m.Type = resolveMediaType(best.MediaType)
	fetchDetailsByType(client, best.ID, m)

	if !updateRescanEntry(database, m) {
		return false
	}

	linkRescanRelations(database, m)
	return true
}

// updateRescanEntry persists updated media to DB, trying TmdbID first.
func updateRescanEntry(database *db.DB, m *db.Media) bool {
	if m.TmdbID <= 0 {
		return updateByID(database, m)
	}
	if err := database.UpdateMediaByTmdbID(m); err == nil {
		return true
	}
	return updateByID(database, m)
}

func updateByID(database *db.DB, m *db.Media) bool {
	if err := database.UpdateMediaByID(m); err != nil {
		errlog.Error("rescan DB update failed for '%s': %v", m.Title, err)
		return false
	}
	return true
}

func linkRescanRelations(database *db.DB, m *db.Media) {
	if m.ID <= 0 {
		return
	}
	if m.Genre != "" {
		database.ReplaceMediaGenres(m.ID, m.Genre)
	}
	if m.Director != "" {
		database.ReplaceMediaDirectors(m.ID, m.Director)
	}
}
