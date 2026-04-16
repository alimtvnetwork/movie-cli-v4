// movie_redo_helpers.go — redo execution and UI helpers.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
)

// ---------------------------------------------------------------------------
// Redo specific action by ID
// ---------------------------------------------------------------------------

func redoActionByID(database *db.DB, scanner *bufio.Scanner, id int64) {
	action, err := database.GetActionByID(id)
	if err != nil {
		errlog.Error("Cannot find action %d: %v", id, err)
		return
	}
	if !action.IsReverted {
		fmt.Printf("⚠️  Action %d is not reverted — nothing to redo.\n", id)
		return
	}

	fmt.Printf("⏩ Redo action %d (%s):\n", action.ActionHistoryId, action.FileActionId)
	if action.Detail != "" {
		fmt.Printf("   %s\n", action.Detail)
	}
	if !confirmRedo(scanner) {
		return
	}

	if err := executeActionRedo(database, action); err != nil {
		errlog.Error("Redo action %d failed: %v", id, err)
		return
	}
	fmt.Printf("✅ Action %d redone successfully.\n", action.ActionHistoryId)
}

// ---------------------------------------------------------------------------
// Redo specific move by ID
// ---------------------------------------------------------------------------

func redoMoveByID(database *db.DB, scanner *bufio.Scanner, id int64) {
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
	if !target.IsReverted {
		fmt.Printf("⚠️  Move %d is not reverted — nothing to redo.\n", id)
		return
	}

	fmt.Println("⏩ Redo move:")
	fmt.Printf("   %s → %s\n", target.FromPath, target.ToPath)
	if !confirmRedo(scanner) {
		return
	}

	if err := executeMoveRedo(database, target); err != nil {
		errlog.Error("Redo move %d failed: %v", id, err)
		return
	}
	fmt.Printf("✅ Move %d redone successfully.\n", target.ID)
}

// ---------------------------------------------------------------------------
// --batch: redo last reverted batch
// ---------------------------------------------------------------------------

func redoLastBatch(database *db.DB, scanner *bufio.Scanner) {
	actions, err := database.ListActions(200)
	if err != nil {
		errlog.Error("Cannot read action history: %v", err)
		return
	}

	batchID := ""
	for _, a := range actions {
		if a.IsReverted && a.BatchId != "" {
			batchID = a.BatchId
			break
		}
	}
	if batchID == "" {
		fmt.Println("📭 No reverted batch operations to redo.")
		return
	}

	batchActions, err := database.ListActionsByBatch(batchID)
	if err != nil {
		errlog.Error("Cannot read batch %s: %v", batchID, err)
		return
	}

	redoable := 0
	for _, a := range batchActions {
		if a.IsReverted {
			redoable++
		}
	}
	if redoable == 0 {
		fmt.Println("📭 Batch has no reverted actions to redo.")
		return
	}

	shortBatch := batchID
	if len(shortBatch) > 8 {
		shortBatch = shortBatch[:8]
	}

	fmt.Printf("⏩ Redo batch %s (%d actions):\n", shortBatch, redoable)
	for _, a := range batchActions {
		if !a.IsReverted {
			continue
		}
		detail := a.Detail
		if detail == "" {
			detail = a.FileActionId.String()
		}
		fmt.Printf("   • %s: %s\n", a.FileActionId, detail)
	}
	if !confirmRedo(scanner) {
		return
	}

	// Redo in original order (oldest first — batchActions already ASC)
	failed := 0
	for i := range batchActions {
		a := batchActions[i]
		if !a.IsReverted {
			continue
		}
		if err := executeActionRedo(database, &a); err != nil {
			errlog.Warn("Failed to redo action %d: %v", a.ActionHistoryId, err)
			failed++
		}
	}

	if failed == 0 {
		fmt.Printf("✅ Batch %s redone (%d actions).\n", shortBatch, redoable)
	} else {
		fmt.Printf("⚠️  Batch %s: %d redone, %d failed.\n", shortBatch, redoable-failed, failed)
	}
}

// ---------------------------------------------------------------------------
// Default: redo last reverted operation
// ---------------------------------------------------------------------------

func redoLastOperation(database *db.DB, scanner *bufio.Scanner) {
	lastMove, moveErr := database.GetLastRevertedMove()
	lastAction, actionErr := database.GetLastRevertedAction()

	haveMove := moveErr == nil && lastMove != nil
	haveAction := actionErr == nil && lastAction != nil

	if !haveMove && !haveAction {
		fmt.Println("📭 No reverted operations to redo.")
		return
	}

	if haveMove && !haveAction {
		fmt.Println("⏩ Redo last move:")
		fmt.Printf("   %s → %s\n", lastMove.FromPath, lastMove.ToPath)
		if !confirmRedo(scanner) {
			return
		}
		if err := executeMoveRedo(database, lastMove); err != nil {
			errlog.Error("Redo failed: %v", err)
			return
		}
		fmt.Println("✅ Redo successful!")
		return
	}

	if haveAction && !haveMove {
		printActionRedo(lastAction)
		if !confirmRedo(scanner) {
			return
		}
		if err := executeActionRedo(database, lastAction); err != nil {
			errlog.Error("Redo failed: %v", err)
			return
		}
		fmt.Println("✅ Redo successful!")
		return
	}

	// Both available — pick the newest reverted
	if lastAction.CreatedAt >= lastMove.MovedAt {
		printActionRedo(lastAction)
		if !confirmRedo(scanner) {
			return
		}
		if err := executeActionRedo(database, lastAction); err != nil {
			errlog.Error("Redo failed: %v", err)
			return
		}
	} else {
		fmt.Println("⏩ Redo last move:")
		fmt.Printf("   %s → %s\n", lastMove.FromPath, lastMove.ToPath)
		if !confirmRedo(scanner) {
			return
		}
		if err := executeMoveRedo(database, lastMove); err != nil {
			errlog.Error("Redo failed: %v", err)
			return
		}
	}
	fmt.Println("✅ Redo successful!")
}

// ---------------------------------------------------------------------------
// Execution helpers
// ---------------------------------------------------------------------------

// executeMoveRedo re-applies a previously reverted file move.
func executeMoveRedo(database *db.DB, m *db.MoveRecord) error {
	if _, err := os.Stat(m.FromPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found at %s — cannot redo", m.FromPath)
		}
		return fmt.Errorf("cannot access %s: %w", m.FromPath, err)
	}

	destDir := m.ToPath[:strings.LastIndex(m.ToPath, string(os.PathSeparator))]
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", destDir, err)
	}

	if err := MoveFile(m.FromPath, m.ToPath); err != nil {
		return fmt.Errorf("redo move: %w", err)
	}

	if err := database.MarkMoveRestored(m.ID); err != nil {
		errlog.Warn("Could not mark move %d as restored: %v", m.ID, err)
	}

	if err := database.UpdateMediaPath(m.MediaID, m.ToPath); err != nil {
		errlog.Warn(fmt.Sprintf("Could not update media path (ID %d): %v", m.ID, err))
	}

	return nil
}

// executeActionRedo re-applies a previously reverted action_history entry.
func executeActionRedo(database *db.DB, a *db.ActionRecord) error {
	switch a.FileActionId {
	case db.FileActionScanAdd:
		if a.MediaSnapshot != "" {
			media, err := db.MediaFromJSON(a.MediaSnapshot)
			if err != nil {
				return fmt.Errorf("parse snapshot for redo action %d: %w", a.ActionHistoryId, err)
			}
			if _, insertErr := database.InsertMedia(media); insertErr != nil {
				return fmt.Errorf("re-insert media for redo: %w", insertErr)
			}
		}

	case db.FileActionScanRemove, db.FileActionDelete:
		if a.MediaId.Valid {
			media, _ := database.GetMediaByID(a.MediaId.Int64)
			if media != nil {
				if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
					return fmt.Errorf("redo delete media %d: %w", a.MediaId.Int64, err)
				}
			}
		}

	case db.FileActionRescanUpdate:
		// Redo rescan = can't re-fetch TMDb here; just mark restored

	case db.FileActionPopout:
		// Popout redo handled via move_history — just mark restored

	case db.FileActionRestore:
		if a.MediaSnapshot != "" {
			media, err := db.MediaFromJSON(a.MediaSnapshot)
			if err != nil {
				return fmt.Errorf("parse snapshot for redo restore %d: %w", a.ActionHistoryId, err)
			}
			if _, insertErr := database.InsertMedia(media); insertErr != nil {
				return fmt.Errorf("redo restore insert: %w", insertErr)
			}
		}

	default:
		return fmt.Errorf("unknown action type for redo: %s", a.FileActionId)
	}

	return database.MarkActionRestored(a.ActionHistoryId)
}

// ---------------------------------------------------------------------------
// UI helpers
// ---------------------------------------------------------------------------

func confirmRedo(scanner *bufio.Scanner) bool {
	fmt.Print("\n  Redo this? [y/N]: ")
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if answer != "y" && answer != "yes" {
		fmt.Println("❌ Redo canceled.")
		return false
	}
	return true
}

func printActionRedo(a *db.ActionRecord) {
	fmt.Printf("⏩ Last reverted action (%s):\n", a.FileActionId)
	if a.Detail != "" {
		fmt.Printf("   %s\n", a.Detail)
	}
	if a.BatchId != "" {
		short := a.BatchId
		if len(short) > 8 {
			short = short[:8]
		}
		fmt.Printf("   Batch: %s\n", short)
	}
}
