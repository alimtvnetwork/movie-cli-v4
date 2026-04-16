// movie_undo_helpers.go — undo execution and UI helpers.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v3/apperror"
	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
)

// ---------------------------------------------------------------------------
// Undo specific action_history by ID
// ---------------------------------------------------------------------------

func undoActionByID(database *db.DB, scanner *bufio.Scanner, id int64) {
	action, err := database.GetActionByID(id)
	if err != nil {
		errlog.Error("Cannot find action %d: %v", id, err)
		return
	}
	if action.IsReverted {
		fmt.Printf("⚠️  Action %d has already been reverted.\n", id)
		return
	}

	fmt.Printf("⏪ Undo action %d (%s):\n", action.ActionHistoryId, action.FileActionId)
	if action.Detail != "" {
		fmt.Printf("   %s\n", action.Detail)
	}
	if !confirmUndo(scanner) {
		return
	}

	if err := executeActionUndo(database, action); err != nil {
		errlog.Error("Undo action %d failed: %v", id, err)
		return
	}
	fmt.Printf("✅ Action %d reverted successfully.\n", action.ActionHistoryId)
}

// ---------------------------------------------------------------------------
// Undo specific move_history by ID
// ---------------------------------------------------------------------------

func undoMoveByID(database *db.DB, scanner *bufio.Scanner, id int64) {
	moves, err := database.ListMoveHistory(1000)
	if err != nil {
		errlog.Error("Cannot read move history: %v", err)
		return
	}
	var target *db.MoveRecord
	for i := range moves {
		if moves[i].ID == id {
			target = &moves[i]
			break
		}
	}
	if target == nil {
		errlog.Error("Move %d not found.", id)
		return
	}
	if target.IsReverted {
		fmt.Printf("⚠️  Move %d has already been reverted.\n", id)
		return
	}

	fmt.Println("⏪ Undo move:")
	fmt.Printf("   %s → %s\n", target.ToPath, target.FromPath)
	if !confirmUndo(scanner) {
		return
	}

	if err := executeMoveUndo(database, target); err != nil {
		errlog.Error("Undo move %d failed: %v", id, err)
		return
	}
	fmt.Printf("✅ Move %d reverted successfully.\n", target.ID)
}

// ---------------------------------------------------------------------------
// --batch: undo last batch
// ---------------------------------------------------------------------------

func undoLastBatch(database *db.DB, scanner *bufio.Scanner) {
	actions, err := database.ListActions(100)
	if err != nil {
		errlog.Error("Cannot read action history: %v", err)
		return
	}

	batchID := ""
	for _, a := range actions {
		if !a.IsReverted && a.BatchId != "" {
			batchID = a.BatchId
			break
		}
	}
	if batchID == "" {
		fmt.Println("📭 No batch operations to undo.")
		return
	}

	batchActions, err := database.ListActionsByBatch(batchID)
	if err != nil {
		errlog.Error("Cannot read batch %s: %v", batchID, err)
		return
	}

	undoable := 0
	for _, a := range batchActions {
		if !a.IsReverted {
			undoable++
		}
	}
	if undoable == 0 {
		fmt.Println("📭 Batch already reverted.")
		return
	}

	fmt.Printf("⏪ Undo batch %s (%d actions):\n", batchID[:8], undoable)
	for _, a := range batchActions {
		if a.IsReverted {
			continue
		}
		detail := a.Detail
		if detail == "" {
			detail = a.FileActionId.String()
		}
		fmt.Printf("   • %s: %s\n", a.FileActionId, detail)
	}
	if !confirmUndo(scanner) {
		return
	}

	// Undo in reverse order (newest first)
	failed := 0
	for i := len(batchActions) - 1; i >= 0; i-- {
		a := batchActions[i]
		if a.IsReverted {
			continue
		}
		if err := executeActionUndo(database, &a); err != nil {
			errlog.Warn("Failed to undo action %d: %v", a.ActionHistoryId, err)
			failed++
		}
	}

	if failed == 0 {
		fmt.Printf("✅ Batch %s reverted (%d actions).\n", batchID[:8], undoable)
	} else {
		fmt.Printf("⚠️  Batch %s: %d reverted, %d failed.\n", batchID[:8], undoable-failed, failed)
	}
}

// ---------------------------------------------------------------------------
// Default: undo last operation (newest of move_history or action_history)
// ---------------------------------------------------------------------------

func undoLastOperation(database *db.DB, scanner *bufio.Scanner) {
	lastMove, moveErr := database.GetLastMove()
	lastAction, actionErr := database.GetLastRevertableAction()

	haveMove := moveErr == nil && lastMove != nil
	haveAction := actionErr == nil && lastAction != nil

	if !haveMove && !haveAction {
		fmt.Println("📭 No operations to undo.")
		return
	}

	if haveMove && !haveAction {
		fmt.Println("⏪ Last move operation:")
		fmt.Printf("   %s → %s\n", lastMove.ToPath, lastMove.FromPath)
		if !confirmUndo(scanner) {
			return
		}
		if err := executeMoveUndo(database, lastMove); err != nil {
			errlog.Error("Undo failed: %v", err)
			return
		}
		fmt.Println("✅ Undo successful!")
		return
	}

	if haveAction && !haveMove {
		printActionUndo(lastAction)
		if !confirmUndo(scanner) {
			return
		}
		if err := executeActionUndo(database, lastAction); err != nil {
			errlog.Error("Undo failed: %v", err)
			return
		}
		fmt.Println("✅ Undo successful!")
		return
	}

	// Both available — compare timestamps
	if lastAction.CreatedAt >= lastMove.MovedAt {
		printActionUndo(lastAction)
		if !confirmUndo(scanner) {
			return
		}
		if err := executeActionUndo(database, lastAction); err != nil {
			errlog.Error("Undo failed: %v", err)
			return
		}
	} else {
		fmt.Println("⏪ Last move operation:")
		fmt.Printf("   %s → %s\n", lastMove.ToPath, lastMove.FromPath)
		if !confirmUndo(scanner) {
			return
		}
		if err := executeMoveUndo(database, lastMove); err != nil {
			errlog.Error("Undo failed: %v", err)
			return
		}
	}
	fmt.Println("✅ Undo successful!")
}

// ---------------------------------------------------------------------------
// Execution helpers
// ---------------------------------------------------------------------------

// executeMoveUndo reverses a file move and updates DB state.
func executeMoveUndo(database *db.DB, m *db.MoveRecord) error {
	if _, err := os.Stat(m.ToPath); err != nil {
		if os.IsNotExist(err) {
			return apperror.Newf("file not found at %s — may have been moved manually", m.ToPath)
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
		// Undo scan_add = delete the media record that was added
		if a.MediaId.Valid {
			if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
				return apperror.Wrapf(err, "undo scan_add (delete media %d)", a.MediaId.Int64)
			}
		}

	case db.FileActionScanRemove, db.FileActionDelete:
		// Undo delete/scan_remove = re-insert from snapshot
		if a.MediaSnapshot == "" {
			return apperror.Newf("no snapshot available for action %d — cannot restore", a.ActionHistoryId)
		}
		media, err := db.MediaFromJSON(a.MediaSnapshot)
		if err != nil {
			return apperror.Wrapf(err, "parse snapshot for action %d", a.ActionHistoryId)
		}
		newID, insertErr := database.InsertMedia(media)
		if insertErr != nil {
			return apperror.Wrap("re-insert media from snapshot", insertErr)
		}
		database.InsertActionSimple(db.FileActionRestore, newID, a.MediaSnapshot,
			fmt.Sprintf("Restored: %s (from undo of action %d)", media.Title, a.ActionHistoryId), "")

	case db.FileActionRescanUpdate:
		// Undo rescan = restore old metadata from snapshot
		if a.MediaSnapshot == "" {
			return apperror.Newf("no snapshot for action %d — cannot revert metadata", a.ActionHistoryId)
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

	case db.FileActionPopout:
		// Popout undo is handled via move_history — just mark action as reverted

	case db.FileActionRestore:
		// Undo a restore = delete the restored record again
		if a.MediaId.Valid {
			if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
				return apperror.Wrapf(err, "undo restore (delete media %d)", a.MediaId.Int64)
			}
		}

	default:
		return apperror.Newf("unknown action type for undo: %s", a.FileActionId)
	}

	return database.MarkActionReverted(a.ActionHistoryId)
}

// ---------------------------------------------------------------------------
// UI helpers
// ---------------------------------------------------------------------------

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
