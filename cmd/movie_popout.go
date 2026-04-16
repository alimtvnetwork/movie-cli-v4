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
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
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

func openDB() (*dbWrapper, error) {
	database, err := dbOpen()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return nil, err
	}
	return database, nil
}

func resolvePopoutDir(args []string, scanner *bufio.Scanner, database *dbWrapper) string {
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		errlog.Error("Cannot determine home directory: %v", homeErr)
		return ""
	}

	var rootDir string
	if len(args) > 0 {
		rootDir = expandHome(args[0], home)
	} else {
		rootDir = promptSourceDirectory(scanner, database.DB, home)
	}
	if rootDir == "" {
		return ""
	}

	info, statErr := os.Stat(rootDir)
	if statErr != nil {
		errlog.Error("Cannot access directory: %v", statErr)
		return ""
	}
	if !info.IsDir() {
		errlog.Error("Path is not a directory: %s", rootDir)
		return ""
	}
	return rootDir
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

// formatYearSuffix returns " (YYYY)" or "" if year is zero.
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
	if confirm != "y" && confirm != "yes" {
		fmt.Println("  ❌ Popout canceled.")
		return false
	}
	return true
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

// discoverNestedVideos walks the directory tree and finds video files that are
// NOT at the root level (i.e., inside at least one subfolder).
func discoverNestedVideos(rootDir string, maxDepth int) []popoutItem {
	var items []popoutItem

	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}

		// Calculate depth relative to root
		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}
		depth := strings.Count(rel, string(os.PathSeparator))

		// Skip root-level files (depth 0) — we only want nested ones
		if depth == 0 {
			return nil
		}

		// Respect max depth
		if info.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}

		// Only process video files
		if info.IsDir() || !cleaner.IsVideoFile(info.Name()) {
			return nil
		}

		result := cleaner.Clean(info.Name())

		var destName string
		if popoutNoRename {
			destName = info.Name()
		} else {
			destName = cleaner.ToCleanFileName(result.CleanTitle, result.Year, result.Extension)
		}

		destPath := filepath.Join(rootDir, destName)

		// Get the immediate subfolder name
		parts := strings.SplitN(rel, string(os.PathSeparator), 2)
		subDir := parts[0]

		items = append(items, popoutItem{
			srcPath:   path,
			destPath:  destPath,
			cleanName: destName,
			result:    result,
			size:      info.Size(),
			subDir:    subDir,
		})

		return nil
	})

	return items
}

// executePopout moves all discovered files and tracks each in the database.
func executePopout(database *db.DB, items []popoutItem, batchID string) (success, failed int) {
	for _, item := range items {
		// Check for destination conflict
		if _, err := os.Stat(item.destPath); err == nil {
			errlog.Warn("Skipped (already exists): %s", item.destPath)
			failed++
			continue
		}

		// Move the file
		if err := MoveFile(item.srcPath, item.destPath); err != nil {
			errlog.Error("Failed to move %s: %v", filepath.Base(item.srcPath), err)
			failed++
			continue
		}

		// Track in DB
		mediaID := trackPopoutMove(database, item, batchID)

		// Log to action_history
		detail := fmt.Sprintf("Popped out: %s from %s/", item.cleanName, item.subDir)
		database.InsertActionSimple(db.FileActionPopout, mediaID, "", detail, batchID)

		success++
	}
	return
}

// trackPopoutMove records the popout in move_history and updates/creates media.
func trackPopoutMove(database *db.DB, item popoutItem, batchID string) int64 {
	mediaID := findExistingMediaID(database, item)

	if mediaID == 0 {
		mediaID = insertPopoutMedia(database, item)
	} else {
		updatePopoutMediaPath(database, mediaID, item.destPath)
	}

	logPopoutMoveHistory(database, mediaID, item)
	saveHistoryLog(database.BasePath, item.result.CleanTitle, item.result.Year, item.srcPath, item.destPath)
	return mediaID
}

func findExistingMediaID(database *db.DB, item popoutItem) int64 {
	existing, searchErr := database.SearchMedia(item.result.CleanTitle)
	if searchErr != nil {
		errlog.Warn("DB search error: %v", searchErr)
		return 0
	}
	for i := range existing {
		if existing[i].CurrentFilePath == item.srcPath || existing[i].OriginalFilePath == item.srcPath {
			return existing[i].ID
		}
	}
	return 0
}

func insertPopoutMedia(database *db.DB, item popoutItem) int64 {
	m := &db.Media{
		Title:            item.result.CleanTitle,
		CleanTitle:       item.result.CleanTitle,
		Year:             item.result.Year,
		Type:             item.result.Type,
		OriginalFileName: filepath.Base(item.srcPath),
		OriginalFilePath: item.srcPath,
		CurrentFilePath:  item.destPath,
		FileExtension:    item.result.Extension,
		FileSize:         item.size,
	}
	mediaID, insertErr := database.InsertMedia(m)
	if insertErr != nil {
		errlog.Error("DB insert error: %v", insertErr)
	}
	return mediaID
}

func updatePopoutMediaPath(database *db.DB, mediaID int64, destPath string) {
	if err := database.UpdateMediaPath(mediaID, destPath); err != nil {
		errlog.Error("DB update path error: %v", err)
	}
}

func logPopoutMoveHistory(database *db.DB, mediaID int64, item popoutItem) {
	if mediaID <= 0 {
		return
	}
	if err := database.InsertMoveHistory(mediaID, int(db.FileActionPopout), item.srcPath, item.destPath,
		filepath.Base(item.srcPath), item.cleanName); err != nil {
		errlog.Warn("DB move history error: %v", err)
	}
}

// offerFolderCleanup lists source subfolders and offers removal options.
func offerFolderCleanup(scanner *bufio.Scanner, database *db.DB, rootDir string, items []popoutItem, batchID string) {
	// Collect unique subfolders that had files popped out
	subDirs := make(map[string]bool)
	for _, item := range items {
		subDirs[item.subDir] = true
	}

	var folders []popoutFolderInfo
	for dir := range subDirs {
		dirPath := filepath.Join(rootDir, dir)
		info, statErr := os.Stat(dirPath)
		if statErr != nil || !info.IsDir() {
			continue
		}

		var files []string
		var totalSize int64
		_ = filepath.Walk(dirPath, func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(dirPath, p)
			files = append(files, fmt.Sprintf("%s (%s)", rel, humanSize(fi.Size())))
			totalSize += fi.Size()
			return nil
		})

		folders = append(folders, popoutFolderInfo{
			name:      dir,
			path:      dirPath,
			files:     files,
			totalSize: totalSize,
		})
	}

	if len(folders) == 0 {
		return
	}

	fmt.Println("  📁 Source folders after popout:")
	fmt.Println()
	for i, f := range folders {
		if len(f.files) == 0 {
			fmt.Printf("  %d. %s/\n     └── (empty)\n", i+1, f.name)
		} else {
			fmt.Printf("  %d. %s/\n     └── %d files remaining (%s)\n",
				i+1, f.name, len(f.files), humanSize(f.totalSize))
		}
	}

	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println("    [a] Remove all listed folders")
	fmt.Println("    [s] Select folders to remove one by one")
	fmt.Println("    [n] Keep all folders")
	fmt.Println("    [l] List files in each folder before deciding")
	fmt.Print("\n  Choose [a/s/n/l]: ")

	if !scanner.Scan() {
		return
	}
	choice := strings.ToLower(strings.TrimSpace(scanner.Text()))

	switch choice {
	case "a":
		for _, f := range folders {
			removeFolder(database, f.path, f.name, batchID)
		}
	case "s":
		selectiveFolderRemoval(scanner, database, folders, batchID)
	case "l":
		listThenDecide(scanner, database, folders, batchID)
	case "n":
		fmt.Println("  📁 All folders kept.")
	default:
		fmt.Println("  📁 No folders removed.")
	}
}

func selectiveFolderRemoval(scanner *bufio.Scanner, database *db.DB, folders []popoutFolderInfo, batchID string) {
	for _, f := range folders {
		status := "empty"
		if len(f.files) > 0 {
			status = fmt.Sprintf("%d files (%s)", len(f.files), humanSize(f.totalSize))
		}
		fmt.Printf("\n  %s/ — %s\n", f.name, status)
		if len(f.files) > 0 {
			fmt.Println("    Files:")
			for _, file := range f.files {
				fmt.Printf("      - %s\n", file)
			}
		}
		fmt.Print("    Remove? [y/N]: ")
		if !scanner.Scan() {
			return
		}
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer == "y" || answer == "yes" {
			removeFolder(database, f.path, f.name, batchID)
		} else {
			fmt.Println("    Kept.")
		}
	}
}

func listThenDecide(scanner *bufio.Scanner, database *db.DB, folders []popoutFolderInfo, batchID string) {
	for _, f := range folders {
		fmt.Printf("\n  📁 %s/\n", f.name)
		if len(f.files) == 0 {
			fmt.Println("    (empty)")
		} else {
			for _, file := range f.files {
				fmt.Printf("    - %s\n", file)
			}
		}
		fmt.Print("    Remove? [y/N]: ")
		if !scanner.Scan() {
			return
		}
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer == "y" || answer == "yes" {
			removeFolder(database, f.path, f.name, batchID)
		} else {
			fmt.Println("    Kept.")
		}
	}
}

func removeFolder(database *db.DB, dirPath, dirName, batchID string) {
	if err := os.RemoveAll(dirPath); err != nil {
		errlog.Error("Failed to remove %s: %v", dirPath, err)
		return
	}
	fmt.Printf("    🗑  Removed: %s/\n", dirName)

	// Track folder deletion in action_history
	detail := fmt.Sprintf("Removed folder: %s/", dirName)
	snapshot := fmt.Sprintf(`{"folder_path":"%s"}`, dirPath)
	database.InsertActionSimple(db.FileActionDelete, 0, snapshot, detail, batchID)
}

// generateBatchID creates a simple random hex ID for grouping related actions.
func generateBatchID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
