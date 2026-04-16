// cleanup.go — find stale media entries where the file no longer exists on disk.
package db

import (
	"fmt"
	"os"
)

// StaleEntry represents a media record whose file is missing from disk.
type StaleEntry struct {
	Media    Media
	FilePath string
}

// FindStaleEntries returns media records where CurrentFilePath or
// OriginalFilePath no longer exists on disk.
func (d *DB) FindStaleEntries(limit int) ([]StaleEntry, error) {
	rows, err := d.Query(`
		SELECT `+mediaColumns+`
		FROM Media
		WHERE OriginalFilePath != ''
		ORDER BY CleanTitle ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	all, err := scanMediaRows(rows)
	if err != nil {
		return nil, err
	}

	var stale []StaleEntry
	for _, m := range all {
		path := m.CurrentFilePath
		if path == "" {
			path = m.OriginalFilePath
		}
		if path == "" {
			continue
		}
		_, statErr := os.Stat(path)
		if statErr == nil {
			continue
		}
		if os.IsNotExist(statErr) {
			stale = append(stale, StaleEntry{Media: m, FilePath: path})
			continue
		}
		fmt.Fprintf(os.Stderr, "⚠️  Cannot stat %s: %v\n", path, statErr)
	}
	return stale, nil
}

// DeleteMedia removes a media record by ID.
func (d *DB) DeleteMedia(id int64) error {
	_, err := d.Exec("DELETE FROM Media WHERE MediaId = ?", id)
	return err
}
