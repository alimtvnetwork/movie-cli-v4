// movie_watch_sync.go — watchlist export/import for backup and sharing.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var watchExportOutput string
var watchImportInput string

var watchExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export watchlist as JSON",
	Long: `Export your watchlist to a JSON file for backup or sharing.

Examples:
  movie watch export                          # Export to default path
  movie watch export -o ~/watchlist.json      # Custom output`,
	Run: runWatchExport,
}

var watchImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import watchlist from JSON",
	Long: `Import a watchlist from a JSON file. Existing entries are skipped.

Examples:
  movie watch import ~/watchlist.json`,
	Args: cobra.ExactArgs(1),
	Run:  runWatchImport,
}

func init() {
	watchExportCmd.Flags().StringVarP(&watchExportOutput, "output", "o", "",
		"Output file path (default: ./data/json/export/watchlist.json)")

	movieWatchCmd.AddCommand(watchExportCmd, watchImportCmd)
}

// watchlistJSON is the export/import format.
type watchlistJSON struct {
	ExportedAt string           `json:"exported_at"`
	Count      int              `json:"count"`
	Entries    []watchEntryJSON `json:"entries"`
}

type watchEntryJSON struct {
	TmdbID    int    `json:"tmdb_id"`
	Title     string `json:"title"`
	Year      int    `json:"year"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	AddedAt   string `json:"added_at"`
	WatchedAt string `json:"watched_at,omitempty"`
}

func runWatchExport(cmd *cobra.Command, args []string) {
	database, dbErr := db.Open()
	if dbErr != nil {
		errlog.Error(msgDatabaseError, dbErr)
		return
	}
	defer database.Close()

	entries, err := database.ListWatchlist("")
	if err != nil {
		errlog.Error("Failed to read watchlist: %v", err)
		return
	}
	if len(entries) == 0 {
		fmt.Println("📋 Watchlist is empty — nothing to export.")
		return
	}

	out := buildWatchlistJSON(entries)

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		errlog.Error("JSON encoding error: %v", err)
		return
	}

	outPath := resolveWatchExportPath()
	if writeErr := writeWatchExportFile(outPath, data); writeErr != nil {
		errlog.Error("%v", writeErr)
		return
	}

	fmt.Printf("✅ Exported %d watchlist entries → %s\n", len(entries), outPath)
}

func buildWatchlistJSON(entries []db.WatchlistEntry) watchlistJSON {
	out := watchlistJSON{
		ExportedAt: db.NowUTC(),
		Count:      len(entries),
	}
	for _, e := range entries {
		entry := watchEntryJSON{
			TmdbID: e.TmdbID, Title: e.Title, Year: e.Year,
			Type: e.Type, Status: e.Status, AddedAt: e.AddedAt,
		}
		if e.WatchedAt.Valid {
			entry.WatchedAt = e.WatchedAt.String
		}
		out.Entries = append(out.Entries, entry)
	}
	return out
}

func resolveWatchExportPath() string {
	if watchExportOutput != "" {
		return watchExportOutput
	}
	return filepath.Join(".", "data", "json", "export", "watchlist.json")
}

func writeWatchExportFile(outPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return apperror.Wrapf(err, "cannot create directory %s", filepath.Dir(outPath))
	}
	return os.WriteFile(outPath, data, 0644)
}

func runWatchImport(cmd *cobra.Command, args []string) {
	database, dbErr := db.Open()
	if dbErr != nil {
		errlog.Error(msgDatabaseError, dbErr)
		return
	}
	defer database.Close()

	data, err := os.ReadFile(args[0])
	if err != nil {
		errlog.Error("Cannot read file: %v", err)
		return
	}

	var input watchlistJSON
	if err := json.Unmarshal(data, &input); err != nil {
		errlog.Error("Invalid JSON: %v", err)
		return
	}

	added := 0
	skipped := 0

	for _, e := range input.Entries {
		existing, _ := database.GetWatchlistByTmdbID(e.TmdbID)
		if existing != nil {
			skipped++
			continue
		}

		if err := database.AddToWatchlist(db.WatchlistInput{
			TmdbID: e.TmdbID, Title: e.Title, Year: e.Year, MediaType: e.Type,
		}); err != nil {
			errlog.Warn("Import error for '%s': %v", e.Title, err)
			continue
		}

		if e.Status == string(db.WatchStatusWatched) {
			database.MarkWatched(e.TmdbID)
		}
		added++
	}

	fmt.Printf("✅ Imported: %d added, %d skipped (already exist)\n", added, skipped)
}
