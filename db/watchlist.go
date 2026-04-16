// watchlist.go — CRUD for the Watchlist table.
package db

import "database/sql"

// WatchlistEntry represents a row in the Watchlist table.
type WatchlistEntry struct {
	ID        int64
	MediaID   sql.NullInt64
	TmdbID    int
	Title     string
	Year      int
	Type      string // MediaTypeMovie or MediaTypeTV
	Status    string // WatchStatusToWatch or WatchStatusWatched
	AddedAt   string
	WatchedAt sql.NullString
}

// WatchlistInput holds fields for adding to the watchlist.
type WatchlistInput struct {
	TmdbID    int
	Title     string
	Year      int
	MediaType string
	MediaID   int64
}

// AddToWatchlist inserts or updates a watchlist entry as "to-watch".
func (d *DB) AddToWatchlist(input WatchlistInput) error {
	var mid sql.NullInt64
	if input.MediaID > 0 {
		mid = sql.NullInt64{Int64: input.MediaID, Valid: true}
	}
	_, err := d.Exec(`
		INSERT INTO Watchlist (TmdbId, Title, Year, Type, Status, MediaId)
		VALUES (?, ?, ?, ?, 'to-watch', ?)
		ON CONFLICT(TmdbId) DO UPDATE SET
			Title = excluded.Title,
			Year  = excluded.Year,
			Type  = excluded.Type,
			MediaId = COALESCE(excluded.MediaId, Watchlist.MediaId)`,
		input.TmdbID, input.Title, input.Year, input.MediaType, mid)
	return err
}

// MarkWatched updates a watchlist entry to "watched".
func (d *DB) MarkWatched(tmdbID int) error {
	_, err := d.Exec(`
		UPDATE Watchlist SET Status = 'watched', WatchedAt = datetime('now')
		WHERE TmdbId = ?`, tmdbID)
	return err
}

// MarkToWatch updates a watchlist entry back to "to-watch".
func (d *DB) MarkToWatch(tmdbID int) error {
	_, err := d.Exec(`
		UPDATE Watchlist SET Status = 'to-watch', WatchedAt = NULL
		WHERE TmdbId = ?`, tmdbID)
	return err
}

// RemoveFromWatchlist deletes a watchlist entry.
func (d *DB) RemoveFromWatchlist(tmdbID int) error {
	_, err := d.Exec("DELETE FROM Watchlist WHERE TmdbId = ?", tmdbID)
	return err
}

// ListWatchlist returns entries filtered by status.
func (d *DB) ListWatchlist(status string) ([]WatchlistEntry, error) {
	query := `SELECT WatchlistId, MediaId, TmdbId, Title, Year, Type, Status, AddedAt, WatchedAt
		FROM Watchlist ORDER BY AddedAt DESC`
	if status != "" {
		query = `SELECT WatchlistId, MediaId, TmdbId, Title, Year, Type, Status, AddedAt, WatchedAt
		FROM Watchlist WHERE Status = ? ORDER BY AddedAt DESC`
	}

	var rows *sql.Rows
	var err error
	if status == "" {
		rows, err = d.Query(query)
	}
	if status != "" {
		rows, err = d.Query(query, status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []WatchlistEntry
	for rows.Next() {
		var e WatchlistEntry
		if err := rows.Scan(&e.ID, &e.MediaID, &e.TmdbID, &e.Title, &e.Year,
			&e.Type, &e.Status, &e.AddedAt, &e.WatchedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// GetWatchlistByTmdbID returns a single watchlist entry.
func (d *DB) GetWatchlistByTmdbID(tmdbID int) (*WatchlistEntry, error) {
	row := d.QueryRow(`
		SELECT WatchlistId, MediaId, TmdbId, Title, Year, Type, Status, AddedAt, WatchedAt
		FROM Watchlist WHERE TmdbId = ?`, tmdbID)
	var e WatchlistEntry
	err := row.Scan(&e.ID, &e.MediaID, &e.TmdbID, &e.Title, &e.Year,
		&e.Type, &e.Status, &e.AddedAt, &e.WatchedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
