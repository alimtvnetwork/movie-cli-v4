// movie_undo_exec.go — execution helpers for undo command.
package cmd

import (
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
			return fmt.Errorf("file not found at %s — may have been moved manually", m.ToPath)
		}
		return fmt.Errorf("cannot access %s: %w", m.ToPath, err)
	}

	destDir := m.FromPath[:strings.LastIndex(m.FromPath, string(os.PathSeparator))]
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", destDir, err)
	}

	if err := MoveFile(m.ToPath, m.FromPath); err != nil {
		return fmt.Errorf("move file back: %w", err)
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
		return fmt.Errorf("unknown action type for undo: %s", a.FileActionId)
	}
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoScanAdd(database *db.DB, a *db.ActionRecord) error {
	if a.MediaId.Valid {
		if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
			return fmt.Errorf("undo scan_add (delete media %d): %w", a.MediaId.Int64, err)
		}
	}
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoDelete(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return fmt.Errorf("no snapshot available for action %d — cannot restore", a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return fmt.Errorf("parse snapshot for action %d: %w", a.ActionHistoryId, err)
	}
	newID, insertErr := database.InsertMedia(media)
	if insertErr != nil {
		return fmt.Errorf("re-insert media from snapshot: %w", insertErr)
	}
	database.InsertActionSimple(db.FileActionRestore, newID, a.MediaSnapshot,
		fmt.Sprintf("Restored: %s (from undo of action %d)", media.Title, a.ActionHistoryId), "")
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoRescanUpdate(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return fmt.Errorf("no snapshot for action %d — cannot revert metadata", a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return fmt.Errorf("parse snapshot for action %d: %w", a.ActionHistoryId, err)
	}
	if media.ID > 0 {
		if updateErr := database.UpdateMediaByID(media); updateErr != nil {
			return fmt.Errorf("restore metadata for media %d: %w", media.ID, updateErr)
		}
	}
	return database.MarkActionReverted(a.ActionHistoryId)
}

func undoRestore(database *db.DB, a *db.ActionRecord) error {
	if a.MediaId.Valid {
		if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
			return fmt.Errorf("undo restore (delete media %d): %w", a.MediaId.Int64, err)
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
