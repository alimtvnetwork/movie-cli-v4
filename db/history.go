package db

// MoveRecord represents a row in MoveHistory.
type MoveRecord struct {
	FromPath         string
	ToPath           string
	OriginalFileName string
	NewFileName      string
	MovedAt          string
	ID               int64
	MediaID          int64
	FileActionId     int
	IsReverted       bool
}

// ListMoveHistory returns all move records ordered by most recent first.
func (d *DB) ListMoveHistory(limit int) ([]MoveRecord, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := d.Query(`
		SELECT MoveHistoryId, MediaId, FileActionId, FromPath, ToPath,
		       OriginalFileName, NewFileName, MovedAt, IsReverted
		FROM MoveHistory ORDER BY MovedAt DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MoveRecord
	for rows.Next() {
		var r MoveRecord
		if scanErr := rows.Scan(&r.ID, &r.MediaID, &r.FileActionId,
			&r.FromPath, &r.ToPath, &r.OriginalFileName, &r.NewFileName,
			&r.MovedAt, &r.IsReverted); scanErr != nil {
			return nil, scanErr
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// MoveInput holds fields needed to insert a move history record.
type MoveInput struct {
	MediaID      int64
	FileActionID int
	FromPath     string
	ToPath       string
	OrigName     string
	NewName      string
}

// InsertMoveHistory logs a move operation.
func (d *DB) InsertMoveHistory(input MoveInput) error {
	_, err := d.Exec(`
		INSERT INTO MoveHistory (MediaId, FileActionId, FromPath, ToPath, OriginalFileName, NewFileName)
		VALUES (?, ?, ?, ?, ?, ?)`, input.MediaID, input.FileActionID, input.FromPath, input.ToPath, input.OrigName, input.NewName)
	return err
}

// GetLastMove returns the latest non-reverted move.
func (d *DB) GetLastMove() (*MoveRecord, error) {
	row := d.QueryRow(`
		SELECT MoveHistoryId, MediaId, FileActionId, FromPath, ToPath,
		       OriginalFileName, NewFileName, IsReverted
		FROM MoveHistory WHERE IsReverted = 0 ORDER BY MovedAt DESC LIMIT 1`)
	r := &MoveRecord{}
	err := row.Scan(&r.ID, &r.MediaID, &r.FileActionId,
		&r.FromPath, &r.ToPath, &r.OriginalFileName, &r.NewFileName, &r.IsReverted)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// MarkMoveReverted marks a MoveHistory record as reverted.
func (d *DB) MarkMoveReverted(id int64) error {
	_, err := d.Exec("UPDATE MoveHistory SET IsReverted = 1 WHERE MoveHistoryId = ?", id)
	return err
}

// MarkMoveRestored marks a MoveHistory record as not reverted (redo).
func (d *DB) MarkMoveRestored(id int64) error {
	_, err := d.Exec("UPDATE MoveHistory SET IsReverted = 0 WHERE MoveHistoryId = ?", id)
	return err
}

// GetLastRevertedMove returns the most recent reverted move (for redo).
func (d *DB) GetLastRevertedMove() (*MoveRecord, error) {
	row := d.QueryRow(`
		SELECT MoveHistoryId, MediaId, FileActionId, FromPath, ToPath,
		       OriginalFileName, NewFileName, MovedAt, IsReverted
		FROM MoveHistory WHERE IsReverted = 1 ORDER BY MovedAt DESC LIMIT 1`)
	r := &MoveRecord{}
	err := row.Scan(&r.ID, &r.MediaID, &r.FileActionId,
		&r.FromPath, &r.ToPath, &r.OriginalFileName, &r.NewFileName, &r.MovedAt, &r.IsReverted)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// ScanRecord represents a row in ScanHistory.
type ScanRecord struct {
	ID           int64
	ScanFolderId int
	TotalFiles   int
	Movies       int
	TV           int
	NewFiles     int
	RemovedFiles int
	UpdatedFiles int
	ErrorCount   int
	DurationMs   int
	ScannedAt    string
}

// ListScanHistory returns recent scan history records.
func (d *DB) ListScanHistory(limit int) ([]ScanRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.Query(`
		SELECT ScanHistoryId, ScanFolderId, TotalFiles, MoviesFound, TvFound,
		       NewFiles, RemovedFiles, UpdatedFiles, ErrorCount, DurationMs, ScannedAt
		FROM ScanHistory
		ORDER BY ScannedAt DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ScanRecord
	for rows.Next() {
		var r ScanRecord
		if scanErr := rows.Scan(&r.ID, &r.ScanFolderId, &r.TotalFiles, &r.Movies, &r.TV,
			&r.NewFiles, &r.RemovedFiles, &r.UpdatedFiles, &r.ErrorCount, &r.DurationMs,
			&r.ScannedAt); scanErr != nil {
			return nil, scanErr
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ScanHistoryInput holds fields for a scan history record.
type ScanHistoryInput struct {
	ScanFolderID int
	TotalFiles   int
	Movies       int
	TV           int
	NewFiles     int
	Removed      int
	Updated      int
	Errors       int
	DurationMs   int
}

// InsertScanHistory logs a scan operation.
func (d *DB) InsertScanHistory(input ScanHistoryInput) error {
	_, err := d.Exec(`
		INSERT INTO ScanHistory (ScanFolderId, TotalFiles, MoviesFound, TvFound,
			NewFiles, RemovedFiles, UpdatedFiles, ErrorCount, DurationMs)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.ScanFolderID, input.TotalFiles, input.Movies, input.TV,
		input.NewFiles, input.Removed, input.Updated, input.Errors, input.DurationMs)
	return err
}

// ScanFolderRecord represents a row in ScanFolder.
type ScanFolderRecord struct {
	ID         int64
	FolderPath string
	IsActive   bool
	CreatedAt  string
	UpdatedAt  string
}

// UpsertScanFolder inserts or returns existing scan folder ID.
func (d *DB) UpsertScanFolder(folderPath string) (int64, error) {
	_, err := d.Exec("INSERT OR IGNORE INTO ScanFolder (FolderPath) VALUES (?)", folderPath)
	if err != nil {
		return 0, err
	}
	var id int64
	err = d.QueryRow("SELECT ScanFolderId FROM ScanFolder WHERE FolderPath = ?", folderPath).Scan(&id)
	return id, err
}

// ListScanFolders returns all registered scan folders.
func (d *DB) ListScanFolders(limit int) ([]ScanFolderRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.Query(`
		SELECT ScanFolderId, FolderPath, IsActive, CreatedAt, UpdatedAt
		FROM ScanFolder ORDER BY FolderPath ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ScanFolderRecord
	for rows.Next() {
		var r ScanFolderRecord
		if scanErr := rows.Scan(&r.ID, &r.FolderPath, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ListDistinctScanFolders returns unique folder paths from ScanFolder.
func (d *DB) ListDistinctScanFolders() ([]string, error) {
	rows, err := d.Query("SELECT FolderPath FROM ScanFolder ORDER BY FolderPath")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []string
	for rows.Next() {
		var f string
		if scanErr := rows.Scan(&f); scanErr != nil {
			return nil, scanErr
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}
