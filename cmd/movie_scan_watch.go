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
	ws := WatchState{
		Client:  tmdb.NewClientWithToken(cfg.Creds.APIKey, cfg.Creds.Token),
		HasTMDb: cfg.Creds.HasAuth(),
		Seen:    seedWatchSeen(cfg.ScanDir),
	}

	fmt.Printf("\n  👁️  Watching for new files (every %ds) — press Ctrl+C to stop\n", scanWatchInterval)
	fmt.Println("  ──────────────────────────────────────────")

	for {
		time.Sleep(interval)
		processWatchCycle(cfg, ws)
	}
}

func seedWatchSeen(scanDir string) map[string]bool {
	seen := make(map[string]bool)
	for _, vf := range collectVideoFiles(scanDir, scanRecursive, scanDepth) {
		seen[vf.FullPath] = true
	}
	return seen
}

func processWatchCycle(cfg ScanServiceConfig, ws WatchState) {
	current := collectVideoFiles(cfg.ScanDir, scanRecursive, scanDepth)
	var newFiles []videoFile
	for _, vf := range current {
		if !ws.Seen[vf.FullPath] {
			newFiles = append(newFiles, vf)
			ws.Seen[vf.FullPath] = true
		}
	}

	if len(newFiles) == 0 {
		return
	}

	fmt.Printf("\n  🔔 Detected %d new file(s) at %s\n",
		len(newFiles), time.Now().Format("15:04:05"))

	watchCtx := &ScanContext{
		Database: cfg.Database, Client: ws.Client,
		HasTMDb: ws.HasTMDb, OutputDir: cfg.OutputDir,
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
