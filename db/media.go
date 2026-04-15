package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
)

// mediaColumns is the standard SELECT column list for Media queries.
const mediaColumns = `MediaId, Title, CleanTitle, Year, Type,
	COALESCE(TmdbId, 0), COALESCE(ImdbId, ''), COALESCE(Description, ''),
	COALESCE(ImdbRating, 0), COALESCE(TmdbRating, 0), COALESCE(Popularity, 0),
	COALESCE(LanguageId, 0), COALESCE(CollectionId, 0),
	COALESCE(Director, ''), COALESCE(ThumbnailPath, ''),
	COALESCE(OriginalFileName, ''), COALESCE(OriginalFilePath, ''),
	COALESCE(CurrentFilePath, ''), COALESCE(FileExtension, ''),
	COALESCE(FileSizeMb, 0),
	COALESCE(Runtime, 0), COALESCE(Budget, 0), COALESCE(Revenue, 0),
	COALESCE(TrailerUrl, ''), COALESCE(Tagline, ''),
	COALESCE(ScanHistoryId, 0)`

// Media represents a row in the Media table.
type Media struct {
	Title            string
	CleanTitle       string
	Type             string // "movie" or "tv"
	ImdbID           string
	Description      string
	Director         string
	ThumbnailPath    string
	OriginalFileName string
	OriginalFilePath string
	CurrentFilePath  string
	FileExtension    string
	TrailerURL       string
	Tagline          string
	ID               int64
	Budget           int64
	Revenue          int64
	ImdbRating       float64
	TmdbRating       float64
	Popularity       float64
	FileSizeMb       float64
	Year             int
	TmdbID           int
	Runtime          int
	LanguageId       int
	CollectionId     int
	ScanHistoryId    int
}

// InsertMedia inserts a new media record and returns the ID.
func (d *DB) InsertMedia(m *Media) (int64, error) {
	var tmdbID interface{}
	if m.TmdbID > 0 {
		tmdbID = m.TmdbID
	}
	var langID interface{}
	if m.LanguageId > 0 {
		langID = m.LanguageId
	}
	var collID interface{}
	if m.CollectionId > 0 {
		collID = m.CollectionId
	}
	var scanID interface{}
	if m.ScanHistoryId > 0 {
		scanID = m.ScanHistoryId
	}

	res, err := d.Exec(`
		INSERT INTO Media (Title, CleanTitle, Year, Type, TmdbId, ImdbId,
			Description, ImdbRating, TmdbRating, Popularity, LanguageId, CollectionId,
			Director, ThumbnailPath, OriginalFileName, OriginalFilePath,
			CurrentFilePath, FileExtension, FileSizeMb,
			Runtime, Budget, Revenue, TrailerUrl, Tagline, ScanHistoryId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.Title, m.CleanTitle, m.Year, m.Type, tmdbID, m.ImdbID,
		m.Description, m.ImdbRating, m.TmdbRating, m.Popularity, langID, collID,
		m.Director, m.ThumbnailPath, m.OriginalFileName, m.OriginalFilePath,
		m.CurrentFilePath, m.FileExtension, m.FileSizeMb,
		m.Runtime, m.Budget, m.Revenue, m.TrailerURL, m.Tagline, scanID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateMediaByID updates an existing record by its primary key.
func (d *DB) UpdateMediaByID(m *Media) error {
	var tmdbID interface{}
	if m.TmdbID > 0 {
		tmdbID = m.TmdbID
	}
	var langID interface{}
	if m.LanguageId > 0 {
		langID = m.LanguageId
	}
	var collID interface{}
	if m.CollectionId > 0 {
		collID = m.CollectionId
	}
	_, err := d.Exec(`
		UPDATE Media SET Title=?, CleanTitle=?, Year=?, Type=?, TmdbId=?, ImdbId=?,
			Description=?, ImdbRating=?, TmdbRating=?, Popularity=?, LanguageId=?, CollectionId=?,
			Director=?, ThumbnailPath=?, CurrentFilePath=?,
			FileExtension=?, FileSizeMb=?,
			Runtime=?, Budget=?, Revenue=?, TrailerUrl=?, Tagline=?,
			UpdatedAt=datetime('now')
		WHERE MediaId=?`,
		m.Title, m.CleanTitle, m.Year, m.Type, tmdbID, m.ImdbID,
		m.Description, m.ImdbRating, m.TmdbRating, m.Popularity, langID, collID,
		m.Director, m.ThumbnailPath, m.CurrentFilePath,
		m.FileExtension, m.FileSizeMb,
		m.Runtime, m.Budget, m.Revenue, m.TrailerURL, m.Tagline,
		m.ID,
	)
	return err
}

// UpdateMediaByTmdbID updates an existing record matched by TmdbId.
func (d *DB) UpdateMediaByTmdbID(m *Media) error {
	var langID interface{}
	if m.LanguageId > 0 {
		langID = m.LanguageId
	}
	var collID interface{}
	if m.CollectionId > 0 {
		collID = m.CollectionId
	}
	_, err := d.Exec(`
		UPDATE Media SET Title=?, CleanTitle=?, Year=?, Type=?, ImdbId=?,
			Description=?, ImdbRating=?, TmdbRating=?, Popularity=?, LanguageId=?, CollectionId=?,
			Director=?, ThumbnailPath=?, CurrentFilePath=?,
			FileExtension=?, FileSizeMb=?,
			Runtime=?, Budget=?, Revenue=?, TrailerUrl=?, Tagline=?,
			UpdatedAt=datetime('now')
		WHERE TmdbId=?`,
		m.Title, m.CleanTitle, m.Year, m.Type, m.ImdbID,
		m.Description, m.ImdbRating, m.TmdbRating, m.Popularity, langID, collID,
		m.Director, m.ThumbnailPath, m.CurrentFilePath,
		m.FileExtension, m.FileSizeMb,
		m.Runtime, m.Budget, m.Revenue, m.TrailerURL, m.Tagline,
		m.TmdbID,
	)
	return err
}

// UpdateMediaPath updates the current file path.
func (d *DB) UpdateMediaPath(mediaID int64, newPath string) error {
	_, err := d.Exec("UPDATE Media SET CurrentFilePath = ?, UpdatedAt = datetime('now') WHERE MediaId = ?", newPath, mediaID)
	return err
}

// ListMedia returns paginated media records.
func (d *DB) ListMedia(offset, limit int) ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE OriginalFilePath != ''
		ORDER BY CleanTitle ASC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// SearchMedia searches by title (fuzzy via LIKE).
func (d *DB) SearchMedia(query string) ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE CleanTitle LIKE ? OR Title LIKE ?
		ORDER BY Popularity DESC LIMIT 20`, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// GetMediaByID returns a single media record.
func (d *DB) GetMediaByID(id int64) (*Media, error) {
	row := d.QueryRow(`SELECT `+mediaColumns+` FROM Media WHERE MediaId = ?`, id)
	return scanMediaRow(row)
}

// GetMediaByTmdbID returns a media record by its TMDb ID.
func (d *DB) GetMediaByTmdbID(tmdbID int) (*Media, error) {
	row := d.QueryRow(`SELECT `+mediaColumns+` FROM Media WHERE TmdbId = ?`, tmdbID)
	return scanMediaRow(row)
}

// CountMedia returns total count of scan-indexed items.
func (d *DB) CountMedia(mediaType string) (int, error) {
	var count int
	var err error
	if mediaType == "" {
		err = d.QueryRow("SELECT COUNT(*) FROM Media WHERE OriginalFilePath != ''").Scan(&count)
	} else {
		err = d.QueryRow("SELECT COUNT(*) FROM Media WHERE Type = ? AND OriginalFilePath != ''", mediaType).Scan(&count)
	}
	return count, err
}

// ListAllMedia returns all media records that have a file path.
func (d *DB) ListAllMedia() ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE OriginalFilePath != ''
		ORDER BY CleanTitle ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// GetMediaWithMissingData returns entries with empty genre/rating/description.
func (d *DB) GetMediaWithMissingData() ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE OriginalFilePath != ''
		AND (COALESCE(TmdbRating, 0) = 0 OR COALESCE(Description, '') = '')
		ORDER BY CleanTitle ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// GetMediaByScanDir returns media whose OriginalFilePath starts with the given directory.
func (d *DB) GetMediaByScanDir(scanDir string) ([]Media, error) {
	prefix := scanDir
	if prefix != "" && prefix[len(prefix)-1] != '/' && prefix[len(prefix)-1] != '\\' {
		prefix += string([]byte{filepath.Separator})
	}
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE OriginalFilePath LIKE ?
		ORDER BY CleanTitle ASC`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// DeleteMediaByIDs deletes multiple media records by their IDs.
func (d *DB) DeleteMediaByIDs(ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, id := range ids {
		if _, err := tx.Exec("DELETE FROM Media WHERE MediaId = ?", id); err != nil {
			tx.Rollback()
			return deleted, err
		}
		deleted++
	}
	return deleted, tx.Commit()
}

// FileSizeStats returns total, largest, and smallest file size in MB.
func (d *DB) FileSizeStats() (total float64, largest float64, smallest float64, err error) {
	err = d.QueryRow(`
		SELECT COALESCE(SUM(FileSizeMb), 0),
		       COALESCE(MAX(FileSizeMb), 0),
		       COALESCE(MIN(NULLIF(FileSizeMb, 0)), 0)
		FROM Media WHERE FileSizeMb > 0`).Scan(&total, &largest, &smallest)
	return
}

// MediaByType returns media filtered by type.
func (d *DB) MediaByType(mediaType string, limit int) ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE Type = ? ORDER BY Popularity DESC LIMIT ?`, mediaType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// scanMediaRow scans a single media row.
func scanMediaRow(row *sql.Row) (*Media, error) {
	m := &Media{}
	err := row.Scan(&m.ID, &m.Title, &m.CleanTitle, &m.Year, &m.Type,
		&m.TmdbID, &m.ImdbID, &m.Description, &m.ImdbRating, &m.TmdbRating,
		&m.Popularity, &m.LanguageId, &m.CollectionId,
		&m.Director, &m.ThumbnailPath,
		&m.OriginalFileName, &m.OriginalFilePath, &m.CurrentFilePath,
		&m.FileExtension, &m.FileSizeMb,
		&m.Runtime, &m.Budget, &m.Revenue, &m.TrailerURL, &m.Tagline,
		&m.ScanHistoryId)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// scanMediaRows scans multiple media rows from a query result.
func scanMediaRows(rows *sql.Rows) ([]Media, error) {
	var list []Media
	for rows.Next() {
		var m Media
		if err := rows.Scan(&m.ID, &m.Title, &m.CleanTitle, &m.Year, &m.Type,
			&m.TmdbID, &m.ImdbID, &m.Description, &m.ImdbRating, &m.TmdbRating,
			&m.Popularity, &m.LanguageId, &m.CollectionId,
			&m.Director, &m.ThumbnailPath,
			&m.OriginalFileName, &m.OriginalFilePath, &m.CurrentFilePath,
			&m.FileExtension, &m.FileSizeMb,
			&m.Runtime, &m.Budget, &m.Revenue, &m.TrailerURL, &m.Tagline,
			&m.ScanHistoryId); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}
