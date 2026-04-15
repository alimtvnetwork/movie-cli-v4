// tags.go — Tag lookup + MediaTag join table helpers.
//
// Tags are now a standalone lookup table linked to media via media_tags join.
package db

import "fmt"

// TagCount holds a tag name and its usage count.
type TagCount struct {
	Tag   string
	Count int
}

// AddTag inserts a tag for a media item. Creates the tag if it doesn't exist,
// then links it via media_tags. Returns UNIQUE constraint error if already linked.
func (d *DB) AddTag(mediaID int, tag string) error {
	// Upsert tag into lookup table
	_, err := d.Exec(
		`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tag,
	)
	if err != nil {
		return fmt.Errorf("insert tag %q: %w", tag, err)
	}

	// Link via join table
	_, err = d.Exec(
		`INSERT INTO media_tags (media_id, tag_id)
		 SELECT ?, id FROM tags WHERE name = ?`,
		mediaID, tag,
	)
	return err
}

// RemoveTag deletes a tag link from a media item.
// Returns (true, nil) if deleted, (false, nil) if link didn't exist.
func (d *DB) RemoveTag(mediaID int, tag string) (bool, error) {
	result, err := d.Exec(
		`DELETE FROM media_tags WHERE media_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ?)`,
		mediaID, tag,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// GetTagsByMediaID returns all tag names for a specific media item.
func (d *DB) GetTagsByMediaID(mediaID int) ([]string, error) {
	rows, err := d.Query(
		`SELECT t.name FROM tags t
		 INNER JOIN media_tags mt ON t.id = mt.tag_id
		 WHERE mt.media_id = ?
		 ORDER BY t.name`,
		mediaID,
	)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// GetAllTagCounts returns all unique tags with their usage count,
// ordered by count descending.
func (d *DB) GetAllTagCounts() ([]TagCount, error) {
	rows, err := d.Query(
		`SELECT t.name, COUNT(*) as cnt
		 FROM tags t
		 INNER JOIN media_tags mt ON t.id = mt.tag_id
		 GROUP BY t.name
		 ORDER BY cnt DESC, t.name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query tag counts: %w", err)
	}
	defer rows.Close()

	var counts []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, fmt.Errorf("scan tag count: %w", err)
		}
		counts = append(counts, tc)
	}
	return counts, rows.Err()
}
