// movie_history_table.go — table-formatted output for movie history
package cmd

import (
	"fmt"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v3/db"
)

func printHistoryTable(records []db.MoveRecord) {
	idW := 5
	statusW := 6
	dateW := 19
	fromW := 30
	toW := 30

	fmt.Println()
	fmt.Printf("  %-*s │ %-*s │ %-*s │ %-*s │ %-*s\n",
		idW, "ID",
		statusW, "Status",
		dateW, "Date",
		fromW, "Original Name",
		toW, "New Name")

	fmt.Printf("  %s─┼─%s─┼─%s─┼─%s─┼─%s\n",
		strings.Repeat("─", idW),
		strings.Repeat("─", statusW),
		strings.Repeat("─", dateW),
		strings.Repeat("─", fromW),
		strings.Repeat("─", toW))

	for i := range records {
		r := &records[i]
		status := "OK"
		if r.Undone {
			status = "Undone"
		}

		origName := truncate(r.OriginalFileName, fromW)
		newName := truncate(r.NewFileName, toW)
		date := truncate(r.MovedAt, dateW)

		fmt.Printf("  %-*d │ %-*s │ %-*s │ %-*s │ %-*s\n",
			idW, r.ID,
			statusW, status,
			dateW, date,
			fromW, origName,
			toW, newName)
	}

	fmt.Printf("  %s─┴─%s─┴─%s─┴─%s─┴─%s\n",
		strings.Repeat("─", idW),
		strings.Repeat("─", statusW),
		strings.Repeat("─", dateW),
		strings.Repeat("─", fromW),
		strings.Repeat("─", toW))

	fmt.Printf("\n  Total: %d records\n\n", len(records))
}
