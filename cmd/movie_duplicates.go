// movie_duplicates.go — detect duplicate media entries in the library.
// Supports detection by TMDb ID, filename, or file size.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
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

	var groups []db.DuplicateGroup
	var label string

	switch duplicatesByFlag {
	case "tmdb":
		label = "TMDb ID"
		groups, err = database.FindDuplicatesByTmdbID()
	case "filename":
		label = "Filename"
		groups, err = database.FindDuplicatesByFileName()
	case "size":
		label = "File Size"
		groups, err = database.FindDuplicatesByFileSize()
	default:
		errlog.Error("Unknown detection method: %s (use tmdb, filename, or size)", duplicatesByFlag)
		return
	}

	if err != nil {
		errlog.Error("Error finding duplicates: %v", err)
		return
	}

	if len(groups) == 0 {
		fmt.Printf("✅ No duplicates found (checked by %s)\n", label)
		return
	}

	totalDupes := 0
	for _, g := range groups {
		totalDupes += len(g.Items)
	}

	fmt.Printf("🔍 Found %d duplicate groups (%d total entries) by %s\n", len(groups), totalDupes, label)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, g := range groups {
		fmt.Printf("\n  Group %d — %s: %s (%d entries)\n", i+1, label, g.Key, len(g.Items))
		for _, m := range g.Items {
			path := m.CurrentFilePath
			if path == "" {
				path = m.OriginalFilePath
			}
			if path == "" {
				path = "(no file path)"
			}
			fmt.Printf("    [ID %d] %s (%d) — %s\n", m.ID, m.Title, m.Year, path)
		}
	}

	fmt.Printf("\n💡 To remove a duplicate, delete its DB entry or file manually.\n")
}
