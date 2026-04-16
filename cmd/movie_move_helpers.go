// movie_move_helpers.go — shared helpers for move/rename/undo operations
//
// -- Shared helpers exported from this file --
//
//	expandHome(path, home)                     — resolve ~ in paths
//	listVideoFiles(dir) ([]FileInfo, error)    — list video files in a directory
//	humanSize(bytes) string                    — format bytes as human-readable size
//	promptSourceDirectory(scanner, db, home)   — interactive source dir picker
//	promptDestination(scanner, db, home)       — interactive destination picker
//	MoveFile(src, dst) error                   — move with cross-device fallback
//	crossDeviceMove(src, dst) error            — copy+delete for cross-filesystem moves
//	saveHistoryLog(input HistoryLogInput)               — write move-log.json
//
// Consumers: movie_move.go, movie_rename.go, movie_undo.go, movie_stats.go
//
// Do NOT duplicate move/size/path logic elsewhere — use these helpers.
package cmd

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

// expandHome replaces ~ with actual home directory.
func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~") {
		return filepath.Join(home, path[1:])
	}
	return path
}

// listVideoFiles returns all video files in a directory.
// Returns an error if the directory cannot be read.
func listVideoFiles(dir string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, apperror.Wrapf(err, "cannot read directory %s", dir)
	}

	var files []os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !cleaner.IsVideoFile(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			errlog.Warn("Cannot stat %s: %v", entry.Name(), infoErr)
			continue
		}
		files = append(files, info)
	}
	return files, nil
}

// humanSize formats bytes into human-readable form.
// Delegates to db.HumanSize to avoid duplication.
func humanSize(bytes int64) string {
	return db.HumanSize(float64(bytes) / (1024 * 1024))
}

// promptSourceDirectory asks the user to pick a directory.
func promptSourceDirectory(scanner interface {
	Scan() bool
	Text() string
}, database *db.DB, home string) string {
	scanDir, cfgErr := database.GetConfig("ScanDir")
	if cfgErr != nil && cfgErr.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error (scan_dir): %v", cfgErr)
	}
	scanDir = expandHome(scanDir, home)

	fmt.Println("📂 Where are your video files?")
	fmt.Println()

	options, labels := buildDirOptions(scanDir, home)

	for i, label := range labels {
		fmt.Printf("  %d. %s\n", i+1, label)
	}
	fmt.Printf("  %d. Enter custom path\n", len(labels)+1)
	fmt.Println()
	fmt.Print("  Choose: ")

	return readDirChoice(scanner, options, home)
}

func buildDirOptions(scanDir, home string) ([]string, []string) {
	var options, labels []string

	if scanDir != "" {
		options = append(options, scanDir)
		labels = append(labels, fmt.Sprintf("Scan folder (%s)", scanDir))
	}

	downloads := filepath.Join(home, "Downloads")
	if info, err := os.Stat(downloads); err == nil && info.IsDir() {
		options = append(options, downloads)
		labels = append(labels, fmt.Sprintf("Downloads (%s)", downloads))
	}

	desktop := filepath.Join(home, "Desktop")
	if info, err := os.Stat(desktop); err == nil && info.IsDir() {
		options = append(options, desktop)
		labels = append(labels, fmt.Sprintf("Desktop (%s)", desktop))
	}
	return options, labels
}

func readDirChoice(scanner interface {
	Scan() bool
	Text() string
}, options []string, home string) string {
	if !scanner.Scan() {
		return ""
	}
	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(options)+1 {
		errlog.Error("Invalid choice")
		return ""
	}
	if choice <= len(options) {
		return options[choice-1]
	}

	fmt.Print("  Enter path: ")
	if !scanner.Scan() {
		return ""
	}
	return expandHome(strings.TrimSpace(scanner.Text()), home)
}

// promptDestination asks the user to choose a move destination.
func promptDestination(scanner interface {
	Scan() bool
	Text() string
}, database *db.DB, home string) string {
	moviesDir, tvDir, archiveDir := loadDestinationDirs(database, home)

	fmt.Println()
	fmt.Println("  📁 Move to:")
	fmt.Printf("  1. 🎬 Movies (%s)\n", moviesDir)
	fmt.Printf("  2. 📺 TV Shows (%s)\n", tvDir)
	fmt.Printf("  3. 📦 Archive (%s)\n", archiveDir)
	fmt.Println("  4. 📂 Custom path")
	fmt.Println()
	fmt.Print("  Choose [1-4]: ")

	if !scanner.Scan() {
		return ""
	}
	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1":
		return moviesDir
	case "2":
		return tvDir
	case "3":
		return archiveDir
	case "4":
		fmt.Print("  Enter path: ")
		if !scanner.Scan() {
			return ""
		}
		return expandHome(strings.TrimSpace(scanner.Text()), home)
	default:
		errlog.Error("Invalid choice")
		return ""
	}
}

func loadDestinationDirs(database *db.DB, home string) (string, string, string) {
	moviesDir := loadConfigDir(database, "MoviesDir", home)
	tvDir := loadConfigDir(database, "TvDir", home)
	archiveDir := loadConfigDir(database, "ArchiveDir", home)

	if moviesDir == "" {
		moviesDir = expandHome("~/Movies", home)
	}
	if tvDir == "" {
		tvDir = expandHome("~/TVShows", home)
	}
	if archiveDir == "" {
		archiveDir = expandHome("~/Archive", home)
	}
	return moviesDir, tvDir, archiveDir
}

func loadConfigDir(database *db.DB, key, home string) string {
	val, err := database.GetConfig(key)
	if err != nil && err.Error() != "sql: no rows in result set" {
		errlog.Warn("Config read error (%s): %v", key, err)
	}
	return expandHome(val, home)
}

// MoveFile moves a file from src to dst using os.Rename with cross-device fallback.
func MoveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil && isCrossDeviceError(err) {
		return crossDeviceMove(src, dst)
	}
	return err
}

// isCrossDeviceError checks whether the error is an EXDEV (cross-device link)
// error, which occurs when os.Rename is called across different filesystems
// (e.g., USB drives, network mounts, different partitions).
func isCrossDeviceError(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		var errno syscall.Errno
		if errors.As(linkErr.Err, &errno) {
			return errno == syscall.EXDEV
		}
	}
	return false
}

// crossDeviceMove copies the file from src to dst, preserves the original file
// permissions, and removes the source only after the destination is fully
// written and synced. This is the fallback when os.Rename fails with EXDEV.
func crossDeviceMove(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return apperror.Wrap("open source", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return apperror.Wrap("stat source", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return apperror.Wrap("create destination", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		os.Remove(dst)
		return apperror.Wrap("copy data", err)
	}

	if err := dstFile.Sync(); err != nil {
		dstFile.Close()
		os.Remove(dst)
		return apperror.Wrap("sync destination", err)
	}
	dstFile.Close()

	return os.Remove(src)
}

// saveHistoryLog writes a JSON move record to the history log.
// All errors are logged via errlog — never swallowed.
func saveHistoryLog(input HistoryLogInput) {
	historyDir := filepath.Join(input.BasePath, "json", "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		errlog.Warn("Cannot create history dir: %v", err)
		return
	}

	record := map[string]interface{}{
		"title":     input.Title,
		"year":      input.Year,
		"from_path": input.FromPath,
		"to_path":   input.ToPath,
		"moved_at":  time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		errlog.Warn("Cannot marshal history JSON: %v", err)
		return
	}

	filename := fmt.Sprintf("move-%s.json", time.Now().UTC().Format("20060102-150405"))
	historyPath := filepath.Join(historyDir, filename)
	if writeErr := os.WriteFile(historyPath, data, 0644); writeErr != nil {
		errlog.Warn("Cannot write history file: %v", writeErr)
	}
}
