// movie_redo.go — movie redo: re-applies the last reverted operation.
//
// Supports redoing:
//   - File moves/renames  (from move_history, IsReverted=1)
//   - Action history ops  (from action_history, IsReverted=1)
//
// Flags:
//
//	--list           Show recent redoable actions
//	--id <id>        Redo a specific action_history record
//	--move-id <id>   Redo a specific move_history record
//	--batch          Redo the entire last reverted batch
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
	redoListFlag  bool
	redoBatchFlag bool
	redoActionID  int64
	redoMoveID    int64
)

var movieRedoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Redo the last reverted operation",
	Long: `Re-applies the most recent reverted operation.

Without flags, redoes the single most recent reverted action
(checks both move_history and action_history, picks the newest).

Flags:
  --list           Show recent redoable actions
  --id <id>        Redo a specific action_history record by ID
  --move-id <id>   Redo a specific move_history record by ID
  --batch          Redo the entire last reverted batch`,
	Run: runMovieRedo,
}

func init() {
	movieRedoCmd.Flags().BoolVar(&redoListFlag, "list", false, "Show recent redoable actions")
	movieRedoCmd.Flags().BoolVar(&redoBatchFlag, "batch", false, "Redo entire last reverted batch")
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
		if m.IsReverted {
			redoableMoves++
		}
	}
	if redoableMoves > 0 {
		fmt.Println("  📁 Moves / Renames:")
		for _, m := range moves {
			if !m.IsReverted {
				continue
			}
			fmt.Printf("    [move-%d]  %s → %s  (%s)\n", m.ID, m.FromPath, m.ToPath, m.MovedAt)
		}
		fmt.Println()
	}

	actions, _ := database.ListActions(40)
	redoableActions := 0
	for _, a := range actions {
		if a.IsReverted {
			redoableActions++
		}
	}
	if redoableActions > 0 {
		fmt.Println("  📋 Actions:")
		for _, a := range actions {
			if !a.IsReverted {
				continue
			}
			detail := a.Detail
			if detail == "" {
				detail = a.FileActionId.String()
			}
			batchStr := ""
			if a.BatchId != "" && len(a.BatchId) >= 8 {
				batchStr = fmt.Sprintf("  batch:%s", a.BatchId[:8])
			}
			fmt.Printf("    [action-%d]  %s  %s  (%s%s)\n",
				a.ActionHistoryId, a.FileActionId, detail, a.CreatedAt, batchStr)
		}
		fmt.Println()
	}

	if redoableMoves == 0 && redoableActions == 0 {
		fmt.Println("  📭 Nothing to redo.")
	}
}
