// movie_rescan.go — movie rescan — re-fetches TMDb data for entries with missing metadata
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// regenerateReports rebuilds HTML report and summary.json for every scan
// directory that has media in the DB.
func regenerateReports(database *db.DB) {
	allMedia, err := database.ListAllMedia()
	if err != nil {
		errlog.Warn("Could not list media for report regeneration: %v", err)
		return
	}
	if len(allMedia) == 0 {
		return
	}

	dirMap := make(map[string][]db.Media)
	for _, m := range allMedia {
		if m.OriginalFilePath == "" {
			continue
		}
		scanDir := filepath.Dir(m.OriginalFilePath)
		dirMap[scanDir] = append(dirMap[scanDir], m)
	}

	for scanDir, items := range dirMap {
		regenerateReportForDir(scanDir, items)
	}
}

func regenerateReportForDir(scanDir string, items []db.Media) {
	outputDir := filepath.Join(scanDir, ".movie-output")
	if _, statErr := os.Stat(outputDir); os.IsNotExist(statErr) {
		return
	}

	movieCount, tvCount := countByType(items)
	stats := ScanStats{
		ScanDir: scanDir, OutputDir: outputDir, Items: items,
		Total: len(items), Movies: movieCount, TV: tvCount,
	}

	if summaryErr := writeScanSummary(stats); summaryErr != nil {
		errlog.Warn("Could not regenerate summary.json for %s: %v", scanDir, summaryErr)
	}
	htmlErr := writeHTMLReport(stats)
	if htmlErr != nil {
		errlog.Warn("Could not regenerate report.html for %s: %v", scanDir, htmlErr)
		return
	}
	fmt.Printf("🌐 Regenerated report.html → %s\n", filepath.Join(outputDir, "report.html"))
}

func countByType(items []db.Media) (int, int) {
	movieCount, tvCount := 0, 0
	for _, m := range items {
		if m.Type == string(db.MediaTypeMovie) {
			movieCount++
			continue
		}
		tvCount++
	}
	return movieCount, tvCount
}

var rescanAll bool
var rescanLimit int

var movieRescanCmd = &cobra.Command{
	Use:   "rescan",
	Short: "Re-fetch TMDb metadata for entries with missing data",
	Long: `Scans the database for media entries that have missing genre, rating,
or description, and re-fetches their metadata from TMDb.

This is useful after fixing API keys or when earlier scans failed to
retrieve complete metadata. No folder scan is needed.

Examples:
  movie rescan              Re-fetch only entries with missing data
  movie rescan --all        Re-fetch TMDb data for ALL entries
  movie rescan --limit 50   Process at most 50 entries`,
	Run: runMovieRescan,
}

func init() {
	movieRescanCmd.Flags().BoolVar(&rescanAll, "all", false,
		"re-fetch TMDb data for all entries, not just those with missing data")
	movieRescanCmd.Flags().IntVar(&rescanLimit, "limit", 0,
		"max number of entries to process (0 = unlimited)")
}

func runMovieRescan(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Database error: %v\n", err)
		return
	}
	defer database.Close()

	creds := resolveScanTMDbCredentials(database)
	if !creds.HasAuth() {
		fmt.Fprintln(os.Stderr, "❌ No TMDb credentials configured. Run: movie config set tmdb_api_key YOUR_KEY")
		return
	}

	if initErr := errlog.Init("", "rescan"); initErr != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not init error logger: %v\n", initErr)
	}
	defer errlog.Close()

	entries, err := fetchRescanEntries(database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Database query error: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("✅ All entries have complete metadata. Nothing to rescan.")
		return
	}

	client := tmdb.NewClientWithToken(creds.APIKey, creds.Token)
	updated, failed := processRescanEntries(database, client, entries)
	printRescanResult(updated, failed, len(entries))

	if updated > 0 {
		regenerateReports(database)
	}
}

func fetchRescanEntries(database *db.DB) ([]db.Media, error) {
	var entries []db.Media
	var err error
	if rescanAll {
		entries, err = database.ListAllMedia()
	}
	if !rescanAll {
		entries, err = database.GetMediaWithMissingData()
	}
	if err != nil {
		return nil, err
	}
	return applyRescanLimit(entries), nil
}

func applyRescanLimit(entries []db.Media) []db.Media {
	if rescanLimit > 0 && len(entries) > rescanLimit {
		return entries[:rescanLimit]
	}
	return entries
}

func processRescanEntries(database *db.DB, client *tmdb.Client, entries []db.Media) (int, int) {
	fmt.Printf("\n🔄 Rescanning %d entries for TMDb metadata...\n\n", len(entries))
	updated, failed := 0, 0
	for i, m := range entries {
		fmt.Printf("  %d/%d  %s", i+1, len(entries), m.CleanTitle)
		if m.Year > 0 {
			fmt.Printf(" (%d)", m.Year)
		}

		if rescanMediaEntry(database, client, &m) {
			fmt.Printf("  ✅ ⭐%.1f %s\n", m.TmdbRating, m.Genre)
			updated++
			continue
		}
		fmt.Printf("  ❌ failed\n")
		failed++
	}
	return updated, failed
}

func printRescanResult(updated, failed, total int) {
	fmt.Printf("\n📊 Rescan Complete!\n")
	fmt.Printf("   Updated:  %d\n", updated)
	fmt.Printf("   Failed:   %d\n", failed)
	fmt.Printf("   Total:    %d\n\n", total)
}
