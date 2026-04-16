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
		return processWalkEntry(WalkEntryInput{
			RootDir: rootDir, Path: path, Info: info, MaxDepth: maxDepth, Items: &items,
		})
	})

	return items
}

func processWalkEntry(input WalkEntryInput) error {
	rel, relErr := filepath.Rel(input.RootDir, input.Path)
	if relErr != nil {
		return nil
	}
	depth := strings.Count(rel, string(os.PathSeparator))

	if depth == 0 {
		return nil
	}
	if input.Info.IsDir() && depth >= input.MaxDepth {
		return filepath.SkipDir
	}
	if input.Info.IsDir() || !cleaner.IsVideoFile(input.Info.Name()) {
		return nil
	}

	result := cleaner.Clean(input.Info.Name())
	destName := buildDestName(input.Info.Name(), result)
	destPath := filepath.Join(input.RootDir, destName)

	parts := strings.SplitN(rel, string(os.PathSeparator), 2)
	*input.Items = append(*input.Items, popoutItem{
		srcPath:   input.Path,
		destPath:  destPath,
		cleanName: destName,
		result:    result,
		size:      input.Info.Size(),
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
		database.InsertActionSimple(db.ActionSimpleInput{
			FileAction: db.FileActionPopout, MediaID: mediaID,
			Detail: detail, BatchID: batchID,
		})
		success++
	}
	return
}

// trackPopoutMove records the popout in move_history and updates/creates media.
func trackPopoutMove(database *db.DB, item popoutItem, batchID string) int64 {
	mediaID := findPopoutMedia(database, item)

	if mediaID != 0 {
		if err := database.UpdateMediaPath(mediaID, item.destPath); err != nil {
			errlog.Error("DB update path error: %v", err)
		}
	}
	if mediaID == 0 {
		mediaID = insertPopoutMedia(database, item)
	}

	if mediaID > 0 {
		if err := database.InsertMoveHistory(db.MoveInput{
			MediaID: mediaID, FileActionID: int(db.FileActionPopout),
			FromPath: item.srcPath, ToPath: item.destPath,
			OrigName: filepath.Base(item.srcPath), NewName: item.cleanName,
		}); err != nil {
			errlog.Warn("DB move history error: %v", err)
		}
	}

	saveHistoryLog(HistoryLogInput{
		BasePath: database.BasePath, Title: item.result.CleanTitle,
		Year: item.result.Year, FromPath: item.srcPath, ToPath: item.destPath,
	})
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
