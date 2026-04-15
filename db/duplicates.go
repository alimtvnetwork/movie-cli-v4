// duplicates.go — duplicate detection queries for the Media table.
package db

import "fmt"

// DuplicateGroup represents a set of media records that share a duplicate key.
type DuplicateGroup struct {
	Key   string
	Items []Media
}

// FindDuplicatesByTmdbID returns groups of media sharing the same TmdbId.
func (d *DB) FindDuplicatesByTmdbID() ([]DuplicateGroup, error) {
	rows, err := d.Query(`
		SELECT TmdbId FROM Media
		WHERE TmdbId > 0
		GROUP BY TmdbId
		HAVING COUNT(*) > 1
		ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning TmdbId: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var groups []DuplicateGroup
	for _, id := range ids {
		items, err := d.mediaByTmdbIDAll(id)
		if err != nil {
			continue
		}
		groups = append(groups, DuplicateGroup{Key: formatInt(id), Items: items})
	}
	return groups, nil
}

// FindDuplicatesByFileName returns groups sharing the same OriginalFileName.
func (d *DB) FindDuplicatesByFileName() ([]DuplicateGroup, error) {
	rows, err := d.Query(`
		SELECT OriginalFileName FROM Media
		WHERE OriginalFileName != ''
		GROUP BY OriginalFileName
		HAVING COUNT(*) > 1
		ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning OriginalFileName: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var groups []DuplicateGroup
	for _, name := range names {
		items, err := d.mediaByFileName(name)
		if err != nil {
			continue
		}
		groups = append(groups, DuplicateGroup{Key: name, Items: items})
	}
	return groups, nil
}

// FindDuplicatesByFileSize returns groups sharing the same FileSizeMb.
func (d *DB) FindDuplicatesByFileSize() ([]DuplicateGroup, error) {
	rows, err := d.Query(`
		SELECT FileSizeMb FROM Media
		WHERE FileSizeMb > 0
		GROUP BY FileSizeMb
		HAVING COUNT(*) > 1
		ORDER BY FileSizeMb DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sizes []float64
	for rows.Next() {
		var size float64
		if err := rows.Scan(&size); err != nil {
			return nil, fmt.Errorf("scanning FileSizeMb: %w", err)
		}
		sizes = append(sizes, size)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var groups []DuplicateGroup
	for _, size := range sizes {
		items, err := d.mediaByFileSize(size)
		if err != nil {
			continue
		}
		groups = append(groups, DuplicateGroup{Key: fmt.Sprintf("%.1f MB", size), Items: items})
	}
	return groups, nil
}

func (d *DB) mediaByTmdbIDAll(tmdbID int) ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE TmdbId = ? ORDER BY MediaId`, tmdbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

func (d *DB) mediaByFileName(name string) ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE OriginalFileName = ? ORDER BY MediaId`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

func (d *DB) mediaByFileSize(size float64) ([]Media, error) {
	rows, err := d.Query(`SELECT `+mediaColumns+`
		FROM Media WHERE FileSizeMb = ? ORDER BY MediaId`, size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaRows(rows)
}

func formatInt(n int) string {
	return fmt.Sprintf("%d", n)
}

// HumanSize formats megabytes into human-readable form.
func HumanSize(mb float64) string {
	switch {
	case mb >= 1024:
		return fmt.Sprintf("%.1f GB", mb/1024)
	case mb >= 1:
		return fmt.Sprintf("%.1f MB", mb)
	default:
		return fmt.Sprintf("%.0f KB", mb*1024)
	}
}
