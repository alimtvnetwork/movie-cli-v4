// movie_ls.go — movie ls
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var lsFormat string

var movieLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List scanned movies and TV shows from your library",
	Long: `Lists scan-indexed movies and TV shows (items with a known file path).
Only items added via 'movie scan' are shown.
Press N for next page, P for previous, Q to quit.

Use --format json to output all items as JSON to stdout for piping.
Use --format table to output all items as a formatted table (no pager).`,
	Run: runMovieLs,
}

func init() {
	movieLsCmd.Flags().StringVar(&lsFormat, "format", "default",
		"output format: default, json, or table")
}

// lsJSONItem represents a single media item in JSON output.
type lsJSONItem struct {
	ID         int64   `json:"id"`
	Title      string  `json:"title"`
	CleanTitle string  `json:"clean_title"`
	Year       int     `json:"year,omitempty"`
	Type       string  `json:"type"`
	TmdbID     int     `json:"tmdb_id,omitempty"`
	ImdbID     string  `json:"imdb_id,omitempty"`
	TmdbRating float64 `json:"tmdb_rating,omitempty"`
	ImdbRating float64 `json:"imdb_rating,omitempty"`
	Popularity float64 `json:"popularity,omitempty"`
	Genre      string  `json:"genre,omitempty"`
	Director   string  `json:"director,omitempty"`
	Runtime    int     `json:"runtime,omitempty"`
	Language   string  `json:"language,omitempty"`
	FilePath   string  `json:"file_path,omitempty"`
	FileSize   int64   `json:"file_size,omitempty"`
}

func runMovieLs(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	switch lsFormat {
	case string(db.OutputFormatJSON):
		runMovieLsJSON(database)
	case string(db.OutputFormatTable):
		runMovieLsTable(database)
	default:
		runMovieLsInteractive(database)
	}
}

func runMovieLsJSON(database *db.DB) {
	allMedia, err := database.ListMedia(0, 100000)
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}

	items := make([]lsJSONItem, len(allMedia))
	for i := range allMedia {
		items[i] = toLsJSONItem(&allMedia[i])
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(items); encErr != nil {
		errlog.Error("JSON encode error: %v", encErr)
	}
}

func toLsJSONItem(m *db.Media) lsJSONItem {
	return lsJSONItem{
		ID: m.ID, Title: m.Title, CleanTitle: m.CleanTitle,
		Year: m.Year, Type: m.Type, TmdbID: m.TmdbID, ImdbID: m.ImdbID,
		TmdbRating: m.TmdbRating, ImdbRating: m.ImdbRating, Popularity: m.Popularity,
		Genre: m.Genre, Director: m.Director, Runtime: m.Runtime,
		Language: m.Language, FilePath: m.CurrentFilePath, FileSize: m.FileSize,
	}
}

func runMovieLsInteractive(database *db.DB) {
	pageSize := resolvePageSize(database)

	total, countErr := database.CountMedia("")
	if countErr != nil {
		errlog.Error("Database error: %v", countErr)
		return
	}
	if total == 0 {
		fmt.Println("📭 No media found. Run 'movie scan <folder>' first.")
		return
	}

	offset := 0
	scanner := bufio.NewScanner(os.Stdin)

	for {
		media, listErr := database.ListMedia(offset, pageSize)
		if listErr != nil {
			errlog.Error("Error: %v", listErr)
			return
		}

		pg := LsPage{Offset: offset, PageSize: pageSize, Total: total}
		printLsPage(database, media, pg)

		if !scanner.Scan() {
			break
		}
		offset = handleLsInput(scanner.Text(), pg, database)
		if offset < 0 {
			return
		}
	}
}

func resolvePageSize(database *db.DB) int {
	pageSizeStr, cfgErr := database.GetConfig("PageSize")
	if cfgErr != nil && cfgErr.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error (page_size): %v", cfgErr)
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 20
	}
	return pageSize
}

func printLsPage(database *db.DB, media []db.Media, pg LsPage) {
	fmt.Print("\033[H\033[2J")
	page := (pg.Offset / pg.PageSize) + 1
	totalPages := (pg.Total + pg.PageSize - 1) / pg.PageSize
	fmt.Printf("🎬 Your Library — Page %d/%d (%d total)\n", page, totalPages, pg.Total)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if page == 1 {
		printScanFolders(database)
	}

	for i := range media {
		printLsMediaRow(pg.Offset+i+1, &media[i])
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Print("  [N] Next  [P] Previous  [Q] Quit  [1-9] View details → ")
}

func printScanFolders(database *db.DB) {
	scanFolders, _ := database.ListDistinctScanFolders()
	if len(scanFolders) == 0 {
		return
	}
	fmt.Print("  📂 Scanned: ")
	for i, f := range scanFolders {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(f)
		if i >= 2 && len(scanFolders) > 3 {
			fmt.Printf(" (+%d more)", len(scanFolders)-3)
			break
		}
	}
	fmt.Println()
	fmt.Println()
}

func printLsMediaRow(num int, m *db.Media) {
	yearStr := ""
	if m.Year > 0 {
		yearStr = fmt.Sprintf("(%d)", m.Year)
	}
	rating := bestRating(m)
	typeIcon := db.TypeIcon(m.Type)
	fmt.Printf("  %3d. %-40s %-6s  ⭐ %-4s  %s %s\n",
		num, m.CleanTitle, yearStr, rating, typeIcon, capitalize(m.Type))
}

func bestRating(m *db.Media) string {
	if m.TmdbRating > 0 {
		return fmt.Sprintf("%.1f", m.TmdbRating)
	}
	if m.ImdbRating > 0 {
		return fmt.Sprintf("%.1f", m.ImdbRating)
	}
	return "N/A"
}

func handleLsInput(input string, pg LsPage, database *db.DB) int {
	switch {
	case input == "n" || input == "N":
		if pg.Offset+pg.PageSize < pg.Total {
			return pg.Offset + pg.PageSize
		}
		fmt.Println("  ⚠️  Already on last page")
		return pg.Offset
	case input == "p" || input == "P":
		if pg.Offset-pg.PageSize >= 0 {
			return pg.Offset - pg.PageSize
		}
		fmt.Println("  ⚠️  Already on first page")
		return pg.Offset
	case input == "q" || input == "Q":
		fmt.Println("👋 Bye!")
		return -1
	default:
		if num, parseErr := strconv.Atoi(input); parseErr == nil && num > 0 && num <= total {
			showMediaDetail(database, int64(num))
			fmt.Print("\nPress Enter to continue...")
		}
		return offset
	}
}
