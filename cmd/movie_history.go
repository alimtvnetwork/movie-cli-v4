// movie_history.go — movie history: unified view of all tracked operations.
//
// Shows moves, renames, scans, deletions, popouts, and rescans from both
// move_history and action_history tables.
//
// Flags:
//
//	--type <type>    Filter by type: move, scan, delete, popout, rescan, all (default: all)
//	--batch <id>     Show all actions in a specific batch
//	--limit <n>      Max records to show (default: 20)
//	--format <fmt>   Output format: default, json, table
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var (
	historyFormat  string
	historyType    string
	historyBatch   string
	historySince   string
	historyLimit   int
)

var movieHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show history of all tracked operations",
	Long: `Displays the history of all state-changing operations including
file moves, renames, scans, deletions, popouts, and metadata rescans.

Flags:
  --type <type>   Filter: move, scan, delete, popout, rescan, all (default: all)
  --batch <id>    Show all actions in a specific batch
  --since <date>  Show only records after this date (e.g. 2026-04-01)
  --limit <n>     Max records (default: 20)
  --format <fmt>  Output: default, json, table`,
	Run: runMovieHistory,
}

func init() {
	movieHistoryCmd.Flags().StringVar(&historyFormat, "format", "default", "output format: default, json, table")
	movieHistoryCmd.Flags().StringVar(&historyType, "type", "all", "filter: move, scan, delete, popout, rescan, all")
	movieHistoryCmd.Flags().StringVar(&historyBatch, "batch", "", "show actions for a specific batch ID")
	movieHistoryCmd.Flags().StringVar(&historySince, "since", "", "show records after this date (e.g. 2026-04-01)")
	movieHistoryCmd.Flags().IntVar(&historyLimit, "limit", 20, "max records to show")
}

// unifiedRecord merges move_history and action_history into one display item.
type unifiedRecord struct {
	Source     string `json:"source"`
	ID         int64  `json:"id"`
	Type       string `json:"type"`
	Detail     string `json:"detail"`
	FromPath   string `json:"from_path,omitempty"`
	ToPath     string `json:"to_path,omitempty"`
	BatchID    string `json:"batch_id,omitempty"`
	Timestamp  string `json:"timestamp"`
	IsReverted bool   `json:"is_reverted"`
}

func runMovieHistory(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error(msgDatabaseError, err)
		return
	}
	defer database.Close()

	if historyBatch != "" {
		showBatchHistory(database)
		return
	}

	records := collectUnifiedRecords(database)

	if len(records) == 0 {
		if historyFormat == "json" {
			fmt.Println("[]")
			return
		}
		fmt.Println("📭 No history found.")
		return
	}

	switch historyFormat {
	case string(db.OutputFormatJSON):
		printUnifiedJSON(records)
	case string(db.OutputFormatTable):
		printUnifiedTable(records)
	default:
		printUnifiedDefault(records)
	}
}

// ---------------------------------------------------------------------------
// Output formatters
// ---------------------------------------------------------------------------

func printUnifiedDefault(records []unifiedRecord) {
	fmt.Printf("📋 History (%d records)\n\n", len(records))

	for _, r := range records {
		status := "✅"
		if r.IsReverted {
			status = "↩️ "
		}

		icon := typeIcon(r.Type)
		fmt.Printf("  %s %s %-14s  %s\n", status, icon, r.Type, r.Timestamp)
		fmt.Printf("     %s\n", r.Detail)

		if r.FromPath != "" {
			fmt.Printf("     Path: %s → %s\n", r.FromPath, r.ToPath)
		}
		if r.BatchID != "" {
			fmt.Printf("     Batch: %s\n", r.BatchID[:minInt(8, len(r.BatchID))])
		}
		fmt.Println()
	}
}

func printUnifiedJSON(records []unifiedRecord) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		errlog.Error("JSON encode error: %v", err)
	}
}

func printUnifiedTable(records []unifiedRecord) {
	printHistoryTableUnified(records)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func typeIcon(t string) string {
	switch t {
	case "move":
		return "📁"
	case "rename":
		return "✏️ "
	case "scan_add":
		return "➕"
	case "scan_remove":
		return "➖"
	case "delete":
		return "🗑 "
	case "popout":
		return "📤"
	case "restore":
		return "♻️ "
	case "rescan_update":
		return "🔄"
	default:
		return "📋"
	}
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sortRecordsByTimestamp sorts unified records by timestamp descending.
func sortRecordsByTimestamp(records []unifiedRecord) {
	for i := 1; i < len(records); i++ {
		for j := i; j > 0 && records[j].Timestamp > records[j-1].Timestamp; j-- {
			records[j], records[j-1] = records[j-1], records[j]
		}
	}
}
