// movie_logs.go — movie logs: display recent error logs from the database
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
)

var logsLimit int
var logsLevel string
var logsFormat string

var movieLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Display recent error logs from the database",
	Long: `Show recent error, warning, and info log entries stored in the
error_logs database table. Includes timestamps, source locations,
stack traces, and the command that was running.

Filter by severity level with --level (ERROR, WARN, INFO).
Control how many entries to show with --limit.

Examples:
  movie logs                  Show last 20 log entries
  movie logs --limit 50       Show last 50 entries
  movie logs --level ERROR    Show only errors
  movie logs --level WARN     Show only warnings
  movie logs --format json    Output as JSON`,
	Run: runMovieLogs,
}

func init() {
	movieLogsCmd.Flags().IntVarP(&logsLimit, "limit", "n", 20,
		"number of log entries to show")
	movieLogsCmd.Flags().StringVarP(&logsLevel, "level", "l", "",
		"filter by level: ERROR, WARN, or INFO")
	movieLogsCmd.Flags().StringVar(&logsFormat, "format", "default",
		"output format: default or json")
}

func runMovieLogs(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	entries, err := database.RecentErrorLogs(logsLimit)
	if err != nil {
		errlog.Error("Failed to read error logs: %v", err)
		return
	}

	// Filter by level if specified
	if logsLevel != "" {
		lvl := strings.ToUpper(logsLevel)
		var filtered []map[string]string
		for _, e := range entries {
			if e["level"] == lvl {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if len(entries) == 0 {
		if logsFormat == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("✅ No log entries found.")
		}
		return
	}

	if logsFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encErr := enc.Encode(entries); encErr != nil {
			errlog.Error("JSON encode error: %v", encErr)
		}
		return
	}

	// Default format
	levelFilter := ""
	if logsLevel != "" {
		levelFilter = fmt.Sprintf(" (level: %s)", strings.ToUpper(logsLevel))
	}
	fmt.Printf("📋 Error Logs — %d entries%s\n", len(entries), levelFilter)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, e := range entries {
		levelIcon := "ℹ️ "
		if e["level"] == "ERROR" {
			levelIcon = "❌"
		} else if e["level"] == "WARN" {
			levelIcon = "⚠️ "
		}

		fmt.Printf("\n  %s [%s] #%s  %s\n", levelIcon, e["level"], e["id"], e["timestamp"])
		fmt.Printf("     Source:   %s\n", e["source"])
		if e["function"] != "" {
			fmt.Printf("     Function: %s\n", e["function"])
		}
		if e["command"] != "" {
			fmt.Printf("     Command:  %s\n", e["command"])
		}
		if e["work_dir"] != "" {
			fmt.Printf("     WorkDir:  %s\n", e["work_dir"])
		}
		fmt.Printf("     Message:  %s\n", e["message"])
		if e["stack_trace"] != "" {
			fmt.Printf("     Stack:\n")
			for _, line := range strings.Split(e["stack_trace"], "\n") {
				if line != "" {
					fmt.Printf("       %s\n", line)
				}
			}
		}
	}
	fmt.Println()
}
