// action_history.go — ActionHistory table: types and helpers.
package db

import (
	"database/sql"
	"fmt"
)

// FileActionType maps to the FileAction lookup table's FileActionId.
type FileActionType int

const (
	FileActionMove                  FileActionType = 1
	FileActionRename                FileActionType = 2
	FileActionDelete                FileActionType = 3
	FileActionPopout                FileActionType = 4
	FileActionRestore               FileActionType = 5
	FileActionScanAdd               FileActionType = 6
	FileActionScanRemove            FileActionType = 7
	FileActionRescanUpdate          FileActionType = 8
	FileActionTagAdd                FileActionType = 9
	FileActionTagRemove             FileActionType = 10
	FileActionWatchlistAdd          FileActionType = 11
	FileActionWatchlistRemove       FileActionType = 12
	FileActionWatchlistStatusChange FileActionType = 13
	FileActionConfigChange          FileActionType = 14
)

// fileActionNames maps FileActionType to display name.
var fileActionNames = map[FileActionType]string{
	FileActionMove:                  "Move",
	FileActionRename:                "Rename",
	FileActionDelete:                "Delete",
	FileActionPopout:                "Popout",
	FileActionRestore:               "Restore",
	FileActionScanAdd:               "ScanAdd",
	FileActionScanRemove:            "ScanRemove",
	FileActionRescanUpdate:          "RescanUpdate",
	FileActionTagAdd:                "TagAdd",
	FileActionTagRemove:             "TagRemove",
	FileActionWatchlistAdd:          "WatchlistAdd",
	FileActionWatchlistRemove:       "WatchlistRemove",
	FileActionWatchlistStatusChange: "WatchlistStatusChange",
	FileActionConfigChange:          "ConfigChange",
}

// String returns the human-readable name for a FileActionType.
func (f FileActionType) String() string {
	if name, ok := fileActionNames[f]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", int(f))
}

// ActionRecord represents a row in ActionHistory.
type ActionRecord struct {
	ActionHistoryId int64
	FileActionId    FileActionType
	MediaId         sql.NullInt64
	MediaSnapshot   string
	Detail          string
	BatchId         string
	IsReverted      bool
	CreatedAt       string
}

const actionCols = `ActionHistoryId, FileActionId, MediaId, MediaSnapshot, Detail, BatchId, IsReverted, CreatedAt`

// InsertAction logs a state-changing action to ActionHistory.
func (d *DB) InsertAction(fileAction FileActionType, mediaId sql.NullInt64, snapshot, detail, batchId string) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO ActionHistory (FileActionId, MediaId, MediaSnapshot, Detail, BatchId)
		VALUES (?, ?, ?, ?, ?)`,
		int(fileAction), mediaId, snapshot, detail, batchId,
	)
	if err != nil {
		return 0, fmt.Errorf("insert action (%s): %w", fileAction, err)
	}
	return res.LastInsertId()
}

// InsertActionSimple is a convenience wrapper when MediaId is a plain int64.
func (d *DB) InsertActionSimple(fileAction FileActionType, mediaId int64, snapshot, detail, batchId string) (int64, error) {
	mid := sql.NullInt64{Int64: mediaId, Valid: mediaId > 0}
	return d.InsertAction(fileAction, mid, snapshot, detail, batchId)
}

// GetLastRevertableAction returns the most recent non-reverted action.
func (d *DB) GetLastRevertableAction() (*ActionRecord, error) {
	row := d.QueryRow(`
		SELECT ` + actionCols + `
		FROM ActionHistory
		WHERE IsReverted = 0
		ORDER BY ActionHistoryId DESC LIMIT 1`)
	return scanActionRow(row)
}

// GetActionByID returns a single action by primary key.
func (d *DB) GetActionByID(id int64) (*ActionRecord, error) {
	row := d.QueryRow(`
		SELECT `+actionCols+`
		FROM ActionHistory WHERE ActionHistoryId = ?`, id)
	return scanActionRow(row)
}

// GetLastRevertedAction returns the most recent reverted action (for redo).
func (d *DB) GetLastRevertedAction() (*ActionRecord, error) {
	row := d.QueryRow(`
		SELECT ` + actionCols + `
		FROM ActionHistory
		WHERE IsReverted = 1
		ORDER BY ActionHistoryId DESC LIMIT 1`)
	return scanActionRow(row)
}

// ListActions returns recent ActionHistory records, newest first.
func (d *DB) ListActions(limit int) ([]ActionRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.Query(`
		SELECT `+actionCols+`
		FROM ActionHistory
		ORDER BY ActionHistoryId DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer rows.Close()
	return scanActionRows(rows)
}

// ListActionsByType filters by FileActionId.
func (d *DB) ListActionsByType(fileAction FileActionType, limit int) ([]ActionRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.Query(`
		SELECT `+actionCols+`
		FROM ActionHistory
		WHERE FileActionId = ?
		ORDER BY ActionHistoryId DESC LIMIT ?`, int(fileAction), limit)
	if err != nil {
		return nil, fmt.Errorf("list actions by type: %w", err)
	}
	defer rows.Close()
	return scanActionRows(rows)
}

// ListActionsByBatch returns all actions sharing a BatchId.
func (d *DB) ListActionsByBatch(batchId string) ([]ActionRecord, error) {
	rows, err := d.Query(`
		SELECT `+actionCols+`
		FROM ActionHistory
		WHERE BatchId = ?
		ORDER BY ActionHistoryId ASC`, batchId)
	if err != nil {
		return nil, fmt.Errorf("list actions by batch: %w", err)
	}
	defer rows.Close()
	return scanActionRows(rows)
}

// MarkActionReverted sets IsReverted = 1 for the given action.
func (d *DB) MarkActionReverted(id int64) error {
	_, err := d.Exec("UPDATE ActionHistory SET IsReverted = 1 WHERE ActionHistoryId = ?", id)
	if err != nil {
		return fmt.Errorf("mark action reverted %d: %w", id, err)
	}
	return nil
}

// MarkActionRestored sets IsReverted = 0 for the given action (redo).
func (d *DB) MarkActionRestored(id int64) error {
	_, err := d.Exec("UPDATE ActionHistory SET IsReverted = 0 WHERE ActionHistoryId = ?", id)
	if err != nil {
		return fmt.Errorf("mark action restored %d: %w", id, err)
	}
	return nil
}

// MarkBatchReverted sets IsReverted = 1 for all actions in a batch.
func (d *DB) MarkBatchReverted(batchId string) error {
	_, err := d.Exec("UPDATE ActionHistory SET IsReverted = 1 WHERE BatchId = ?", batchId)
	if err != nil {
		return fmt.Errorf("mark batch reverted %s: %w", batchId, err)
	}
	return nil
}

// MarkBatchRestored sets IsReverted = 0 for all actions in a batch.
func (d *DB) MarkBatchRestored(batchId string) error {
	_, err := d.Exec("UPDATE ActionHistory SET IsReverted = 0 WHERE BatchId = ?", batchId)
	if err != nil {
		return fmt.Errorf("mark batch restored %s: %w", batchId, err)
	}
	return nil
}

func scanActionRow(row *sql.Row) (*ActionRecord, error) {
	r := &ActionRecord{}
	err := row.Scan(&r.ActionHistoryId, &r.FileActionId, &r.MediaId, &r.MediaSnapshot,
		&r.Detail, &r.BatchId, &r.IsReverted, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan action row: %w", err)
	}
	return r, nil
}

func scanActionRows(rows *sql.Rows) ([]ActionRecord, error) {
	var records []ActionRecord
	for rows.Next() {
		var r ActionRecord
		if err := rows.Scan(&r.ActionHistoryId, &r.FileActionId, &r.MediaId, &r.MediaSnapshot,
			&r.Detail, &r.BatchId, &r.IsReverted, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan action rows: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("action rows iteration: %w", err)
	}
	return records, nil
}
