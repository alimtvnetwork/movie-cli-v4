// movie_move_batch.go — batch and interactive move flows.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

// moveItem groups data for a single batch move operation.
type moveItem struct {
	srcPath   string
	destPath  string
	destDir   string
	cleanName string
	result    cleaner.Result
	fileInfo  os.FileInfo
}

// runBatchMove moves all video files at once, auto-routing by type.
func runBatchMove(database *db.DB, scanner *bufio.Scanner, sourceDir string, files []os.FileInfo, home string) {
	moviesDir, tvDir := resolveMoveTargetDirs(database, home)
	moves := previewBatchMoves(files, sourceDir, moviesDir, tvDir)

	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n  Move all %d files? [y/N]: ", len(moves))

	if !scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("  ❌ Batch move canceled.")
		return
	}

	executeBatchMoves(database, moves)
}

func resolveMoveTargetDirs(database *db.DB, home string) (string, string) {
	moviesDir, cfgErr := database.GetConfig("MoviesDir")
	if cfgErr != nil && cfgErr.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error (movies_dir): %v", cfgErr)
	}
	tvDir, cfgErr := database.GetConfig("TvDir")
	if cfgErr != nil && cfgErr.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error (tv_dir): %v", cfgErr)
	}
	moviesDir = expandHome(moviesDir, home)
	tvDir = expandHome(tvDir, home)

	if moviesDir == "" {
		moviesDir = expandHome("~/Movies", home)
	}
	if tvDir == "" {
		tvDir = expandHome("~/TVShows", home)
	}
	return moviesDir, tvDir
}

func previewBatchMoves(files []os.FileInfo, sourceDir, moviesDir, tvDir string) []moveItem {
	var moves []moveItem

	fmt.Printf("\n🎬 Batch move — %d video files in: %s\n\n", len(files), sourceDir)
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, f := range files {
		result := cleaner.Clean(f.Name())
		cleanName := cleaner.ToCleanFileName(result.CleanTitle, result.Year, result.Extension)

		destDir := moviesDir
		typeIcon := db.TypeIcon(result.Type)
		if result.Type == string(db.MediaTypeTV) {
			destDir = tvDir
		}

		srcPath := filepath.Join(sourceDir, f.Name())
		destPath := filepath.Join(destDir, cleanName)

		yearStr := ""
		if result.Year > 0 {
			yearStr = fmt.Sprintf(" (%d)", result.Year)
		}

		fmt.Printf("  %s %s%s  [%s]\n", typeIcon, result.CleanTitle, yearStr, humanSize(f.Size()))
		fmt.Printf("     → %s\n", destPath)

		moves = append(moves, moveItem{
			srcPath:   srcPath,
			destPath:  destPath,
			destDir:   destDir,
			cleanName: cleanName,
			result:    result,
			fileInfo:  f,
		})
	}
	return moves
}

func executeBatchMoves(database *db.DB, moves []moveItem) {
	success := 0
	failed := 0

	for i := range moves {
		if mkdirErr := os.MkdirAll(moves[i].destDir, 0755); mkdirErr != nil {
			errlog.Error("Cannot create dir %s: %v", moves[i].destDir, mkdirErr)
			failed++
			continue
		}

		if moveErr := MoveFile(moves[i].srcPath, moves[i].destPath); moveErr != nil {
			errlog.Error("Failed to move %s: %v", moves[i].fileInfo.Name(), moveErr)
			failed++
			continue
		}

		trackMove(database, moves[i].result, moves[i].fileInfo, moves[i].srcPath, moves[i].destPath, moves[i].cleanName)
		success++
	}

	fmt.Println()
	if failed == 0 {
		fmt.Printf("  ✅ All %d files moved successfully!\n", success)
	} else {
		fmt.Printf("  ⚠️  %d moved, %d failed\n", success, failed)
	}
}

// runInteractiveMove is the original single-file interactive flow.
func runInteractiveMove(database *db.DB, scanner *bufio.Scanner, sourceDir string, files []os.FileInfo, home string) {
	printFileList(files, sourceDir)

	selectedFile, selectedPath := selectFile(scanner, files, sourceDir)
	if selectedFile == nil {
		return
	}
	result := cleaner.Clean(selectedFile.Name())

	fmt.Printf("\n  Selected: %s\n", result.CleanTitle)
	if result.Year > 0 {
		fmt.Printf("  Year:     %d\n", result.Year)
	}
	fmt.Printf("  Type:     %s\n", result.Type)

	destDir := promptDestination(scanner, database, home)
	if destDir == "" {
		return
	}

	cleanName := cleaner.ToCleanFileName(result.CleanTitle, result.Year, result.Extension)
	destPath := filepath.Join(destDir, cleanName)

	if !confirmInteractiveMove(scanner, selectedPath, destPath) {
		return
	}

	if mkdirErr := os.MkdirAll(destDir, 0755); mkdirErr != nil {
		errlog.Error("Cannot create directory: %v", mkdirErr)
		return
	}

	if moveErr := MoveFile(selectedPath, destPath); moveErr != nil {
		errlog.Error("Move failed: %v", moveErr)
		return
	}

	trackMove(database, result, selectedFile, selectedPath, destPath, cleanName)

	fmt.Println()
	fmt.Println("  ✅ Moved successfully!")
	fmt.Printf("     %s\n", selectedPath)
	fmt.Printf("     → %s\n", destPath)
}

func printFileList(files []os.FileInfo, sourceDir string) {
	fmt.Printf("\n🎬 Video files in: %s\n\n", sourceDir)
	for i, f := range files {
		result := cleaner.Clean(f.Name())
		typeIcon := db.TypeIcon(result.Type)
		yearStr := ""
		if result.Year > 0 {
			yearStr = fmt.Sprintf("(%d)", result.Year)
		}
		fmt.Printf("  %2d. %s %s %s  [%s]\n", i+1, typeIcon, result.CleanTitle, yearStr, humanSize(f.Size()))
	}
}

func selectFile(scanner *bufio.Scanner, files []os.FileInfo, sourceDir string) (os.FileInfo, string) {
	fmt.Println()
	fmt.Print("  Select file [number]: ")
	if !scanner.Scan() {
		return nil, ""
	}
	choice, parseErr := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if parseErr != nil || choice < 1 || choice > len(files) {
		errlog.Error("Invalid selection")
		return nil, ""
	}
	selected := files[choice-1]
	return selected, filepath.Join(sourceDir, selected.Name())
}

func confirmInteractiveMove(scanner *bufio.Scanner, srcPath, destPath string) bool {
	fmt.Println()
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  📄 From: %s\n", srcPath)
	fmt.Printf("  📁 To:   %s\n", destPath)
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Print("  Are you sure? [y/N]: ")

	if !scanner.Scan() {
		return false
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return confirm == "y" || confirm == "yes"
}

// trackMove records a move in the database and JSON history log.
func trackMove(database *db.DB, result cleaner.Result, fileInfo os.FileInfo, srcPath, destPath, cleanName string) {
	mediaID := findOrCreateMoveMedia(database, result, fileInfo, srcPath, destPath)

	if mediaID > 0 {
		if histErr := database.InsertMoveHistory(mediaID, int(db.FileActionMove), srcPath, destPath,
			fileInfo.Name(), cleanName); histErr != nil {
			errlog.Warn("DB history error: %v", histErr)
		}
	}

	saveHistoryLog(database.BasePath, result.CleanTitle, result.Year, srcPath, destPath)
}

func findOrCreateMoveMedia(database *db.DB, result cleaner.Result, fileInfo os.FileInfo, srcPath, destPath string) int64 {
	var mediaID int64
	existing, searchErr := database.SearchMedia(result.CleanTitle)
	if searchErr != nil {
		errlog.Warn("DB search error: %v", searchErr)
	}
	for i := range existing {
		if existing[i].CurrentFilePath == srcPath || existing[i].OriginalFilePath == srcPath {
			mediaID = existing[i].ID
			break
		}
	}

	if mediaID == 0 {
		m := &db.Media{
			Title:            result.CleanTitle,
			CleanTitle:       result.CleanTitle,
			Year:             result.Year,
			Type:             result.Type,
			OriginalFileName: fileInfo.Name(),
			OriginalFilePath: srcPath,
			CurrentFilePath:  destPath,
			FileExtension:    result.Extension,
			FileSize:         fileInfo.Size(),
		}
		var insertErr error
		mediaID, insertErr = database.InsertMedia(m)
		if insertErr != nil {
			errlog.Error("DB insert error: %v", insertErr)
		}
	} else {
		if updateErr := database.UpdateMediaPath(mediaID, destPath); updateErr != nil {
			errlog.Error("DB update path error: %v", updateErr)
		}
	}
	return mediaID
}
