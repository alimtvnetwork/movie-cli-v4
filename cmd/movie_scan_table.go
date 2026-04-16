// movie_scan_table.go — table-formatted output for movie scan
package cmd

import (
	"fmt"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v3/cleaner"
	"github.com/alimtvnetwork/movie-cli-v3/db"
)

// scanTableRow holds data for one row in the scan table output.
type scanTableRow struct {
	Index      int
	FileName   string
	CleanTitle string
	Year       int
	Type       string
	Rating     float64
	Status     string // "new", "skipped", "error"
}

// printScanTableHeader prints the table header row.
func printScanTableHeader() {
	fmt.Println()
	fmt.Printf("  %-4s │ %-30s │ %-30s │ %-5s │ %-6s │ %-6s │ %-8s\n",
		"#", "File Name", "Clean Title", "Year", "Type", "Rating", "Status")
	fmt.Printf("  %s─┼─%s─┼─%s─┼─%s─┼─%s─┼─%s─┼─%s\n",
		strings.Repeat("─", 4),
		strings.Repeat("─", 30),
		strings.Repeat("─", 30),
		strings.Repeat("─", 5),
		strings.Repeat("─", 6),
		strings.Repeat("─", 6),
		strings.Repeat("─", 8))
}

// printScanTableRow prints a single row in the scan table.
// SHARED: used by scan, scan_process, scan_process_helpers
func printScanTableRow(row scanTableRow) {
	fileName := truncate(row.FileName, 30)
	title := truncate(row.CleanTitle, 30)

	yearStr := "  -  "
	if row.Year > 0 {
		yearStr = fmt.Sprintf("%5d", row.Year)
	}

	ratingStr := "   -  "
	if row.Rating > 0 {
		ratingStr = fmt.Sprintf("%5.1f ", row.Rating)
	}

	statusIcon := "✅ new"
	switch row.Status {
	case "skipped":
		statusIcon = "⏩ skip"
	case "error":
		statusIcon = "❌ err"
	}

	fmt.Printf("  %-4d │ %-30s │ %-30s │ %s │ %-6s │ %s│ %s\n",
		row.Index, fileName, title, yearStr, row.Type, ratingStr, statusIcon)
}

// printScanTableFooter prints a closing line after the table.
func printScanTableFooter() {
	fmt.Printf("  %s─┴─%s─┴─%s─┴─%s─┴─%s─┴─%s─┴─%s\n",
		strings.Repeat("─", 4),
		strings.Repeat("─", 30),
		strings.Repeat("─", 30),
		strings.Repeat("─", 5),
		strings.Repeat("─", 6),
		strings.Repeat("─", 6),
		strings.Repeat("─", 8))
}

// buildDryRunTableRows creates table rows from video files in dry-run mode.
func buildDryRunTableRows(videoFiles []videoFile) (rows []scanTableRow, movies, tvShows int) {
	for i, vf := range videoFiles {
		result := cleaner.Clean(vf.Name)
		row := scanTableRow{
			Index:      i + 1,
			FileName:   vf.Name,
			CleanTitle: result.CleanTitle,
			Year:       result.Year,
			Type:       result.Type,
			Status:     "new",
		}
		rows = append(rows, row)
		if result.Type == string(db.MediaTypeMovie) {
			movies++
		} else {
			tvShows++
		}
	}
	return
}

// buildMediaTableRow creates a table row from a processed Media item.
// SHARED: used by scan, scan_process, scan_process_helpers
func buildMediaTableRow(index int, m *db.Media, status string) scanTableRow {
	return scanTableRow{
		Index:      index,
		FileName:   m.OriginalFileName,
		CleanTitle: m.CleanTitle,
		Year:       m.Year,
		Type:       m.Type,
		Rating:     m.TmdbRating,
		Status:     status,
	}
}

