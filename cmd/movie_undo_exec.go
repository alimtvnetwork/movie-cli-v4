// movie_undo_exec.go — execution helpers for undo command.
package cmd

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

// executeMoveUndo reverses a file move and updates DB state.
func executeMoveUndo(database *db.DB, m *db.MoveRecord) error {
	if _, err := os.Stat(m.ToPath); err != nil {
		if os.IsNotExist(err) {
			return apperror.New("file not found at %s — may have been moved manually", m.ToPath)
		}
		return apperror.Wrapf(err, "cannot access %s", m.ToPath)
	}

	destDir := m.FromPath[:strings.LastIndex(m.FromPath, string(os.PathSeparator))]
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return apperror.Wrapf(err, "cannot create directory %s", destDir)
	}

	if err := MoveFile(m.ToPath, m.FromPath); err != nil {
		return apperror.Wrap("move file back", err)
	}

	if err := database.MarkMoveReverted(m.ID); err != nil {
		errlog.Warn("Could not mark move %d as reverted: %v", m.ID, err)
	}

	if err := database.UpdateMediaPath(m.MediaID, m.FromPath); err != nil {
		errlog.Warn(fmt.Sprintf("Could not update media path (ID %d): %v", m.ID, err))
	}

	return nil
}

// executeActionUndo reverses an action_history entry based on its FileActionId.
func executeActionUndo(database *db.DB, a *db.ActionRecord) error {
	switch a.FileActionId {
	case db.FileActionScanAdd:
		return undoScanAdd(database, a)
	case db.FileActionScanRemove, db.FileActionDelete:
		return undoDelete(database, a)
	case db.FileActionRescanUpdate:
		return undoRescanUpdate(database, a)
	case db.FileActionPopout:
		// Handled via move_history; just mark reverted
	case db.FileActionRestore:
		return undoRestore(database, a)
	default:
		return apperror.New("unknown action type for undo: %s", a.FileActionId)
	}
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoScanAdd(database *db.DB, a *db.ActionRecord) error {
	if a.MediaId.Valid {
		if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
			return apperror.Wrapf(err, "undo scan_add (delete media %d)", a.MediaId.Int64)
		}
	}
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoDelete(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return apperror.New("no snapshot available for action %d — cannot restore", a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return apperror.Wrapf(err, "parse snapshot for action %d", a.ActionHistoryId)
	}
	newID, insertErr := database.InsertMedia(media)
	if insertErr != nil {
		return apperror.Wrap("re-insert media from snapshot", insertErr)
	}
	database.InsertActionSimple(db.ActionSimpleInput{
		FileAction: db.FileActionRestore, MediaID: newID, Snapshot: a.MediaSnapshot,
		Detail: fmt.Sprintf("Restored: %s (from undo of action %d)", media.Title, a.ActionHistoryId),
	})
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoRescanUpdate(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return apperror.New("no snapshot for action %d — cannot revert metadata", a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return apperror.Wrapf(err, "parse snapshot for action %d", a.ActionHistoryId)
	}
	if media.ID > 0 {
		if updateErr := database.UpdateMediaByID(media); updateErr != nil {
			return apperror.Wrapf(updateErr, "restore metadata for media %d", media.ID)
		}
	}
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoRestore(database *db.DB, a *db.ActionRecord) error {
	if a.MediaId.Valid {
		if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
			return apperror.Wrapf(err, "undo restore (delete media %d)", a.MediaId.Int64)
		}
	}
	return database.MarkActionReverted(a.ActionHistoryId)
}

func confirmUndo(scanner *bufio.Scanner) bool {
	fmt.Print("\n  Undo this? [y/N]: ")
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if answer != "y" && answer != "yes" {
		fmt.Println("❌ Undo canceled.")
		return false
	}
	return true
}

func printActionUndo(a *db.ActionRecord) {
	fmt.Printf("⏪ Last action (%s):\n", a.FileActionId)
	if a.Detail != "" {
		fmt.Printf("   %s\n", a.Detail)
	}
	if a.BatchId != "" {
		fmt.Printf("   Batch: %s\n", a.BatchId[:8])
	}
}
