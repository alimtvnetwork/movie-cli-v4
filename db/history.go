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
	IsUndone         bool
}

// ListMoveHistory returns all move records ordered by most recent first.
func (d *DB) ListMoveHistory(limit int) ([]MoveRecord, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := d.Query(`
		SELECT MoveHistoryId, MediaId, FileActionId, FromPath, ToPath,
		       OriginalFileName, NewFileName, MovedAt, IsUndone
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
			&r.MovedAt, &r.IsUndone); scanErr != nil {
			return nil, scanErr
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// InsertMoveHistory logs a move operation.
func (d *DB) InsertMoveHistory(mediaID int64, fileActionId int, fromPath, toPath, origName, newName string) error {
	_, err := d.Exec(`
		INSERT INTO MoveHistory (MediaId, FileActionId, FromPath, ToPath, OriginalFileName, NewFileName)
		VALUES (?, ?, ?, ?, ?, ?)`, mediaID, fileActionId, fromPath, toPath, origName, newName)
	return err
}

// GetLastMove returns the latest un-undone move.
func (d *DB) GetLastMove() (*MoveRecord, error) {
	row := d.QueryRow(`
		SELECT MoveHistoryId, MediaId, FileActionId, FromPath, ToPath,
		       OriginalFileName, NewFileName, IsUndone
		FROM MoveHistory WHERE IsUndone = 0 ORDER BY MovedAt DESC LIMIT 1`)
	r := &MoveRecord{}
	err := row.Scan(&r.ID, &r.MediaID, &r.FileActionId,
		&r.FromPath, &r.ToPath, &r.OriginalFileName, &r.NewFileName, &r.IsUndone)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// MarkMoveUndone marks a MoveHistory record as undone.
func (d *DB) MarkMoveUndone(id int64) error {
	_, err := d.Exec("UPDATE MoveHistory SET IsUndone = 1 WHERE MoveHistoryId = ?", id)
	return err
}

// MarkMoveRedone marks a MoveHistory record as not undone (redo).
func (d *DB) MarkMoveRedone(id int64) error {
	_, err := d.Exec("UPDATE MoveHistory SET IsUndone = 0 WHERE MoveHistoryId = ?", id)
	return err
}

// GetLastUndoneMove returns the most recent undone move (for redo).
func (d *DB) GetLastUndoneMove() (*MoveRecord, error) {
	row := d.QueryRow(`
		SELECT MoveHistoryId, MediaId, FileActionId, FromPath, ToPath,
		       OriginalFileName, NewFileName, MovedAt, IsUndone
		FROM MoveHistory WHERE IsUndone = 1 ORDER BY MovedAt DESC LIMIT 1`)
	r := &MoveRecord{}
	err := row.Scan(&r.ID, &r.MediaID, &r.FileActionId,
		&r.FromPath, &r.ToPath, &r.OriginalFileName, &r.NewFileName, &r.MovedAt, &r.IsUndone)
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

// InsertScanHistory logs a scan operation.
func (d *DB) InsertScanHistory(scanFolderId int, total, movies, tv, newFiles, removed, updated, errors, durationMs int) error {
	_, err := d.Exec(`
		INSERT INTO ScanHistory (ScanFolderId, TotalFiles, MoviesFound, TvFound,
			NewFiles, RemovedFiles, UpdatedFiles, ErrorCount, DurationMs)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		scanFolderId, total, movies, tv, newFiles, removed, updated, errors, durationMs)
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
