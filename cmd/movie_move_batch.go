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
func runBatchMove(mc MoveContext) {
	moviesDir, tvDir := resolveMoveTargetDirs(mc.Database, mc.Home)
	moves := previewBatchMoves(BatchMovePreview{
		Files: mc.Files, SourceDir: mc.SourceDir, MoviesDir: moviesDir, TVDir: tvDir,
	})

	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n  Move all %d files? [y/N]: ", len(moves))

	if !mc.Scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(mc.Scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("  ❌ Batch move canceled.")
		return
	}

	executeBatchMoves(mc.Database, moves)
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

func previewBatchMoves(input BatchMovePreview) []moveItem {
	var moves []moveItem

	fmt.Printf("\n🎬 Batch move — %d video files in: %s\n\n", len(input.Files), input.SourceDir)
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, f := range input.Files {
		result := cleaner.Clean(f.Name())
		cleanName := cleaner.ToCleanFileName(result.CleanTitle, result.Year, result.Extension)

		destDir := input.MoviesDir
		typeIcon := db.TypeIcon(result.Type)
		if result.Type == string(db.MediaTypeTV) {
			destDir = input.TVDir
		}

		srcPath := filepath.Join(input.SourceDir, f.Name())
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

		trackMove(TrackMoveInput{
			Database: database, Result: moves[i].result, FileInfo: moves[i].fileInfo,
			SrcPath: moves[i].srcPath, DestPath: moves[i].destPath, CleanName: moves[i].cleanName,
		})
		success++
	}

	fmt.Println()
	if failed == 0 {
		fmt.Printf("  ✅ All %d files moved successfully!\n", success)
		return
	}
	fmt.Printf("  ⚠️  %d moved, %d failed\n", success, failed)
}

// runInteractiveMove is the original single-file interactive flow.
func runInteractiveMove(mc MoveContext) {
	printFileList(mc.Files, mc.SourceDir)

	selectedFile, _ := selectFile(mc.Scanner, mc.Files)
	if selectedFile == nil {
		return
	}
	selectedPath := filepath.Join(mc.SourceDir, selectedFile.Name())
		return
	}
	result := cleaner.Clean(selectedFile.Name())

	fmt.Printf("\n  Selected: %s\n", result.CleanTitle)
	if result.Year > 0 {
		fmt.Printf("  Year:     %d\n", result.Year)
	}
	fmt.Printf("  Type:     %s\n", result.Type)

	destDir := promptDestination(mc.Scanner, mc.Database, mc.Home)
	if destDir == "" {
		return
	}

	cleanName := cleaner.ToCleanFileName(result.CleanTitle, result.Year, result.Extension)
	destPath := filepath.Join(destDir, cleanName)

	if !confirmInteractiveMove(mc.Scanner, selectedPath, destPath) {
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

	trackMove(TrackMoveInput{
		Database: mc.Database, Result: result, FileInfo: selectedFile,
		SrcPath: selectedPath, DestPath: destPath, CleanName: cleanName,
	})

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

func selectFile(scanner *bufio.Scanner, files []os.FileInfo) (os.FileInfo, string) {
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
	return selected, ""
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
		if histErr := database.InsertMoveHistory(db.MoveInput{
			MediaID: mediaID, FileActionID: int(db.FileActionMove),
			FromPath: srcPath, ToPath: destPath,
			OrigName: fileInfo.Name(), NewName: cleanName,
		}); histErr != nil {
			errlog.Warn("DB history error: %v", histErr)
		}
	}

	saveHistoryLog(HistoryLogInput{
		BasePath: database.BasePath, Title: result.CleanTitle,
		Year: result.Year, FromPath: srcPath, ToPath: destPath,
	})
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

	if mediaID != 0 {
		if updateErr := database.UpdateMediaPath(mediaID, destPath); updateErr != nil {
			errlog.Error("DB update path error: %v", updateErr)
		}
		return mediaID
	}

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
	return mediaID
}
