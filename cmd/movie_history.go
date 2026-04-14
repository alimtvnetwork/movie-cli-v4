// movie_history.go — movie history command
// Shows move/rename history from the database.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
)

var historyFormat string

var movieHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show move and rename history",
	Long: `Displays the history of file move and rename operations.

Use --format json to output as JSON.
Use --format table to output as a formatted table.`,
	Run: runMovieHistory,
}

func init() {
	movieHistoryCmd.Flags().StringVar(&historyFormat, "format", "default",
		"output format: default, json, or table")
}

func runMovieHistory(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Database error: %v\n", err)
		return
	}
	defer database.Close()

	records, listErr := database.ListMoveHistory(0)
	if listErr != nil {
		fmt.Fprintf(os.Stderr, "❌ Error reading history: %v\n", listErr)
		return
	}

	if len(records) == 0 {
		if historyFormat == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("📭 No move history found.")
		}
		return
	}

	switch historyFormat {
	case "json":
		printHistoryJSON(records)
	case "table":
		printHistoryTable(records)
	default:
		printHistoryDefault(records)
	}
}

func printHistoryDefault(records []db.MoveRecord) {
	fmt.Printf("📋 Move History (%d records)\n\n", len(records))

	for i := range records {
		r := &records[i]
		status := "✅"
		if r.Undone {
			status = "↩️ "
		}

		fmt.Printf("  %s #%-4d  %s\n", status, r.ID, r.MovedAt)
		fmt.Printf("       From: %s\n", r.OriginalFileName)
		fmt.Printf("       To:   %s\n", r.NewFileName)
		fmt.Printf("       Path: %s → %s\n\n", r.FromPath, r.ToPath)
	}
}

// historyJSONItem is the JSON representation of a move record.
type historyJSONItem struct {
	ID               int64  `json:"id"`
	MediaID          int64  `json:"media_id"`
	FromPath         string `json:"from_path"`
	ToPath           string `json:"to_path"`
	OriginalFileName string `json:"original_file_name"`
	NewFileName      string `json:"new_file_name"`
	MovedAt          string `json:"moved_at"`
	Undone           bool   `json:"undone"`
}

func printHistoryJSON(records []db.MoveRecord) {
	items := make([]historyJSONItem, len(records))
	for i := range records {
		items[i] = historyJSONItem{
			ID:               records[i].ID,
			MediaID:          records[i].MediaID,
			FromPath:         records[i].FromPath,
			ToPath:           records[i].ToPath,
			OriginalFileName: records[i].OriginalFileName,
			NewFileName:      records[i].NewFileName,
			MovedAt:          records[i].MovedAt,
			Undone:           records[i].Undone,
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(items); encErr != nil {
		fmt.Fprintf(os.Stderr, "❌ JSON encode error: %v\n", encErr)
	}
}
