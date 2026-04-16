// movie_info_helpers.go — helpers shared by movie info and scan for thumbnail downloads.
package cmd

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// downloadThumbnailForMedia downloads a poster and sets m.ThumbnailPath.
func downloadThumbnailForMedia(client *tmdb.Client, database *db.DB, m *db.Media, posterPath string) {
	slug := cleaner.ToSlug(m.CleanTitle)
	if m.Year > 0 {
		slug += "-" + strconv.Itoa(m.Year)
	}
	thumbDir := filepath.Join(database.BasePath, "thumbnails", slug)
	if mkdirErr := os.MkdirAll(thumbDir, 0755); mkdirErr != nil {
		errlog.Warn("Cannot create thumbnail dir: %v", mkdirErr)
		return
	}
	thumbPath := filepath.Join(thumbDir, slug+".jpg")
	if dlErr := client.DownloadPoster(posterPath, thumbPath); dlErr != nil {
		errlog.Warn("Thumbnail download failed: %v", dlErr)
		return
	}
	m.ThumbnailPath = thumbPath
}
