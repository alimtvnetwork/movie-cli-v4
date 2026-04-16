// movie_duplicates.go — detect duplicate media entries in the library.
// Supports detection by TMDb ID, filename, or file size.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var duplicatesByFlag string

var movieDuplicatesCmd = &cobra.Command{
	Use:   "duplicates",
	Short: "Detect duplicate media entries",
	Long: `Scan the library for duplicate entries and display them grouped.

Detection modes (use --by flag):
  tmdb      Match by TMDb ID (default) — same movie/show added twice
  filename  Match by original filename — same file scanned from different locations
  size      Match by file size — potential duplicates with different names

Examples:
  movie duplicates              # duplicates by TMDb ID
  movie duplicates --by tmdb    # same as above
  movie duplicates --by filename
  movie duplicates --by size`,
	Run: runMovieDuplicates,
}

func init() {
	movieDuplicatesCmd.Flags().StringVar(&duplicatesByFlag, "by", "tmdb",
		"detection method: tmdb, filename, size")
}

func runMovieDuplicates(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	groups, label, err := findDuplicateGroups(database)
	if err != nil {
		errlog.Error("Error finding duplicates: %v", err)
		return
	}

	if len(groups) == 0 {
		fmt.Printf("✅ No duplicates found (checked by %s)\n", label)
		return
	}

	printDuplicateGroups(groups, label)
}

func findDuplicateGroups(database *db.DB) ([]db.DuplicateGroup, string, error) {
	switch duplicatesByFlag {
	case "tmdb":
		groups, err := database.FindDuplicatesByTmdbID()
		return groups, "TMDb ID", err
	case "filename":
		groups, err := database.FindDuplicatesByFileName()
		return groups, "Filename", err
	case "size":
		groups, err := database.FindDuplicatesByFileSize()
		return groups, "File Size", err
	default:
		errlog.Error("Unknown detection method: %s (use tmdb, filename, or size)", duplicatesByFlag)
		return nil, "", nil
	}
}

func printDuplicateGroups(groups []db.DuplicateGroup, label string) {
	totalDupes := 0
	for _, g := range groups {
		totalDupes += len(g.Items)
	}

	fmt.Printf("🔍 Found %d duplicate groups (%d total entries) by %s\n", len(groups), totalDupes, label)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, g := range groups {
		fmt.Printf("\n  Group %d — %s: %s (%d entries)\n", i+1, label, g.Key, len(g.Items))
		for _, m := range g.Items {
			path := resolveDuplicatePath(m)
			fmt.Printf("    [ID %d] %s (%d) — %s\n", m.ID, m.Title, m.Year, path)
		}
	}

	fmt.Printf("\n💡 To remove a duplicate, delete its DB entry or file manually.\n")
}

func resolveDuplicatePath(m db.Media) string {
	if m.CurrentFilePath != "" {
		return m.CurrentFilePath
	}
	if m.OriginalFilePath != "" {
		return m.OriginalFilePath
	}
	return "(no file path)"
}
