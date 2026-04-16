// movie_db.go — movie db: print resolved database path for debugging
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
)

var movieDBCmd = &cobra.Command{
	Use:   "db",
	Short: "Show the resolved database path and status",
	Long:  `Prints the full resolved path to the SQLite database and data directory. Useful for debugging data location issues.`,
	Run:   runMovieDB,
}

func runMovieDB(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	fmt.Println()
	fmt.Println("📂 Movie CLI — Data Location")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Data directory:  %s\n", database.BasePath)
	fmt.Printf("  Database file:   %s/movie.db\n", database.BasePath)
	fmt.Printf("  Thumbnails:      %s/thumbnails/\n", database.BasePath)
	fmt.Printf("  JSON metadata:   %s/json/\n", database.BasePath)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Show record counts
	total, countErr := database.CountMedia("")
	if countErr == nil {
		movies, _ := database.CountMedia(string(db.MediaTypeMovie))
		tv, _ := database.CountMedia(string(db.MediaTypeTV))
		fmt.Printf("\n  Records: %d total (%d movies, %d TV shows)\n", total, movies, tv)
	}

	fmt.Println()
}
