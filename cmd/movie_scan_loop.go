// movie_scan_loop.go — main scan processing loop extracted from movie_scan.go.
package cmd

import (
	"fmt"
	"os"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// runMainScanLoop processes all video files: detects removals, rescans existing, processes new.
func runMainScanLoop(ctx *ScanContext, videoFiles []videoFile, creds *tmdb.Client,
	scanDir, scanBatchID string, useJSON, useTable, useTMDb bool, jsonItems *[]scanJSONItem) int {
	database := ctx.Database

	existingMedia, _ := database.GetMediaByScanDir(scanDir)
	diskPaths := make(map[string]bool, len(videoFiles))
	for _, vf := range videoFiles {
		diskPaths[vf.FullPath] = true
	}

	removed := removeStaleEntries(database, existingMedia, diskPaths, scanBatchID, useJSON, useTable)

	existingPaths := make(map[string]*db.Media, len(existingMedia))
	for i := range existingMedia {
		existingPaths[existingMedia[i].OriginalFilePath] = &existingMedia[i]
	}

	client := tmdb.NewClientWithToken(creds.APIKey, creds.AccessToken)
	for _, vf := range videoFiles {
		if em, found := existingPaths[vf.FullPath]; found {
			processExistingMedia(ctx, em, vf, client, database, scanBatchID, useTMDb, useTable, useJSON)
			continue
		}
		processVideoFile(vf, ctx)
	}

	if useJSON {
		for i := range ctx.ScannedItems {
			status := "existing"
			if existingPaths[ctx.ScannedItems[i].OriginalFilePath] == nil {
				status = "new"
			}
			*jsonItems = append(*jsonItems, buildMediaJSONItem(&ctx.ScannedItems[i], status))
		}
	}

	return removed
}

func removeStaleEntries(database *db.DB, existingMedia []db.Media, diskPaths map[string]bool,
	scanBatchID string, useJSON, useTable bool) int {
	var removeIDs []int64
	var removeMedia []*db.Media
	for i := range existingMedia {
		if !diskPaths[existingMedia[i].OriginalFilePath] {
			removeIDs = append(removeIDs, existingMedia[i].ID)
			removeMedia = append(removeMedia, &existingMedia[i])
		}
	}

	if len(removeIDs) == 0 {
		return 0
	}

	snapshotRemovedMedia(database, removeMedia, scanBatchID)

	delCount, delErr := database.DeleteMediaByIDs(removeIDs)
	if delErr != nil {
		errlog.Warn("Could not remove %d stale entries: %v", len(removeIDs), delErr)
		return 0
	}

	if !useJSON && !useTable {
		fmt.Printf("  🗑️  Removed %d entries (files no longer on disk)\n\n", delCount)
	}
	return delCount
}

func snapshotRemovedMedia(database *db.DB, media []*db.Media, scanBatchID string) {
	for _, rm := range media {
		snapshot, snapErr := db.MediaToJSON(rm)
		if snapErr != nil {
			errlog.Warn("Could not snapshot media %d for undo: %v", rm.ID, snapErr)
			continue
		}
		detail := fmt.Sprintf("Scan removed: %s (%s)", rm.CleanTitle, rm.OriginalFilePath)
		database.InsertActionSimple(db.FileActionScanRemove, rm.ID, snapshot, detail, scanBatchID)
	}
}

func processExistingMedia(ctx *ScanContext, em *db.Media, vf videoFile, client *tmdb.Client,
	database *db.DB, scanBatchID string, useTMDb, useTable, useJSON bool) {
	ctx.TotalFiles++

	if useTMDb && mediaNeedsRescan(em) {
		handleRescan(ctx, em, client, database, scanBatchID, useTable, useJSON)
	} else {
		handleSkippedMedia(ctx, em, useTable, useJSON)
	}

	ctx.ScannedItems = append(ctx.ScannedItems, *em)
	if em.Type == string(db.MediaTypeMovie) {
		ctx.MovieCount++
	} else {
		ctx.TVCount++
	}
}

func handleRescan(ctx *ScanContext, em *db.Media, client *tmdb.Client,
	database *db.DB, scanBatchID string, useTable, useJSON bool) {
	preSnapshot, _ := db.MediaToJSON(em)
	if rescanMediaEntry(database, client, em) {
		detail := fmt.Sprintf("Rescan updated: %s", em.CleanTitle)
		database.InsertActionSimple(db.FileActionRescanUpdate, em.ID, preSnapshot, detail, scanBatchID)
		if useTable {
			printScanTableRow(buildMediaTableRow(ctx.TotalFiles, em, "rescanned"))
		} else if !useJSON {
			printRescanSuccess(ctx.TotalFiles, em)
		}
	} else {
		ctx.Skipped++
		if !useTable && !useJSON {
			printRescanFailed(ctx.TotalFiles, em)
		}
	}
}

func printRescanSuccess(idx int, em *db.Media) {
	typeIcon := db.TypeIcon(em.Type)
	fmt.Printf("\n  %d. %s %s", idx, typeIcon, em.CleanTitle)
	if em.Year > 0 {
		fmt.Printf(" (%d)", em.Year)
	}
	fmt.Printf(" [%s]\n", em.Type)
	fmt.Printf("     🔄 Rescanned — ⭐%.1f %s\n", em.TmdbRating, em.Genre)
}

func printRescanFailed(idx int, em *db.Media) {
	fmt.Printf("\n  %d. %s", idx, em.CleanTitle)
	fmt.Printf(" [%s]\n", em.Type)
	fmt.Println("     ⚠️  Rescan failed — kept existing data")
}

func handleSkippedMedia(ctx *ScanContext, em *db.Media, useTable, useJSON bool) {
	ctx.Skipped++
	if useTable {
		printScanTableRow(buildMediaTableRow(ctx.TotalFiles, em, "existing"))
	} else if !useJSON {
		typeIcon := db.TypeIcon(em.Type)
		fmt.Printf("\n  %d. %s %s", ctx.TotalFiles, typeIcon, em.CleanTitle)
		if em.Year > 0 {
			fmt.Printf(" (%d)", em.Year)
		}
		fmt.Printf(" [%s]\n", em.Type)
		fmt.Println("     ⏩ Already in database")
	}
}

// printScanFooter shows the scan summary at the end.
func printScanFooter(scanDir, outputDir string, items []db.Media, total, movies, tv, skipped, removed int) {
	fmt.Println()
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  📊 Scan complete: %d total (%d movies, %d TV shows)\n", total, movies, tv)
	if skipped > 0 {
		fmt.Printf("     ⏩ %d skipped (already in DB)\n", skipped)
	}
	if removed > 0 {
		fmt.Printf("     🗑️  %d removed (files deleted from disk)\n", removed)
	}
	fmt.Printf("  📂 Output: %s\n", outputDir)

	if !scanDryRun {
		writeScanSummary(scanDir, outputDir, items, total, movies, tv, skipped)
		if err := writeHTMLReport(scanDir, outputDir, items, total, movies, tv, scanRestPort); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  HTML report: %v\n", err)
		}
	}
}
