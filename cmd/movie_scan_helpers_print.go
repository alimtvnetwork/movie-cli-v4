// movie_scan_helpers_print.go — print helpers for scan footer (extracted from movie_scan_helpers.go)
package cmd

import (
	"fmt"

	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

func printScanCounts(stats ScanStats) {
	fmt.Printf("     Total files: %d\n", stats.Total)
	fmt.Printf("     Movies:      %d\n", stats.Movies)
	fmt.Printf("     TV Shows:    %d\n", stats.TV)
	newCount := stats.Total - stats.Skipped
	if newCount > 0 {
		fmt.Printf("     New:         %d\n", newCount)
	}
	if stats.Skipped > 0 {
		fmt.Printf("     Existing:    %d (already in DB)\n", stats.Skipped)
	}
	if stats.Removed > 0 {
		fmt.Printf("     Removed:     %d (files no longer on disk)\n", stats.Removed)
	}
}

func printScanOutputFiles(stats ScanStats) {
	fmt.Println()
	fmt.Println("  ■ Output Files")
	fmt.Println("  ──────────────────────────────────────────")
	fmt.Printf("  📁 %s/\n", stats.OutputDir)

	writeScanOutputSummary(stats)
	writeScanOutputHTML(stats)

	fmt.Printf("  ├── 📁 json/movie/       Per-movie JSON metadata\n")
	fmt.Printf("  ├── 📁 json/tv/          Per-show JSON metadata\n")
	fmt.Printf("  └── 📁 thumbnails/       Movie poster thumbnails\n")
}

func writeScanOutputSummary(stats ScanStats) {
	if err := writeScanSummary(stats); err != nil {
		errlog.Warn("Could not write summary.json: %v", err)
		return
	}
	fmt.Printf("  ├── 📄 summary.json      Scan report with metadata\n")
}

func writeScanOutputHTML(stats ScanStats) {
	if err := writeHTMLReport(stats); err != nil {
		errlog.Warn("Could not write report.html: %v", err)
		return
	}
	fmt.Printf("  ├── 🌐 report.html       Interactive HTML report\n")
}
