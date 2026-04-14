// movie_scan_watch.go — file watcher for movie scan --watch
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alimtvnetwork/movie-cli-v3/cleaner"
	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
	"github.com/alimtvnetwork/movie-cli-v3/tmdb"
)

var scanWatch bool
var scanWatchInterval int

// runWatchLoop polls the scan directory for new video files at a fixed interval.
// It keeps a set of already-seen file paths and processes only new ones.
func runWatchLoop(scanDir, outputDir string, database *db.DB, creds tmdbCreds) {
	seen := make(map[string]bool)

	// Seed with existing files so we don't re-process them
	initial := collectVideoFiles(scanDir, scanRecursive, scanDepth)
	for _, vf := range initial {
		seen[vf.FullPath] = true
	}

	interval := time.Duration(scanWatchInterval) * time.Second
	client := tmdb.NewClientWithToken(creds.APIKey, creds.Token)
	useTMDb := creds.HasAuth()

	fmt.Printf("\n  👁️  Watching for new files (every %ds) — press Ctrl+C to stop\n", scanWatchInterval)
	fmt.Println("  ──────────────────────────────────────────")

	cycle := 0
	for {
		time.Sleep(interval)
		cycle++

		current := collectVideoFiles(scanDir, scanRecursive, scanDepth)
		var newFiles []videoFile
		for _, vf := range current {
			if !seen[vf.FullPath] {
				newFiles = append(newFiles, vf)
				seen[vf.FullPath] = true
			}
		}

		if len(newFiles) == 0 {
			continue
		}

		fmt.Printf("\n  🔔 Detected %d new file(s) at %s\n",
			len(newFiles), time.Now().Format("15:04:05"))

		var totalFiles, movieCount, tvCount, skipped int
		var scannedItems []db.Media

		for _, vf := range newFiles {
			processVideoFile(vf, database, client, useTMDb, outputDir,
				&totalFiles, &movieCount, &tvCount, &skipped, &scannedItems, false)
		}

		// Log watch cycle
		if !scanDryRun {
			if histErr := database.InsertScanHistory(scanDir, totalFiles, movieCount, tvCount); histErr != nil {
				errlog.Warn("Could not log watch scan history: %v", histErr)
			}
		}

		fmt.Printf("  ✅ Processed: %d files (%d movies, %d TV)\n", totalFiles, movieCount, tvCount)
	}
}

// isVideoFileInDir checks if a path is a video file within the scan directory.
func isVideoFileInDir(path, scanDir string) bool {
	rel, err := filepath.Rel(scanDir, path)
	if err != nil || rel == "." {
		return false
	}
	return cleaner.IsVideoFile(filepath.Base(path))
}

// ignoreWatchDir returns true for directories the watcher should skip.
func ignoreWatchDir(name string) bool {
	return name == ".movie-output" ||
		(len(name) > 0 && name[0] == '.')
}

// watchPrintNewFile prints a discovered file in watch mode.
func watchPrintNewFile(index int, vf videoFile) {
	result := cleaner.Clean(vf.Name)
	typeIcon := "🎬"
	if result.Type == "tv" {
		typeIcon = "📺"
	}
	fmt.Printf("  %d. %s %s", index, typeIcon, result.CleanTitle)
	if result.Year > 0 {
		fmt.Printf(" (%d)", result.Year)
	}
	fmt.Printf(" [%s]\n", result.Type)
	fmt.Printf("     └─ %s\n", vf.Name)
}

// ensureWatchOutputDirs creates output dirs if they don't exist yet.
func ensureWatchOutputDirs(outputDir string) {
	if err := os.MkdirAll(filepath.Join(outputDir, "thumbnails"), 0755); err != nil {
		errlog.Warn("watch: could not ensure output dirs: %v", err)
	}
}
