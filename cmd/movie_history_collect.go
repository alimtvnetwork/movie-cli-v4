// movie_history_collect.go — data collection and helpers for history command.
package cmd

import (
	"fmt"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

// collectUnifiedRecords gathers records from both tables based on --type filter.
func collectUnifiedRecords(database *db.DB) []unifiedRecord {
	var records []unifiedRecord

	// Include move_history records
	if historyType == "all" || historyType == "move" || historyType == "rename" {
		records = collectMoveRecords(database, records)
	}

	// Include action_history records
	if shouldIncludeActions() {
		records = collectActionRecords(database, records)
	}

	// Sort by timestamp descending (newest first)
	sortRecordsByTimestamp(records)

	// Apply --since filter
	if historySince != "" {
		var filtered []unifiedRecord
		for _, r := range records {
			if r.Timestamp >= historySince {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	// Apply limit
	if len(records) > historyLimit {
		records = records[:historyLimit]
	}

	return records
}

func collectMoveRecords(database *db.DB, records []unifiedRecord) []unifiedRecord {
	moves, err := database.ListMoveHistory(historyLimit)
	if err != nil {
		errlog.Warn("Error reading move history: %v", err)
	}
	for _, m := range moves {
		recType := detectMoveType(m)
		if historyType != "all" && historyType != recType {
			continue
		}
		detail := fmt.Sprintf("%s → %s", m.OriginalFileName, m.NewFileName)
		records = append(records, unifiedRecord{
			Source:     "move",
			ID:         m.ID,
			Type:       recType,
			Detail:     detail,
			FromPath:   m.FromPath,
			ToPath:     m.ToPath,
			Timestamp:  m.MovedAt,
			IsReverted: m.IsReverted,
		})
	}
	return records
}

func collectActionRecords(database *db.DB, records []unifiedRecord) []unifiedRecord {
	var actions []db.ActionRecord
	var err error

	switch historyType {
	case "scan":
		adds, _ := database.ListActionsByType(db.FileActionScanAdd, historyLimit)
		removes, _ := database.ListActionsByType(db.FileActionScanRemove, historyLimit)
		actions = append(adds, removes...)
	case "delete":
		actions, err = database.ListActionsByType(db.FileActionDelete, historyLimit)
	case "popout":
		actions, err = database.ListActionsByType(db.FileActionPopout, historyLimit)
	case "rescan":
		actions, err = database.ListActionsByType(db.FileActionRescanUpdate, historyLimit)
	default: // "all"
		actions, err = database.ListActions(historyLimit)
	}
	if err != nil {
		errlog.Warn("Error reading action history: %v", err)
	}

	for _, a := range actions {
		detail := actionDetail(a)
		records = append(records, unifiedRecord{
			Source:     "action",
			ID:         a.ActionHistoryId,
			Type:       a.FileActionId.String(),
			Detail:     detail,
			BatchID:    a.BatchId,
			Timestamp:  a.CreatedAt,
			IsReverted: a.IsReverted,
		})
	}
	return records
}

func shouldIncludeActions() bool {
	switch historyType {
	case "move", "rename":
		return false
	default:
		return true
	}
}

// detectMoveType returns "rename" if source and dest share a directory, otherwise "move".
func detectMoveType(m db.MoveRecord) string {
	if m.FromPath == "" || m.ToPath == "" {
		return "move"
	}
	if dirOf(m.FromPath) == dirOf(m.ToPath) {
		return "rename"
	}
	return "move"
}

func showBatchHistory(database *db.DB) {
	actions, err := database.ListActionsByBatch(historyBatch)
	if err != nil {
		errlog.Error("Error reading batch %s: %v", historyBatch, err)
		return
	}
	if len(actions) == 0 {
		actions = findPartialBatchMatch(database, historyBatch)
		if len(actions) == 0 {
			fmt.Printf("📭 No actions found for batch: %s\n", historyBatch)
			return
		}
	}

	fmt.Printf("📋 Batch: %s (%d actions)\n\n", historyBatch, len(actions))
	for _, a := range actions {
		printBatchAction(a)
	}
}

func findPartialBatchMatch(database *db.DB, batchPrefix string) []db.ActionRecord {
	allActions, listErr := database.ListActions(200)
	if listErr != nil {
		errlog.Error("Error reading actions: %v", listErr)
		return nil
	}
	var matched []db.ActionRecord
	for _, a := range allActions {
		if len(a.BatchId) >= len(batchPrefix) && a.BatchId[:len(batchPrefix)] == batchPrefix {
			matched = append(matched, a)
		}
	}
	return matched
}

func printBatchAction(a db.ActionRecord) {
	status := "✅"
	if a.IsReverted {
		status = "↩️ "
	}
	fmt.Printf("  %s [%s] %s\n", status, a.FileActionId, actionDetail(a))
	fmt.Printf("     ID: %d  Created: %s\n\n", a.ActionHistoryId, a.CreatedAt)
}
