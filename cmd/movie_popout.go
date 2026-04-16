// movie_popout.go — movie popout: extract nested video files to root directory.
//
// Discovers video files inside subfolders of a target directory and moves
// them up to the root level with clean filenames. Each move is tracked in
// move_history and action_history for full undo support.
//
// Flags:
//
//	--dry-run      Preview without moving
//	--no-rename    Keep original filename
//	--depth N      Max subfolder depth (default 3)
package cmd

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var (
	popoutDryRun   bool
	popoutNoRename bool
	popoutDepth    int
)

var moviePopoutCmd = &cobra.Command{
	Use:   "popout [directory]",
	Short: "Extract video files from subfolders to root directory",
	Long: `Finds video files nested inside subfolders and moves them up to the
parent directory with clean filenames. Useful for downloaded movies
that come wrapped in folders with extras, samples, and subtitles.

Example:
  movie popout ~/Downloads

All moves are tracked for undo support (movie undo --batch).`,
	Args: cobra.MaximumNArgs(1),
	Run:  runMoviePopout,
}

func init() {
	moviePopoutCmd.Flags().BoolVar(&popoutDryRun, "dry-run", false, "Preview only, no file moves")
	moviePopoutCmd.Flags().BoolVar(&popoutNoRename, "no-rename", false, "Keep original filename")
	moviePopoutCmd.Flags().IntVar(&popoutDepth, "depth", 3, "Max subfolder depth to search")
}

// popoutItem represents a video file discovered in a subfolder.
type popoutItem struct {
	srcPath   string
	destPath  string
	cleanName string
	result    cleaner.Result
	size      int64
	subDir    string
}

// popoutFolderInfo holds info about a subfolder for the cleanup phase.
type popoutFolderInfo struct {
	name      string
	path      string
	files     []string
	totalSize int64
}

func runMoviePopout(cmd *cobra.Command, args []string) {
	database, openErr := db.Open()
	if openErr != nil {
		errlog.Error("Database error: %v", openErr)
		return
	}
	defer database.Close()

	scanner := bufio.NewScanner(os.Stdin)
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		errlog.Error("Cannot determine home directory: %v", homeErr)
		return
	}

	var rootDir string
	if len(args) > 0 {
		rootDir = expandHome(args[0], home)
	} else {
		rootDir = promptSourceDirectory(scanner, database, home)
	}
	if rootDir == "" {
		return
	}

	info, statErr := os.Stat(rootDir)
	if statErr != nil {
		errlog.Error("Cannot access directory: %v", statErr)
		return
	}
	if !info.IsDir() {
		errlog.Error("Path is not a directory: %s", rootDir)
		return
	}

	items := discoverNestedVideos(rootDir, popoutDepth)
	if len(items) == 0 {
		fmt.Printf("📭 No nested video files found in: %s\n", rootDir)
		return
	}

	printPopoutPreview(items)

	if popoutDryRun {
		fmt.Println("\n  (dry-run mode — no files moved)")
		return
	}

	if !confirmPopout(scanner, len(items)) {
		return
	}

	batchID := generateBatchID()
	success, failed := executePopout(database, items, batchID)
	printPopoutResult(success, failed, batchID)

	if success > 0 {
		fmt.Println()
		offerFolderCleanup(scanner, database, rootDir, items, batchID)
	}
}

func printPopoutPreview(items []popoutItem) {
	fmt.Printf("\n🎬 Movie Popout — %d files found in subfolders\n\n", len(items))
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for i, item := range items {
		yearStr := formatYearSuffix(item.result.Year)
		fmt.Printf("\n  %d. %s%s  [%s]\n", i+1, item.result.CleanTitle, yearStr, humanSize(item.size))
		fmt.Printf("     From: %s\n", item.srcPath)
		fmt.Printf("     To:   %s\n", item.destPath)
	}
	fmt.Println("\n  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func formatYearSuffix(year int) string {
	if year <= 0 {
		return ""
	}
	return fmt.Sprintf(" (%d)", year)
}

func confirmPopout(scanner *bufio.Scanner, count int) bool {
	fmt.Printf("\n  Pop out all %d files? [y/N]: ", count)
	if !scanner.Scan() {
		return false
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return confirm == "y" || confirm == "yes"
}

func printPopoutResult(success, failed int, batchID string) {
	fmt.Println()
	if failed == 0 {
		fmt.Printf("  ✅ All %d files popped out successfully!\n", success)
	} else {
		fmt.Printf("  ⚠️  %d moved, %d failed\n", success, failed)
	}
	fmt.Printf("  📋 Batch: %s\n", batchID[:8])
}

// generateBatchID creates a simple random hex ID for grouping related actions.
func generateBatchID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
