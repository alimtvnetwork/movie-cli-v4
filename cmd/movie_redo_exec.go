// movie_redo_exec.go — execution helpers for redo command.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

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
		return redoScanAdd(database, a)
	case db.FileActionScanRemove, db.FileActionDelete:
		return redoDelete(database, a)
	case db.FileActionRescanUpdate:
		// Can't re-fetch TMDb here; just mark restored
	case db.FileActionPopout:
		// Handled via move_history; just mark restored
	case db.FileActionRestore:
		return redoRestore(database, a)
	default:
		return fmt.Errorf("unknown action type for redo: %s", a.FileActionId)
	}
	return database.MarkActionRestored(a.ActionHistoryId)
}

func redoScanAdd(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return database.MarkActionRestored(a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return fmt.Errorf("parse snapshot for redo action %d: %w", a.ActionHistoryId, err)
	}
	if _, insertErr := database.InsertMedia(media); insertErr != nil {
		return fmt.Errorf("re-insert media for redo: %w", insertErr)
	}
	return database.MarkActionRestored(a.ActionHistoryId)
}

func redoDelete(database *db.DB, a *db.ActionRecord) error {
	if a.MediaId.Valid {
		media, _ := database.GetMediaByID(a.MediaId.Int64)
		if media != nil {
			if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
				return fmt.Errorf("redo delete media %d: %w", a.MediaId.Int64, err)
			}
		}
	}
	return database.MarkActionRestored(a.ActionHistoryId)
}

func redoRestore(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return database.MarkActionRestored(a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return fmt.Errorf("parse snapshot for redo restore %d: %w", a.ActionHistoryId, err)
	}
	if _, insertErr := database.InsertMedia(media); insertErr != nil {
		return fmt.Errorf("redo restore insert: %w", insertErr)
	}
	return database.MarkActionRestored(a.ActionHistoryId)
}

func findLastRevertedBatch(database *db.DB) string {
	actions, err := database.ListActions(200)
	if err != nil {
		errlog.Error("Cannot read action history: %v", err)
		return ""
	}
	for _, a := range actions {
		if a.IsReverted && a.BatchId != "" {
			return a.BatchId
		}
	}
	return ""
}

func countReverted(actions []db.ActionRecord) int {
	count := 0
	for _, a := range actions {
		if a.IsReverted {
			count++
		}
	}
	return count
}

func printRevertedActions(actions []db.ActionRecord) {
	for _, a := range actions {
		if !a.IsReverted {
			continue
		}
		detail := a.Detail
		if detail == "" {
			detail = a.FileActionId.String()
		}
		fmt.Printf("   • %s: %s\n", a.FileActionId, detail)
	}
}

func executeRedoBatch(database *db.DB, actions []db.ActionRecord) int {
	failed := 0
	for i := range actions {
		if !actions[i].IsReverted {
			continue
		}
		if err := executeActionRedo(database, &actions[i]); err != nil {
			errlog.Warn("Failed to redo action %d: %v", actions[i].ActionHistoryId, err)
			failed++
		}
	}
	return failed
}

func printRedoBatchResult(shortBatch string, redoable, failed int) {
	if failed == 0 {
		fmt.Printf("✅ Batch %s redone (%d actions).\n", shortBatch, redoable)
	} else {
		fmt.Printf("⚠️  Batch %s: %d redone, %d failed.\n", shortBatch, redoable-failed, failed)
	}
}

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
