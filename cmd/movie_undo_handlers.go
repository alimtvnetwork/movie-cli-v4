// movie_undo_handlers.go — undo subcommand handlers (list, by-id, batch, last).
package cmd

import (
	"bufio"
	"fmt"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

func showUndoableList(database *db.DB) {
	fmt.Println("⏪ Recent undoable operations")
	fmt.Println()

	undoableMoves := printUndoableMoves(database)
	undoableActions := printUndoableActions(database)

	if undoableMoves == 0 && undoableActions == 0 {
		fmt.Println("  📭 Nothing to undo.")
	}
}

func printUndoableMoves(database *db.DB) int {
	moves, _ := database.ListMoveHistory(10)
	count := 0
	for _, m := range moves {
		if !m.IsReverted {
			count++
		}
	}
	if count > 0 {
		fmt.Println("  📁 Moves / Renames:")
		for _, m := range moves {
			if m.IsReverted {
				continue
			}
			fmt.Printf("    [move-%d]  %s → %s  (%s)\n", m.ID, m.FromPath, m.ToPath, m.MovedAt)
		}
		fmt.Println()
	}
	return count
}

func printUndoableActions(database *db.DB) int {
	actions, _ := database.ListActions(20)
	count := 0
	for _, a := range actions {
		if !a.IsReverted {
			count++
		}
	}
	if count > 0 {
		fmt.Println("  📋 Actions:")
		for _, a := range actions {
			if a.IsReverted {
				continue
			}
			detail := a.Detail
			if detail == "" {
				detail = a.FileActionId.String()
			}
			batchStr := ""
			if a.BatchId != "" {
				batchStr = fmt.Sprintf("  batch:%s", a.BatchId[:8])
			}
			fmt.Printf("    [action-%d]  %s  %s  (%s%s)\n",
				a.ActionHistoryId, a.FileActionId, detail, a.CreatedAt, batchStr)
		}
		fmt.Println()
	}
	return count
}

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

func undoLastBatch(database *db.DB, scanner *bufio.Scanner) {
	batchID := findLastUndoableBatch(database)
	if batchID == "" {
		fmt.Println("📭 No batch operations to undo.")
		return
	}

	batchActions, err := database.ListActionsByBatch(batchID)
	if err != nil {
		errlog.Error("Cannot read batch %s: %v", batchID, err)
		return
	}

	undoable := countUndoable(batchActions)
	if undoable == 0 {
		fmt.Println("📭 Batch already reverted.")
		return
	}

	fmt.Printf("⏪ Undo batch %s (%d actions):\n", batchID[:8], undoable)
	printUndoableActionsList(batchActions)
	if !confirmUndo(scanner) {
		return
	}

	failed := executeUndoBatch(database, batchActions)
	printUndoBatchResult(batchID[:8], undoable, failed)
}

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
		undoSingleMove(database, scanner, lastMove)
		return
	}

	if haveAction && !haveMove {
		undoSingleAction(database, scanner, lastAction)
		return
	}

	if lastAction.CreatedAt >= lastMove.MovedAt {
		undoSingleAction(database, scanner, lastAction)
	} else {
		undoSingleMove(database, scanner, lastMove)
	}
}

func undoSingleMove(database *db.DB, scanner *bufio.Scanner, m *db.MoveRecord) {
	fmt.Println("⏪ Last move operation:")
	fmt.Printf("   %s → %s\n", m.ToPath, m.FromPath)
	if !confirmUndo(scanner) {
		return
	}
	if err := executeMoveUndo(database, m); err != nil {
		errlog.Error("Undo failed: %v", err)
		return
	}
	fmt.Println("✅ Undo successful!")
}

func undoSingleAction(database *db.DB, scanner *bufio.Scanner, a *db.ActionRecord) {
	printActionUndo(a)
	if !confirmUndo(scanner) {
		return
	}
	if err := executeActionUndo(database, a); err != nil {
		errlog.Error("Undo failed: %v", err)
		return
	}
	fmt.Println("✅ Undo successful!")
}

func findLastUndoableBatch(database *db.DB) string {
	actions, err := database.ListActions(100)
	if err != nil {
		errlog.Error("Cannot read action history: %v", err)
		return ""
	}
	for _, a := range actions {
		if !a.IsReverted && a.BatchId != "" {
			return a.BatchId
		}
	}
	return ""
}

func countUndoable(actions []db.ActionRecord) int {
	count := 0
	for _, a := range actions {
		if !a.IsReverted {
			count++
		}
	}
	return count
}

func printUndoableActionsList(actions []db.ActionRecord) {
	for _, a := range actions {
		if a.IsReverted {
			continue
		}
		detail := a.Detail
		if detail == "" {
			detail = a.FileActionId.String()
		}
		fmt.Printf("   • %s: %s\n", a.FileActionId, detail)
	}
}

func executeUndoBatch(database *db.DB, actions []db.ActionRecord) int {
	failed := 0
	for i := len(actions) - 1; i >= 0; i-- {
		if actions[i].IsReverted {
			continue
		}
		if err := executeActionUndo(database, &actions[i]); err != nil {
			errlog.Warn("Failed to undo action %d: %v", actions[i].ActionHistoryId, err)
			failed++
		}
	}
	return failed
}

func printUndoBatchResult(shortBatch string, undoable, failed int) {
	if failed == 0 {
		fmt.Printf("✅ Batch %s reverted (%d actions).\n", shortBatch, undoable)
	} else {
		fmt.Printf("⚠️  Batch %s: %d reverted, %d failed.\n", shortBatch, undoable-failed, failed)
	}
}
