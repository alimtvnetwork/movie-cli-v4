// movie_popout_discover.go — discovery and execution for popout command.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

// discoverNestedVideos walks the directory tree and finds video files that are
// NOT at the root level (i.e., inside at least one subfolder).
func discoverNestedVideos(rootDir string, maxDepth int) []popoutItem {
	var items []popoutItem

	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		return processWalkEntry(rootDir, path, info, maxDepth, &items)
	})

	return items
}

func processWalkEntry(rootDir, path string, info os.FileInfo, maxDepth int, items *[]popoutItem) error {
	rel, relErr := filepath.Rel(rootDir, path)
	if relErr != nil {
		return nil
	}
	depth := strings.Count(rel, string(os.PathSeparator))

	if depth == 0 {
		return nil
	}
	if info.IsDir() && depth >= maxDepth {
		return filepath.SkipDir
	}
	if info.IsDir() || !cleaner.IsVideoFile(info.Name()) {
		return nil
	}

	result := cleaner.Clean(info.Name())
	destName := buildDestName(info.Name(), result)
	destPath := filepath.Join(rootDir, destName)

	parts := strings.SplitN(rel, string(os.PathSeparator), 2)
	*items = append(*items, popoutItem{
		srcPath:   path,
		destPath:  destPath,
		cleanName: destName,
		result:    result,
		size:      info.Size(),
		subDir:    parts[0],
	})

	return nil
}

func buildDestName(origName string, result cleaner.Result) string {
	if popoutNoRename {
		return origName
	}
	return cleaner.ToCleanFileName(result.CleanTitle, result.Year, result.Extension)
}

// executePopout moves all discovered files and tracks each in the database.
func executePopout(database *db.DB, items []popoutItem, batchID string) (success, failed int) {
	for _, item := range items {
		if _, err := os.Stat(item.destPath); err == nil {
			errlog.Warn("Skipped (already exists): %s", item.destPath)
			failed++
			continue
		}

		if err := MoveFile(item.srcPath, item.destPath); err != nil {
			errlog.Error("Failed to move %s: %v", filepath.Base(item.srcPath), err)
			failed++
			continue
		}

		mediaID := trackPopoutMove(database, item, batchID)
		detail := fmt.Sprintf("Popped out: %s from %s/", item.cleanName, item.subDir)
		database.InsertActionSimple(db.FileActionPopout, mediaID, "", detail, batchID)
		success++
	}
	return
}

// trackPopoutMove records the popout in move_history and updates/creates media.
func trackPopoutMove(database *db.DB, item popoutItem, batchID string) int64 {
	mediaID := findPopoutMedia(database, item)

	if mediaID == 0 {
		mediaID = insertPopoutMedia(database, item)
	} else {
		if err := database.UpdateMediaPath(mediaID, item.destPath); err != nil {
			errlog.Error("DB update path error: %v", err)
		}
	}

	if mediaID > 0 {
		if err := database.InsertMoveHistory(mediaID, int(db.FileActionPopout), item.srcPath, item.destPath,
			filepath.Base(item.srcPath), item.cleanName); err != nil {
			errlog.Warn("DB move history error: %v", err)
		}
	}

	saveHistoryLog(database.BasePath, item.result.CleanTitle, item.result.Year, item.srcPath, item.destPath)
	return mediaID
}

func findPopoutMedia(database *db.DB, item popoutItem) int64 {
	existing, searchErr := database.SearchMedia(item.result.CleanTitle)
	if searchErr != nil {
		errlog.Warn("DB search error: %v", searchErr)
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
