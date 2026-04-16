// movie_redo_handlers.go — redo subcommand handlers (list, by-id, batch, last).
package cmd

import (
	"bufio"
	"fmt"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

func showRedoableList(database *db.DB) {
	fmt.Println("⏩ Recent redoable operations")
	fmt.Println()

	redoableMoves := printRedoableMoves(database)
	redoableActions := printRedoableActions(database)

	if redoableMoves == 0 && redoableActions == 0 {
		fmt.Println("  📭 Nothing to redo.")
	}
}

func printRedoableMoves(database *db.DB) int {
	moves, _ := database.ListMoveHistory(20)
	count := 0
	for _, m := range moves {
		if m.IsReverted {
			count++
		}
	}
	if count > 0 {
		fmt.Println("  📁 Moves / Renames:")
		for _, m := range moves {
			if !m.IsReverted {
				continue
			}
			fmt.Printf("    [move-%d]  %s → %s  (%s)\n", m.ID, m.FromPath, m.ToPath, m.MovedAt)
		}
		fmt.Println()
	}
	return count
}

func printRedoableActions(database *db.DB) int {
	actions, _ := database.ListActions(40)
	count := countReverted(actions)
	if count == 0 {
		return 0
	}
	fmt.Println("  📋 Actions:")
	for _, a := range actions {
		if !a.IsReverted {
			continue
		}
		fmt.Printf("    [action-%d]  %s  %s  (%s%s)\n",
			a.ActionHistoryId, a.FileActionId, actionDetail(a), a.CreatedAt, batchSuffix(a.BatchId))
	}
	fmt.Println()
	return count
}

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

func redoLastBatch(database *db.DB, scanner *bufio.Scanner) {
	batchID := findLastRevertedBatch(database)
	if batchID == "" {
		fmt.Println("📭 No reverted batch operations to redo.")
		return
	}

	batchActions, err := database.ListActionsByBatch(batchID)
	if err != nil {
		errlog.Error("Cannot read batch %s: %v", batchID, err)
		return
	}

	redoable := countReverted(batchActions)
	if redoable == 0 {
		fmt.Println("📭 Batch has no reverted actions to redo.")
		return
	}

	shortBatch := batchID
	if len(shortBatch) > 8 {
		shortBatch = shortBatch[:8]
	}

	fmt.Printf("⏩ Redo batch %s (%d actions):\n", shortBatch, redoable)
	printRevertedActions(batchActions)
	if !confirmRedo(scanner) {
		return
	}

	failed := executeRedoBatch(database, batchActions)
	printRedoBatchResult(shortBatch, redoable, failed)
}

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
		redoSingleMove(database, scanner, lastMove)
		return
	}

	if haveAction && !haveMove {
		redoSingleAction(database, scanner, lastAction)
		return
	}

	// Both available — pick the newest reverted
	if lastAction.CreatedAt >= lastMove.MovedAt {
		redoSingleAction(database, scanner, lastAction)
		return
	}
	redoSingleMove(database, scanner, lastMove)
}

func redoSingleMove(database *db.DB, scanner *bufio.Scanner, m *db.MoveRecord) {
	fmt.Println("⏩ Redo last move:")
	fmt.Printf("   %s → %s\n", m.FromPath, m.ToPath)
	if !confirmRedo(scanner) {
		return
	}
	if err := executeMoveRedo(database, m); err != nil {
		errlog.Error("Redo failed: %v", err)
		return
	}
	fmt.Println("✅ Redo successful!")
}

func redoSingleAction(database *db.DB, scanner *bufio.Scanner, a *db.ActionRecord) {
	printActionRedo(a)
	if !confirmRedo(scanner) {
		return
	}
	if err := executeActionRedo(database, a); err != nil {
		errlog.Error("Redo failed: %v", err)
		return
	}
	fmt.Println("✅ Redo successful!")
}
