// movie_scan_process.go — per-file processing and TMDb enrichment for movie scan
package cmd

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// ScanContext holds shared state for a scan session.
type ScanContext struct {
	Database     *db.DB
	Client       *tmdb.Client
	HasTMDb      bool
	OutputDir    string
	TotalFiles   int
	MovieCount   int
	TVCount      int
	Skipped      int
	ScannedItems []db.Media
	UseTable     bool
	BatchID      string
}

// processVideoFile handles a single video file: clean, check DB, fetch TMDb, insert, write JSON.
func processVideoFile(vf videoFile, ctx *ScanContext) bool {
	ctx.TotalFiles++
	result := cleaner.Clean(vf.Name)

	printScanFileHeader(ctx, result)

	if isAlreadyScanned(ctx, vf, result) {
		return true
	}

	m := buildScanMedia(vf, result)
	if m == nil {
		return false
	}

	if ctx.HasTMDb {
		enrichFromTMDb(ctx.Client, ctx.Database, m, result, ctx.OutputDir)
	}

	mediaID := insertScanMedia(ctx, m)
	trackScanAction(ctx, m, vf.FullPath, mediaID, nil)
	writeScanJSON(ctx, m)

	ctx.ScannedItems = append(ctx.ScannedItems, *m)
	if ctx.UseTable {
		printScanTableRow(buildMediaTableRow(ctx.TotalFiles, m, "new"))
	}
	incrementTypeCount(ctx, m.Type)

	if !ctx.UseTable {
		fmt.Println()
	}
	return true
}

func printScanFileHeader(ctx *ScanContext, result cleaner.Result) {
	if ctx.UseTable {
		return
	}
	typeIcon := db.TypeIcon(result.Type)
	fmt.Printf("\n  %d. %s %s", ctx.TotalFiles, typeIcon, result.CleanTitle)
	if result.Year > 0 {
		fmt.Printf(" (%d)", result.Year)
	}
	fmt.Printf(" [%s]\n", result.Type)
}

func buildScanMedia(vf videoFile, result cleaner.Result) *db.Media {
	fi, fiErr := os.Stat(vf.FullPath)
	if fiErr != nil {
		logStatError(vf.FullPath, fiErr)
		return nil
	}
	m := &db.Media{
		Title: result.CleanTitle, CleanTitle: result.CleanTitle,
		Year: result.Year, Type: result.Type,
		OriginalFileName: vf.Name, OriginalFilePath: vf.FullPath,
		CurrentFilePath: vf.FullPath, FileExtension: result.Extension,
	}
	if fi != nil {
		m.FileSizeMb = float64(fi.Size()) / (1024 * 1024)
	}
	return m
}

func insertScanMedia(ctx *ScanContext, m *db.Media) int64 {
	mediaID, insertErr := ctx.Database.InsertMedia(m)
	if insertErr != nil {
		handleInsertError(ctx, m, insertErr)
		return 0
	}
	if mediaID > 0 {
		linkScanMediaRelations(ctx, m, mediaID)
	}
	return mediaID
}

func linkScanMediaRelations(ctx *ScanContext, m *db.Media, mediaID int64) {
	if m.Genre != "" {
		if linkErr := ctx.Database.LinkMediaGenres(mediaID, m.Genre); linkErr != nil {
			errlog.Warn("Genre link error for '%s': %v", m.Title, linkErr)
		}
	}
	if m.Director != "" {
		if linkErr := ctx.Database.LinkMediaDirectors(mediaID, m.Director); linkErr != nil {
			errlog.Warn("Director link error for '%s': %v", m.Title, linkErr)
		}
	}
}

func writeScanJSON(ctx *ScanContext, m *db.Media) {
	if jsonErr := writeMediaJSON(ctx.OutputDir, m); jsonErr != nil {
		errlog.Warn("JSON write error for '%s': %v", m.Title, jsonErr)
	}
}

// enrichFromTMDb fetches metadata, details, and thumbnail from TMDb.
func enrichFromTMDb(client *tmdb.Client, database *db.DB, m *db.Media, result cleaner.Result, outputDir string) {
	searchQuery := buildTMDbSearchQuery(result)

	tmdbResults, tmdbErr := client.SearchMulti(searchQuery)
	if tmdbErr != nil {
		logTMDbSearchError(searchQuery, tmdbErr)
		return
	}
	if len(tmdbResults) == 0 {
		errlog.Warn("no TMDb match for '%s' — inserted with local data only", searchQuery)
		return
	}

	applyTMDbResult(client, database, m, tmdbResults[0], outputDir)
}

func buildTMDbSearchQuery(result cleaner.Result) string {
	searchTitle := result.CleanTitle
	if result.Year > 0 {
		yearStr := strconv.Itoa(result.Year)
		re := regexp.MustCompile(`\s+` + regexp.QuoteMeta(yearStr) + `$`)
		searchTitle = re.ReplaceAllString(searchTitle, "")
	}
	if result.Year > 0 {
		return searchTitle + " " + strconv.Itoa(result.Year)
	}
	return searchTitle
}

func logTMDbSearchError(query string, tmdbErr error) {
	switch {
	case errors.Is(tmdbErr, tmdb.ErrAuthInvalid):
		errlog.Error("❌ TMDb API key is invalid. Run: movie config set tmdb_api_key YOUR_KEY")
	case errors.Is(tmdbErr, tmdb.ErrRateLimited):
		errlog.Warn("TMDb rate limit exceeded — try again in a few seconds")
	case errors.Is(tmdbErr, tmdb.ErrServerError):
		errlog.Warn("⚠️ TMDb is temporarily unavailable. Try again later.")
	case errors.Is(tmdbErr, tmdb.ErrTimeout):
		errlog.Warn("⚠️ TMDb request timed out. Check your internet connection.")
	case errors.Is(tmdbErr, tmdb.ErrNetworkError):
		errlog.Warn("⚠️ Network unavailable — scanning with local data only for '%s'", query)
	default:
		errlog.Warn("TMDb search failed for '%s': %v", query, tmdbErr)
	}
}

func applyTMDbResult(client *tmdb.Client, database *db.DB, m *db.Media, best tmdb.SearchResult, outputDir string) {
	m.TmdbID = best.ID
	m.TmdbRating = best.VoteAvg
	m.Popularity = best.Popularity
	m.Description = best.Overview
	m.Genre = tmdb.GenreNames(best.GenreIDs)

	if best.MediaType == string(db.MediaTypeTV) {
		m.Type = string(db.MediaTypeTV)
		fetchTVDetails(client, best.ID, m)
	} else {
		m.Type = string(db.MediaTypeMovie)
		fetchMovieDetails(client, best.ID, m)
	}

	downloadThumbnail(client, database, m, best.PosterPath, outputDir)
	fmt.Printf("     ⭐ %.1f  %s\n", m.TmdbRating, m.Title)
}
