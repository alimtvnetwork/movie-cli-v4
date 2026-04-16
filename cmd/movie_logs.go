// movie_logs.go — movie logs: display recent error logs from the database
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
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

	entries = filterLogEntries(entries)

	if len(entries) == 0 {
		printEmptyLogs()
		return
	}

	if logsFormat == "json" {
		printLogsJSON(entries)
		return
	}
	printLogsDefault(entries)
}

func filterLogEntries(entries []map[string]string) []map[string]string {
	if logsLevel == "" {
		return entries
	}
	lvl := strings.ToUpper(logsLevel)
	var filtered []map[string]string
	for _, e := range entries {
		if e["level"] == lvl {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func printEmptyLogs() {
	if logsFormat == "json" {
		fmt.Println("[]")
		return
	}
	fmt.Println("✅ No log entries found.")
}

func printLogsJSON(entries []map[string]string) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(entries); encErr != nil {
		errlog.Error("JSON encode error: %v", encErr)
	}
}

func printLogsDefault(entries []map[string]string) {
	levelFilter := ""
	if logsLevel != "" {
		levelFilter = fmt.Sprintf(" (level: %s)", strings.ToUpper(logsLevel))
	}
	fmt.Printf("📋 Error Logs — %d entries%s\n", len(entries), levelFilter)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, e := range entries {
		printLogEntry(e)
	}
	fmt.Println()
}

func printLogEntry(e map[string]string) {
	levelIcon := logLevelIcon(e["level"])
	fmt.Printf("\n  %s [%s] #%s  %s\n", levelIcon, e["level"], e["id"], e["timestamp"])
	fmt.Printf("     Source:   %s\n", e["source"])
	printOptionalField("     Function: %s\n", e["function"])
	printOptionalField("     Command:  %s\n", e["command"])
	printOptionalField("     WorkDir:  %s\n", e["work_dir"])
	fmt.Printf("     Message:  %s\n", e["message"])
	printStackTrace(e["stack_trace"])
}

func logLevelIcon(level string) string {
	switch level {
	case "ERROR":
		return "❌"
	case "WARN":
		return "⚠️ "
	default:
		return "ℹ️ "
	}
}

func printOptionalField(format, value string) {
	if value != "" {
		fmt.Printf(format, value)
	}
}

func printStackTrace(trace string) {
	if trace == "" {
		return
	}
	fmt.Printf("     Stack:\n")
	for _, line := range strings.Split(trace, "\n") {
		if line != "" {
			fmt.Printf("       %s\n", line)
		}
	}
}
