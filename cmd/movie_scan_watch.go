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
func runWatchLoop(cfg ScanServiceConfig) {
	seen := seedWatchSeen(cfg.ScanDir)
	interval := time.Duration(scanWatchInterval) * time.Second
	client := tmdb.NewClientWithToken(cfg.Creds.APIKey, cfg.Creds.Token)
	useTMDb := cfg.Creds.HasAuth()

	fmt.Printf("\n  👁️  Watching for new files (every %ds) — press Ctrl+C to stop\n", scanWatchInterval)
	fmt.Println("  ──────────────────────────────────────────")

	for {
		time.Sleep(interval)
		processWatchCycle(cfg, client, useTMDb, seen)
	}
}

func seedWatchSeen(scanDir string) map[string]bool {
	seen := make(map[string]bool)
	for _, vf := range collectVideoFiles(scanDir, scanRecursive, scanDepth) {
		seen[vf.FullPath] = true
	}
	return seen
}

func processWatchCycle(cfg ScanServiceConfig, client *tmdb.Client, useTMDb bool, seen map[string]bool) {
	current := collectVideoFiles(cfg.ScanDir, scanRecursive, scanDepth)
	var newFiles []videoFile
	for _, vf := range current {
		if !seen[vf.FullPath] {
			newFiles = append(newFiles, vf)
			seen[vf.FullPath] = true
		}
	}

	if len(newFiles) == 0 {
		return
	}

	fmt.Printf("\n  🔔 Detected %d new file(s) at %s\n",
		len(newFiles), time.Now().Format("15:04:05"))

	watchCtx := &ScanContext{
		Database: cfg.Database, Client: client,
		HasTMDb: useTMDb, OutputDir: cfg.OutputDir,
	}

	for _, vf := range newFiles {
		processVideoFile(vf, watchCtx)
	}

	logWatchScanHistory(cfg, watchCtx)
	fmt.Printf("  ✅ Processed: %d files (%d movies, %d TV)\n",
		watchCtx.TotalFiles, watchCtx.MovieCount, watchCtx.TVCount)
}

func logWatchScanHistory(cfg ScanServiceConfig, ctx *ScanContext) {
	if scanDryRun {
		return
	}
	folderId, folderErr := cfg.Database.UpsertScanFolder(cfg.ScanDir)
	if folderErr != nil {
		errlog.Warn("Could not register scan folder: %v", folderErr)
		return
	}
	if histErr := cfg.Database.InsertScanHistory(db.ScanHistoryInput{
		ScanFolderID: int(folderId), TotalFiles: ctx.TotalFiles,
		Movies: ctx.MovieCount, TV: ctx.TVCount,
	}); histErr != nil {
		errlog.Warn("Could not log watch scan history: %v", histErr)
	}
}
