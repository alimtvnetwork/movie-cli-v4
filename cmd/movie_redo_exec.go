// movie_redo_exec.go — execution helpers for redo command.
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

// executeMoveRedo re-applies a previously reverted file move.
func executeMoveRedo(database *db.DB, m *db.MoveRecord) error {
	statErr := checkFileExists(m.FromPath)
	if statErr != nil {
		return statErr
	}

	destDir := m.ToPath[:strings.LastIndex(m.ToPath, string(os.PathSeparator))]
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return apperror.Wrapf(err, "cannot create directory %s", destDir)
	}

	if err := MoveFile(m.FromPath, m.ToPath); err != nil {
		return apperror.Wrap("redo move", err)
	}

	if err := database.MarkMoveRestored(m.ID); err != nil {
		errlog.Warn("Could not mark move %d as restored: %v", m.ID, err)
	}

	if err := database.UpdateMediaPath(m.MediaID, m.ToPath); err != nil {
		errlog.Warn(fmt.Sprintf("Could not update media path (ID %d): %v", m.ID, err))
	}

	return nil
}

func checkFileExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return apperror.New("file not found at %s — cannot redo", path)
	}
	return apperror.Wrapf(err, "cannot access %s", path)
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
		return apperror.New("unknown action type for redo: %s", a.FileActionId)
	}
	return database.MarkActionRestored(a.ActionHistoryId)
}

func redoScanAdd(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return database.MarkActionRestored(a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return apperror.Wrapf(err, "parse snapshot for redo action %d", a.ActionHistoryId)
	}
	if _, insertErr := database.InsertMedia(media); insertErr != nil {
		return apperror.Wrap("re-insert media for redo", insertErr)
	}
	return database.MarkActionRestored(a.ActionHistoryId)
}

func redoDelete(database *db.DB, a *db.ActionRecord) error {
	if !a.MediaId.Valid {
		return database.MarkActionRestored(a.ActionHistoryId)
	}
	if err := database.DeleteMediaByID(a.MediaId.Int64); err != nil {
		return apperror.Wrapf(err, "redo delete media %d", a.MediaId.Int64)
	}
	return database.MarkActionRestored(a.ActionHistoryId)
}

func redoRestore(database *db.DB, a *db.ActionRecord) error {
	if a.MediaSnapshot == "" {
		return database.MarkActionRestored(a.ActionHistoryId)
	}
	media, err := db.MediaFromJSON(a.MediaSnapshot)
	if err != nil {
		return apperror.Wrapf(err, "parse snapshot for redo restore %d", a.ActionHistoryId)
	}
	if _, insertErr := database.InsertMedia(media); insertErr != nil {
		return apperror.Wrap("redo restore insert", insertErr)
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
		fmt.Printf("   • %s: %s\n", a.FileActionId, actionDetail(a))
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
		return
	}
	fmt.Printf("⚠️  Batch %s: %d redone, %d failed.\n", shortBatch, redoable-failed, failed)
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
	detail := actionDetail(*a)
	fmt.Printf("   %s\n", detail)
	suffix := batchSuffix(a.BatchId)
	if suffix != "" {
		fmt.Printf("   Batch: %s\n", a.BatchId[:8])
	}
}
