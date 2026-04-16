// genre.go — Genre and MediaGenre (many-to-many) operations.
package db

import (
	"fmt"
	"strings"
)

// EnsureGenre inserts a genre if it doesn't exist, returns its GenreId.
func (d *DB) EnsureGenre(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("genre name is empty")
	}
	_, err := d.Exec("INSERT OR IGNORE INTO Genre (Name) VALUES (?)", name)
	if err != nil {
		return 0, fmt.Errorf("insert genre %q: %w", name, err)
	}
	var id int64
	err = d.QueryRow("SELECT GenreId FROM Genre WHERE Name = ?", name).Scan(&id)
	return id, err
}

// LinkMediaGenres parses a comma-separated genre string, ensures each genre
// exists, and creates MediaGenre rows. Existing links are preserved (INSERT OR IGNORE).
func (d *DB) LinkMediaGenres(mediaID int64, genreCSV string) error {
	if mediaID == 0 || genreCSV == "" {
		return nil
	}
	for _, raw := range strings.Split(genreCSV, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		genreID, err := d.EnsureGenre(name)
		if err != nil {
			return fmt.Errorf("ensure genre %q: %w", name, err)
		}
		if _, err := d.Exec(
			"INSERT OR IGNORE INTO MediaGenre (MediaId, GenreId) VALUES (?, ?)",
			mediaID, genreID,
		); err != nil {
			return fmt.Errorf("link media %d genre %d: %w", mediaID, genreID, err)
		}
	}
	return nil
}

// ReplaceMediaGenres removes all existing genre links for a media and re-links.
func (d *DB) ReplaceMediaGenres(mediaID int64, genreCSV string) error {
	if _, err := d.Exec("DELETE FROM MediaGenre WHERE MediaId = ?", mediaID); err != nil {
		return fmt.Errorf("clear genres for media %d: %w", mediaID, err)
	}
	return d.LinkMediaGenres(mediaID, genreCSV)
}

// GetMediaGenres returns a comma-separated genre string for a media.
func (d *DB) GetMediaGenres(mediaID int64) (string, error) {
	rows, err := d.Query(`
		SELECT g.Name FROM MediaGenre mg
		INNER JOIN Genre g ON mg.GenreId = g.GenreId
		WHERE mg.MediaId = ?
		ORDER BY g.Name`, mediaID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return "", err
		}
		names = append(names, n)
	}
	return strings.Join(names, ", "), rows.Err()
}

// MediaHasGenres returns true if the media has at least one genre linked.
func (d *DB) MediaHasGenres(mediaID int64) (bool, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM MediaGenre WHERE MediaId = ?", mediaID).Scan(&count)
	return count > 0, err
}

// SearchMediaByGenre returns media that have the given genre (exact match on genre name).
func (d *DB) SearchMediaByGenre(genreName string) ([]Media, error) {
	rows, err := d.Query(`
		SELECT `+mediaColumns+`
		FROM Media
		WHERE MediaId IN (
			SELECT mg.MediaId FROM MediaGenre mg
			INNER JOIN Genre g ON mg.GenreId = g.GenreId
			WHERE g.Name = ?
		)
		ORDER BY Popularity DESC`, genreName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// SearchMediaByGenreLike returns media matching a genre name pattern (LIKE).
func (d *DB) SearchMediaByGenreLike(pattern string) ([]Media, error) {
	rows, err := d.Query(`
		SELECT `+mediaColumns+`
		FROM Media
		WHERE MediaId IN (
			SELECT mg.MediaId FROM MediaGenre mg
			INNER JOIN Genre g ON mg.GenreId = g.GenreId
			WHERE g.Name LIKE ?
		)
		ORDER BY Popularity DESC`, "%"+pattern+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

// ListGenres returns all genres with their media count.
func (d *DB) ListGenres() (map[string]int, error) {
	rows, err := d.Query(`
		SELECT g.Name, COUNT(mg.MediaId) as cnt
		FROM Genre g
		LEFT JOIN MediaGenre mg ON g.GenreId = mg.GenreId
		GROUP BY g.GenreId, g.Name
		ORDER BY g.Name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var name string
		var cnt int
		if err := rows.Scan(&name, &cnt); err != nil {
			return nil, err
		}
		result[name] = cnt
	}
	return result, rows.Err()
}
