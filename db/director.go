// director.go — Director normalization via M:N table.
package db

import (
	"strings"
)

// LinkMediaDirectors normalizes a comma-separated director string into
// the Director + MediaDirector M:N tables.
func (d *DB) LinkMediaDirectors(mediaID int64, directorCSV string) error {
	names := splitDirectors(directorCSV)
	for _, name := range names {
		dirID, err := d.ensureDirector(name)
		if err != nil {
			return err
		}
		_, err = d.Exec(
			"INSERT OR IGNORE INTO MediaDirector (MediaId, DirectorId) VALUES (?, ?)",
			mediaID, dirID)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReplaceMediaDirectors replaces all director links for a media entry.
func (d *DB) ReplaceMediaDirectors(mediaID int64, directorCSV string) {
	d.Exec("DELETE FROM MediaDirector WHERE MediaId = ?", mediaID)
	d.LinkMediaDirectors(mediaID, directorCSV)
}

// DirectorsByMediaID returns comma-separated director names for a media entry.
func (d *DB) DirectorsByMediaID(mediaID int64) string {
	rows, err := d.Query(`
		SELECT d.Name FROM Director d
		JOIN MediaDirector md ON md.DirectorId = d.DirectorId
		WHERE md.MediaId = ?
		ORDER BY d.Name`, mediaID)
	if err != nil {
		return ""
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

// ensureDirector inserts or finds a director by name, returns DirectorId.
func (d *DB) ensureDirector(name string) (int64, error) {
	name = strings.TrimSpace(name)
	res, err := d.Exec("INSERT OR IGNORE INTO Director (Name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if id > 0 {
		return id, nil
	}
	var dirID int64
	err = d.QueryRow("SELECT DirectorId FROM Director WHERE Name = ?", name).Scan(&dirID)
	return dirID, err
}

// splitDirectors splits a comma-separated director string into trimmed names.
func splitDirectors(csv string) []string {
	parts := strings.Split(csv, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
