// movie_info_table.go — table-formatted output for movie info
package cmd

import (
	"fmt"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
)

// printMediaDetailTable outputs a media item as a formatted key-value table.
func printMediaDetailTable(m *db.Media) {
	labelWidth := 12
	valueWidth := 55

	fmt.Println()
	fmt.Printf("  %-*s │ %-*s\n", labelWidth, "Field", valueWidth, "Value")
	fmt.Printf("  %s─┼─%s\n",
		strings.Repeat("─", labelWidth),
		strings.Repeat("─", valueWidth))

	rows := buildDetailTableRows(m, valueWidth)
	for _, r := range rows {
		fmt.Printf("  %-*s │ %-*s\n", labelWidth, r.label, valueWidth, r.value)
	}

	fmt.Printf("  %s─┴─%s\n",
		strings.Repeat("─", labelWidth),
		strings.Repeat("─", valueWidth))
	fmt.Println()
}

type detailRow struct {
	label string
	value string
}

func buildDetailTableRows(m *db.Media, maxWidth int) []detailRow {
	rows := []detailRow{
		{"Title", m.Title},
		{"Year", fmt.Sprintf("%d", m.Year)},
		{"Type", m.Type},
	}

	if m.TmdbID > 0 {
		rows = append(rows, detailRow{"TMDb ID", fmt.Sprintf("%d", m.TmdbID)})
	}
	if m.ImdbID != "" {
		rows = append(rows, detailRow{"IMDb ID", m.ImdbID})
	}

	rows = append(rows, detailRow{"Rating", formatRating(m.TmdbRating, m.ImdbRating)})

	optionalFields := []struct {
		label string
		value string
		show  bool
	}{
		{"Genre", m.Genre, m.Genre != ""},
		{"Director", truncate(m.Director, maxWidth), m.Director != ""},
		{"Cast", truncate(m.CastList, maxWidth), m.CastList != ""},
		{"Runtime", fmt.Sprintf("%d min", m.Runtime), m.Runtime > 0},
		{"Language", m.Language, m.Language != ""},
		{"Tagline", truncate(m.Tagline, maxWidth), m.Tagline != ""},
		{"Trailer", m.TrailerURL, m.TrailerURL != ""},
		{"Budget", fmt.Sprintf("$%d", m.Budget), m.Budget > 0},
		{"Revenue", fmt.Sprintf("$%d", m.Revenue), m.Revenue > 0},
		{"File", truncate(m.CurrentFilePath, maxWidth), m.CurrentFilePath != ""},
		{"Description", truncate(m.Description, maxWidth), m.Description != ""},
	}

	for _, f := range optionalFields {
		if f.show {
			rows = append(rows, detailRow{f.label, f.value})
		}
	}
	return rows
}
