// movie_scan_helpers_print.go — print helpers for scan footer (extracted from movie_scan_helpers.go)
package cmd

import (
	"fmt"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

func printScanCounts(totalFiles, movieCount, tvCount, skipped, removed int) {
	fmt.Printf("     Total files: %d\n", totalFiles)
	fmt.Printf("     Movies:      %d\n", movieCount)
	fmt.Printf("     TV Shows:    %d\n", tvCount)
	newCount := totalFiles - skipped
	if newCount > 0 {
		fmt.Printf("     New:         %d\n", newCount)
	}
	if skipped > 0 {
		fmt.Printf("     Existing:    %d (already in DB)\n", skipped)
	}
	if removed > 0 {
		fmt.Printf("     Removed:     %d (files no longer on disk)\n", removed)
	}
}

func printScanOutputFiles(outputDir, scanDir string, items []db.Media, total, movies, tv, skipped int) {
	fmt.Println()
	fmt.Println("  ■ Output Files")
	fmt.Println("  ──────────────────────────────────────────")
	fmt.Printf("  📁 %s/\n", outputDir)

	writeScanOutputSummary(outputDir, scanDir, items, total, movies, tv, skipped)
	writeScanOutputHTML(outputDir, scanDir, items, total, movies, tv, skipped)

	fmt.Printf("  ├── 📁 json/movie/       Per-movie JSON metadata\n")
	fmt.Printf("  ├── 📁 json/tv/          Per-show JSON metadata\n")
	fmt.Printf("  └── 📁 thumbnails/       Movie poster thumbnails\n")
}

func writeScanOutputSummary(outputDir, scanDir string, items []db.Media, total, movies, tv, skipped int) {
	summaryErr := writeScanSummary(outputDir, scanDir, items, total, movies, tv, skipped)
	if summaryErr != nil {
		errlog.Warn("Could not write summary.json: %v", summaryErr)
		return
	}
	fmt.Printf("  ├── 📄 summary.json      Scan report with metadata\n")
}

func writeScanOutputHTML(outputDir, scanDir string, items []db.Media, total, movies, tv, skipped int) {
	htmlErr := writeHTMLReport(outputDir, scanDir, items, total, movies, tv, skipped)
	if htmlErr != nil {
		errlog.Warn("Could not write report.html: %v", htmlErr)
		return
	}
	fmt.Printf("  ├── 🌐 report.html       Interactive HTML report\n")
}
