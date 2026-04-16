// movie_move.go — movie move
package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var moveAllFlag bool

var movieMoveCmd = &cobra.Command{
	Use:   "move [directory]",
	Short: "Browse a local directory and move a movie/TV show file",
	Long: `Browse a local directory for video files, select one, and move it
to a configured destination (Movies, TV Shows, Archive, or custom path).
The move is logged for undo support.

Use --all to move all video files at once. Movies go to movies_dir,
TV shows go to tv_dir (auto-detected from filename).

If no directory is given, you'll be prompted to choose one.`,
	Args: cobra.MaximumNArgs(1),
	Run:  runMovieMove,
}

func init() {
	movieMoveCmd.Flags().BoolVar(&moveAllFlag, "all", false, "Move all video files in the directory at once")
}

func runMovieMove(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error(msgDatabaseError, err)
		return
	}
	defer database.Close()

	scanner := bufio.NewScanner(os.Stdin)
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		errlog.Error("Cannot determine home directory: %v", homeErr)
		return
	}

	mc := MoveContext{Database: database, Scanner: scanner, Home: home}
	sourceDir := resolveSourceDir(args, mc)
	if sourceDir == "" {
		return
	}

	files, valid := validateAndListVideos(sourceDir)
	if !valid {
		return
	}

	mc.SourceDir = sourceDir
	mc.Files = files
	if moveAllFlag {
		runBatchMove(mc)
		return
	}
	runInteractiveMove(mc)
}

// validateAndListVideos checks the directory and returns its video files.
func validateAndListVideos(sourceDir string) ([]os.FileInfo, bool) {
	if !validateDirectory(sourceDir) {
		return nil, false
	}
	files, listErr := listVideoFiles(sourceDir)
	if listErr != nil {
		errlog.Error("%v", listErr)
		return nil, false
	}
	if len(files) == 0 {
		fmt.Printf("📭 No video files found in: %s\n", sourceDir)
		return nil, false
	}
	return files, true
}

func resolveSourceDir(args []string, mc MoveContext) string {
	if len(args) > 0 {
		return expandHome(args[0], mc.Home)
	}
	return promptSourceDirectory(mc.Scanner, mc.Database, mc.Home)
}
