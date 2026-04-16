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
func downloadThumbnailForMedia(input ThumbnailInput) {
	slug := cleaner.ToSlug(input.Media.CleanTitle)
	if input.Media.Year > 0 {
		slug += "-" + strconv.Itoa(input.Media.Year)
	}
	thumbDir := filepath.Join(input.Database.BasePath, "thumbnails", slug)
	if mkdirErr := os.MkdirAll(thumbDir, 0755); mkdirErr != nil {
		errlog.Warn("Cannot create thumbnail dir: %v", mkdirErr)
		return
	}
	thumbPath := filepath.Join(thumbDir, slug+".jpg")
	if dlErr := input.Client.DownloadPoster(input.PosterPath, thumbPath); dlErr != nil {
		errlog.Warn("Thumbnail download failed: %v", dlErr)
		return
	}
	input.Media.ThumbnailPath = thumbPath
}
