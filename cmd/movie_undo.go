// movie_undo.go — movie undo: reverts the last state-changing operation.
//
// Supports undoing:
//   - File moves/renames  (from move_history)
//   - Deletions           (from action_history)
//   - Scan additions      (from action_history)
//   - Scan removals       (from action_history)
//   - Rescan updates      (from action_history)
//
// Flags:
//
//	--list           Show recent undoable actions
//	--id <id>        Undo a specific action_history record
//	--batch          Undo entire last batch
//	--move-id <id>   Undo a specific move_history record
package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
)

var (
	undoListFlag    bool
	undoBatchFlag   bool
	undoActionID    int64
	undoMoveID      int64
)

var movieUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo the last operation (move, rename, delete, scan)",
	Long: `Reverts the most recent state-changing operation.

Without flags, undoes the single most recent undoable action
(checks both move_history and action_history, picks the newest).

Flags:
  --list           Show recent undoable actions
  --id <id>        Undo a specific action_history record by ID
  --move-id <id>   Undo a specific move_history record by ID
  --batch          Undo the entire last batch (e.g. a full scan)`,
	Run: runMovieUndo,
}

func init() {
	movieUndoCmd.Flags().BoolVar(&undoListFlag, "list", false, "Show recent undoable actions")
	movieUndoCmd.Flags().BoolVar(&undoBatchFlag, "batch", false, "Undo entire last batch")
	movieUndoCmd.Flags().Int64Var(&undoActionID, "id", 0, "Undo specific action_history record")
	movieUndoCmd.Flags().Int64Var(&undoMoveID, "move-id", 0, "Undo specific move_history record")
}

func runMovieUndo(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	scanner := bufio.NewScanner(os.Stdin)

	if undoListFlag {
		showUndoableList(database)
		return
	}
	if undoActionID > 0 {
		undoActionByID(database, scanner, undoActionID)
		return
	}
	if undoMoveID > 0 {
		undoMoveByID(database, scanner, undoMoveID)
		return
	}
	if undoBatchFlag {
		undoLastBatch(database, scanner)
		return
	}

	undoLastOperation(database, scanner)
}

// ---------------------------------------------------------------------------
// --list
// ---------------------------------------------------------------------------

func showUndoableList(database *db.DB) {
	fmt.Println("⏪ Recent undoable operations")
	fmt.Println()

	// Move history
	moves, _ := database.ListMoveHistory(10)
	undoableMoves := 0
	for _, m := range moves {
		if !m.IsReverted {
			undoableMoves++
		}
	}
	if undoableMoves > 0 {
		fmt.Println("  📁 Moves / Renames:")
		for _, m := range moves {
			if m.IsReverted {
				continue
			}
			fmt.Printf("    [move-%d]  %s → %s  (%s)\n", m.ID, m.FromPath, m.ToPath, m.MovedAt)
		}
		fmt.Println()
	}

	// Action history
	actions, _ := database.ListActions(20)
	undoableActions := 0
	for _, a := range actions {
		if !a.IsReverted {
			undoableActions++
		}
	}
	if undoableActions > 0 {
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

	if undoableMoves == 0 && undoableActions == 0 {
		fmt.Println("  📭 Nothing to undo.")
	}
}
