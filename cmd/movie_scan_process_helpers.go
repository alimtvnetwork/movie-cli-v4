// movie_scan_process_helpers.go — extracted helpers for processVideoFile.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alimtvnetwork/movie-cli-v3/cleaner"
	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
	"github.com/alimtvnetwork/movie-cli-v3/tmdb"
)

// isAlreadyScanned checks if a file is already in the DB and updates counters.
func isAlreadyScanned(ctx *ScanContext, vf videoFile, result cleaner.Result) bool {
	existing, searchErr := ctx.Database.SearchMedia(result.CleanTitle)
	if searchErr != nil {
		errlog.Warn("DB search error for '%s': %v", result.CleanTitle, searchErr)
	}

	for i := range existing {
		if existing[i].OriginalFilePath != vf.FullPath {
			continue
		}
		if ctx.UseTable {
			printScanTableRow(buildMediaTableRow(ctx.TotalFiles, &db.Media{
				OriginalFileName: vf.Name,
				CleanTitle:       result.CleanTitle,
				Year:             result.Year,
				Type:             result.Type,
			}, "skipped"))
		} else {
			fmt.Println("     ⏩ Already in database, skipping")
		}
		ctx.Skipped++
		incrementTypeCount(ctx, result.Type)
		return true
	}
	return false
}

// incrementTypeCount bumps MovieCount or TVCount based on media type.
func incrementTypeCount(ctx *ScanContext, mediaType string) {
	if mediaType == string(db.MediaTypeMovie) {
		ctx.MovieCount++
		return
	}
	ctx.TVCount++
}

// logStatError logs a file stat error with appropriate message per spec.
func logStatError(path string, err error) {
	switch {
	case os.IsNotExist(err):
		errlog.Error("❌ File not found: %s", path)
	case os.IsPermission(err):
		errlog.Error("❌ Permission denied: %s", path)
	default:
		errlog.Error("cannot stat file %s: %v", path, err)
	}
}

// handleInsertError handles DB insert failure by attempting update if TmdbID exists.
func handleInsertError(ctx *ScanContext, m *db.Media, insertErr error) {
	if m.TmdbID == 0 {
		errlog.Error("DB insert error for '%s': %v", m.Title, insertErr)
		return
	}

	updateErr := ctx.Database.UpdateMediaByTmdbID(m)
	if updateErr != nil {
		errlog.Error("DB update error for '%s': %v", m.Title, updateErr)
		return
	}

	if m.Genre == "" {
		return
	}

	existing, _ := ctx.Database.GetMediaByTmdbID(m.TmdbID)
	if existing != nil {
		ctx.Database.ReplaceMediaGenres(existing.ID, m.Genre)
	}
}

// trackScanAction records scan_add in action_history for undo support.
func trackScanAction(ctx *ScanContext, m *db.Media, fullPath string, mediaID int64, insertErr error) {
	if insertErr != nil || mediaID <= 0 || ctx.BatchID == "" {
		return
	}
	detail := fmt.Sprintf("Scan added: %s (%s)", m.CleanTitle, fullPath)
	ctx.Database.InsertActionSimple(db.FileActionScanAdd, mediaID, "", detail, ctx.BatchID)
}

// downloadThumbnail downloads poster from TMDb and saves to output + data dirs.
func downloadThumbnail(client *tmdb.Client, database *db.DB, m *db.Media, posterPath, outputDir string) {
	if posterPath == "" {
		return
	}

	slug := cleaner.ToSlug(m.CleanTitle)
	if m.Year > 0 {
		slug += "-" + strconv.Itoa(m.Year)
	}
	thumbFileName := slug + "-" + strconv.Itoa(m.TmdbID) + ".jpg"

	thumbDir := filepath.Join(outputDir, "thumbnails")
	if mkdirErr := os.MkdirAll(thumbDir, 0755); mkdirErr != nil {
		logMkdirError(thumbDir, mkdirErr)
		return
	}

	thumbPath := filepath.Join(thumbDir, thumbFileName)
	if dlErr := client.DownloadPoster(posterPath, thumbPath); dlErr != nil {
		logPosterDownloadError(m.CleanTitle, dlErr)
		return
	}

	m.ThumbnailPath = "thumbnails/" + thumbFileName
	fmt.Println("     🖼️  Thumbnail saved")
	copyThumbnailToDataDir(database.BasePath, thumbPath, thumbFileName)
}

// logMkdirError logs directory creation failure with appropriate message.
func logMkdirError(dir string, err error) {
	if os.IsPermission(err) {
		errlog.Warn("⚠️ Cannot create thumbnail dir — skipping poster download")
		return
	}
	errlog.Error("cannot create thumbnail dir %s: %v", dir, err)
}

// logPosterDownloadError logs poster download failure per spec §1.4.
func logPosterDownloadError(title string, dlErr error) {
	if errors.Is(dlErr, tmdb.ErrTimeout) || errors.Is(dlErr, tmdb.ErrNetworkError) {
		errlog.Warn("⚠️ Poster download timed out — skipping for '%s'", title)
		return
	}
	errlog.Warn("thumbnail download failed for '%s': %v", title, dlErr)
}

// copyThumbnailToDataDir copies a thumbnail to the database data directory for REST access.
func copyThumbnailToDataDir(basePath, thumbPath, fileName string) {
	dbThumbDir := filepath.Join(basePath, "thumbnails")
	if mkErr := os.MkdirAll(dbThumbDir, 0755); mkErr != nil {
		return
	}

	src, rErr := os.ReadFile(thumbPath)
	if rErr != nil {
		return
	}

	dbThumbPath := filepath.Join(dbThumbDir, fileName)
	if wErr := os.WriteFile(dbThumbPath, src, 0644); wErr != nil {
		errlog.Warn("could not copy thumbnail to data dir: %v", wErr)
	}
}
