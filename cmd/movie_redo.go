// movie_redo.go — movie redo: re-applies the last undone operation.
//
// Supports redoing:
//   - File moves/renames  (from move_history, undone=1)
//   - Action history ops  (from action_history, undone=1)
//
// Flags:
//
//	--list           Show recent redoable actions
//	--id <id>        Redo a specific action_history record
//	--move-id <id>   Redo a specific move_history record
//	--batch          Redo the entire last undone batch
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
)

var (
	redoListFlag  bool
	redoBatchFlag bool
	redoActionID  int64
	redoMoveID    int64
)

var movieRedoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Redo the last undone operation",
	Long: `Re-applies the most recent undone operation.

Without flags, redoes the single most recent undone action
(checks both move_history and action_history, picks the newest).

Flags:
  --list           Show recent redoable actions
  --id <id>        Redo a specific action_history record by ID
  --move-id <id>   Redo a specific move_history record by ID
  --batch          Redo the entire last undone batch`,
	Run: runMovieRedo,
}

func init() {
	movieRedoCmd.Flags().BoolVar(&redoListFlag, "list", false, "Show recent redoable actions")
	movieRedoCmd.Flags().BoolVar(&redoBatchFlag, "batch", false, "Redo entire last undone batch")
	movieRedoCmd.Flags().Int64Var(&redoActionID, "id", 0, "Redo specific action_history record")
	movieRedoCmd.Flags().Int64Var(&redoMoveID, "move-id", 0, "Redo specific move_history record")
}

func runMovieRedo(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	scanner := bufio.NewScanner(os.Stdin)

	if redoListFlag {
		showRedoableList(database)
		return
	}
	if redoActionID > 0 {
		redoActionByID(database, scanner, redoActionID)
		return
	}
	if redoMoveID > 0 {
		redoMoveByID(database, scanner, redoMoveID)
		return
	}
	if redoBatchFlag {
		redoLastBatch(database, scanner)
		return
	}

	// Default: redo the most recent undone operation
	redoLastOperation(database, scanner)
}

// ---------------------------------------------------------------------------
// --list
// ---------------------------------------------------------------------------

func showRedoableList(database *db.DB) {
	fmt.Println("⏩ Recent redoable operations")
	fmt.Println()

	moves, _ := database.ListMoveHistory(20)
	redoableMoves := 0
	for _, m := range moves {
		if m.Undone {
			redoableMoves++
		}
	}
	if redoableMoves > 0 {
		fmt.Println("  📁 Moves / Renames:")
		for _, m := range moves {
			if !m.Undone {
				continue
			}
			fmt.Printf("    [move-%d]  %s → %s  (%s)\n", m.ID, m.FromPath, m.ToPath, m.MovedAt)
		}
		fmt.Println()
	}

	actions, _ := database.ListActions(40)
	redoableActions := 0
	for _, a := range actions {
		if a.Undone {
			redoableActions++
		}
	}
	if redoableActions > 0 {
		fmt.Println("  📋 Actions:")
		for _, a := range actions {
			if !a.Undone {
				continue
			}
			detail := a.Detail
			if detail == "" {
				detail = string(a.ActionType)
			}
			batchStr := ""
			if a.BatchID != "" && len(a.BatchID) >= 8 {
				batchStr = fmt.Sprintf("  batch:%s", a.BatchID[:8])
			}
			fmt.Printf("    [action-%d]  %s  %s  (%s%s)\n",
				a.ID, a.ActionType, detail, a.CreatedAt, batchStr)
		}
		fmt.Println()
	}

	if redoableMoves == 0 && redoableActions == 0 {
		fmt.Println("  📭 Nothing to redo.")
	}
}

// ---------------------------------------------------------------------------
// Redo specific action by ID
// ---------------------------------------------------------------------------

func redoActionByID(database *db.DB, scanner *bufio.Scanner, id int64) {
	action, err := database.GetActionByID(id)
	if err != nil {
		errlog.Error("Cannot find action %d: %v", id, err)
		return
	}
	if !action.Undone {
		fmt.Printf("⚠️  Action %d is not undone — nothing to redo.\n", id)
		return
	}

	fmt.Printf("⏩ Redo action %d (%s):\n", action.ID, action.ActionType)
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
	fmt.Printf("✅ Action %d redone successfully.\n", action.ID)
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
	if !target.Undone {
		fmt.Printf("⚠️  Move %d is not undone — nothing to redo.\n", id)
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
// --batch: redo last undone batch
// ---------------------------------------------------------------------------

func redoLastBatch(database *db.DB, scanner *bufio.Scanner) {
	actions, err := database.ListActions(200)
	if err != nil {
		errlog.Error("Cannot read action history: %v", err)
		return
	}

	batchID := ""
	for _, a := range actions {
		if a.Undone && a.BatchID != "" {
			batchID = a.BatchID
			break
		}
	}
	if batchID == "" {
		fmt.Println("📭 No undone batch operations to redo.")
		return
	}

	batchActions, err := database.ListActionsByBatch(batchID)
	if err != nil {
		errlog.Error("Cannot read batch %s: %v", batchID, err)
		return
	}

	redoable := 0
	for _, a := range batchActions {
		if a.Undone {
			redoable++
		}
	}
	if redoable == 0 {
		fmt.Println("📭 Batch has no undone actions to redo.")
		return
	}

	shortBatch := batchID
	if len(shortBatch) > 8 {
		shortBatch = shortBatch[:8]
	}

	fmt.Printf("⏩ Redo batch %s (%d actions):\n", shortBatch, redoable)
	for _, a := range batchActions {
		if !a.Undone {
			continue
		}
		detail := a.Detail
		if detail == "" {
			detail = string(a.ActionType)
		}
		fmt.Printf("   • %s: %s\n", a.ActionType, detail)
	}
	if !confirmRedo(scanner) {
		return
	}

	// Redo in original order (oldest first — batchActions already ASC)
	failed := 0
	for i := range batchActions {
		a := batchActions[i]
		if !a.Undone {
			continue
		}
		if err := executeActionRedo(database, &a); err != nil {
			errlog.Warn("Failed to redo action %d: %v", a.ID, err)
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
// Default: redo last undone operation
// ---------------------------------------------------------------------------

func redoLastOperation(database *db.DB, scanner *bufio.Scanner) {
	lastMove, moveErr := database.GetLastUndoneMove()
	lastAction, actionErr := database.GetLastRedoableAction()

	haveMove := moveErr == nil && lastMove != nil
	haveAction := actionErr == nil && lastAction != nil

	if !haveMove && !haveAction {
		fmt.Println("📭 No undone operations to redo.")
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

	// Both available — pick the newest undone
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

// executeMoveRedo re-applies a previously undone file move.
func executeMoveRedo(database *db.DB, m *db.MoveRecord) error {
	// File should be at FromPath (it was moved back during undo)
	if _, err := os.Stat(m.FromPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found at %s — cannot redo", m.FromPath)
		}
		return fmt.Errorf("cannot access %s: %w", m.FromPath, err)
	}

	// Ensure destination directory exists
	destDir := m.ToPath[:strings.LastIndex(m.ToPath, string(os.PathSeparator))]
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", destDir, err)
	}

	// Move file forward again
	if err := MoveFile(m.FromPath, m.ToPath); err != nil {
		return fmt.Errorf("redo move: %w", err)
	}

	// Mark as not-undone
	if err := database.MarkMoveRedone(m.ID); err != nil {
		errlog.Warn("Could not mark move %d as redone: %v", m.ID, err)
	}

	// Update media path
	if err := database.UpdateMediaPath(m.MediaID, m.ToPath); err != nil {
		errlog.Warn(fmt.Sprintf("Could not update media path (ID %d): %v", m.ID, err))
	}

	return nil
}

// executeActionRedo re-applies a previously undone action_history entry.
func executeActionRedo(database *db.DB, a *db.ActionRecord) error {
	switch a.ActionType {
	case db.ActionScanAdd:
		// Redo scan_add = the media was deleted during undo, re-insert from snapshot if available
		// If no snapshot, the redo is a no-op (just mark redone)
		if a.MediaSnapshot != "" {
			media, err := db.MediaFromJSON(a.MediaSnapshot)
			if err != nil {
				return fmt.Errorf("parse snapshot for redo action %d: %w", a.ID, err)
			}
			if _, insertErr := database.InsertMedia(media); insertErr != nil {
				return fmt.Errorf("re-insert media for redo: %w", insertErr)
			}
		}

	case db.ActionScanRemove, db.ActionDelete:
		// Redo delete/scan_remove = delete the media again
		if a.MediaID.Valid {
			// Snapshot current state before re-deleting
			media, _ := database.GetMediaByID(a.MediaID.Int64)
			if media != nil {
				if err := database.DeleteMediaByID(a.MediaID.Int64); err != nil {
					return fmt.Errorf("redo delete media %d: %w", a.MediaID.Int64, err)
				}
			}
		}

	case db.ActionRescanUpdate:
		// Redo rescan = we can't re-fetch TMDb here; just mark redone
		// The next scan will re-fetch automatically

	case db.ActionPopout:
		// Popout redo handled via move_history — just mark redone

	case db.ActionRestore:
		// Redo restore = re-insert from snapshot
		if a.MediaSnapshot != "" {
			media, err := db.MediaFromJSON(a.MediaSnapshot)
			if err != nil {
				return fmt.Errorf("parse snapshot for redo restore %d: %w", a.ID, err)
			}
			if _, insertErr := database.InsertMedia(media); insertErr != nil {
				return fmt.Errorf("redo restore insert: %w", insertErr)
			}
		}

	default:
		return fmt.Errorf("unknown action type for redo: %s", a.ActionType)
	}

	return database.MarkActionRedone(a.ID)
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
	fmt.Printf("⏩ Last undone action (%s):\n", a.ActionType)
	if a.Detail != "" {
		fmt.Printf("   %s\n", a.Detail)
	}
	if a.BatchID != "" {
		short := a.BatchID
		if len(short) > 8 {
			short = short[:8]
		}
		fmt.Printf("   Batch: %s\n", short)
	}
}
