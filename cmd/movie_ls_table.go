// movie_ls_table.go — table-formatted output for movie ls
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
)

// table column widths
const (
	colNum      = 5
	colTitle    = 40
	colYear     = 6
	colType     = 8
	colRating   = 6
	colGenre    = 25
	colDirector = 20
)

// runMovieLsTable outputs all library items as a formatted table (no pager).
func runMovieLsTable(database *db.DB) {
	allMedia, err := database.ListMedia(0, 100000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Database error: %v\n", err)
		return
	}

	if len(allMedia) == 0 {
		fmt.Println("📭 No media found. Run 'movie scan <folder>' first.")
		return
	}

	fmt.Println()
	printLsTableHeader()
	for i := range allMedia {
		printLsTableRow(i+1, &allMedia[i])
	}
	printLsTableDivider("┴")
	fmt.Printf("\n  Total: %d items\n\n", len(allMedia))
}

func printLsTableHeader() {
	fmt.Printf("  %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s\n",
		colNum, "#", colTitle, "Title", colYear, "Year",
		colType, "Type", colRating, "Rating", colGenre, "Genre",
		colDirector, "Director")
	printLsTableDivider("┼")
}

func printLsTableDivider(joint string) {
	fmt.Printf("  %s─%s─%s─%s─%s─%s─%s─%s─%s─%s─%s─%s─%s\n",
		strings.Repeat("─", colNum), joint,
		strings.Repeat("─", colTitle), joint,
		strings.Repeat("─", colYear), joint,
		strings.Repeat("─", colType), joint,
		strings.Repeat("─", colRating), joint,
		strings.Repeat("─", colGenre), joint,
		strings.Repeat("─", colDirector))
}

func printLsTableRow(num int, m *db.Media) {
	title := truncate(m.CleanTitle, colTitle)
	yearStr := ""
	if m.Year > 0 {
		yearStr = fmt.Sprintf("%d", m.Year)
	}

	rating := formatRating(m.TmdbRating, m.ImdbRating)
	genre := truncate(m.Genre, colGenre)
	director := truncate(m.Director, colDirector)

	fmt.Printf("  %-*d │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s\n",
		colNum, num, colTitle, title, colYear, yearStr,
		colType, capitalize(m.Type), colRating, rating,
		colGenre, genre, colDirector, director)
}

func formatRating(tmdbRating, imdbRating float64) string {
	if tmdbRating > 0 {
		return fmt.Sprintf("%.1f", tmdbRating)
	}
	if imdbRating > 0 {
		return fmt.Sprintf("%.1f", imdbRating)
	}
	return "N/A"
}

// truncate shortens a string to maxLen, adding "…" if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return s[:maxLen-1] + "…"
}
