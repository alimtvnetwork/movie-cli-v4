// movie_scan_watch.go — file watcher for movie scan --watch
package cmd

import (
	"fmt"
	"time"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

var scanWatch bool
var scanWatchInterval int

// runWatchLoop polls the scan directory for new video files at a fixed interval.
// It keeps a set of already-seen file paths and processes only new ones.
func runWatchLoop(cfg ScanServiceConfig) {
	seen := make(map[string]bool)

	// Seed with existing files so we don't re-process them
	initial := collectVideoFiles(cfg.ScanDir, scanRecursive, scanDepth)
	for _, vf := range initial {
		seen[vf.FullPath] = true
	}

	interval := time.Duration(scanWatchInterval) * time.Second
	client := tmdb.NewClientWithToken(cfg.Creds.APIKey, cfg.Creds.Token)
	useTMDb := cfg.Creds.HasAuth()

	fmt.Printf("\n  👁️  Watching for new files (every %ds) — press Ctrl+C to stop\n", scanWatchInterval)
	fmt.Println("  ──────────────────────────────────────────")

	cycle := 0
	for {
		time.Sleep(interval)
		cycle++

		current := collectVideoFiles(cfg.ScanDir, scanRecursive, scanDepth)
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

		watchCtx := &ScanContext{
			Database:  cfg.Database,
			Client:    client,
			HasTMDb:   useTMDb,
			OutputDir: cfg.OutputDir,
		}

		for _, vf := range newFiles {
			processVideoFile(vf, watchCtx)
		}

		// Log watch cycle
		if !scanDryRun {
			folderId, folderErr := cfg.Database.UpsertScanFolder(cfg.ScanDir)
			if folderErr != nil {
				errlog.Warn("Could not register scan folder: %v", folderErr)
			} else if histErr := cfg.Database.InsertScanHistory(db.ScanHistoryInput{
				ScanFolderID: int(folderId), TotalFiles: watchCtx.TotalFiles,
				Movies: watchCtx.MovieCount, TV: watchCtx.TVCount,
			}); histErr != nil {
				errlog.Warn("Could not log watch scan history: %v", histErr)
			}
		}

		fmt.Printf("  ✅ Processed: %d files (%d movies, %d TV)\n", watchCtx.TotalFiles, watchCtx.MovieCount, watchCtx.TVCount)
	}
}

