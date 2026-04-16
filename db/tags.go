// tags.go — Tag lookup + MediaTag join table helpers.
package db

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
)

// TagCount holds a tag name and its usage count.
type TagCount struct {
	Tag   string
	Count int
}

// AddTag inserts a tag for a media item. Creates the tag if it doesn't exist,
// then links it via MediaTag.
func (d *DB) AddTag(mediaID int, tag string) error {
	_, err := d.Exec("INSERT OR IGNORE INTO Tag (Name) VALUES (?)", tag)
	if err != nil {
		return apperror.Wrap("insert tag %q", tag, err)
	}

	_, err = d.Exec(`
		INSERT INTO MediaTag (MediaId, TagId)
		SELECT ?, TagId FROM Tag WHERE Name = ?`,
		mediaID, tag,
	)
	return err
}

// RemoveTag deletes a tag link from a media item.
func (d *DB) RemoveTag(mediaID int, tag string) (bool, error) {
	result, err := d.Exec(
		`DELETE FROM MediaTag WHERE MediaId = ? AND TagId = (SELECT TagId FROM Tag WHERE Name = ?)`,
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
	rows, err := d.Query(`
		SELECT t.Name FROM Tag t
		INNER JOIN MediaTag mt ON t.TagId = mt.TagId
		WHERE mt.MediaId = ?
		ORDER BY t.Name`, mediaID)
	if err != nil {
		return nil, apperror.Wrap("query tags", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, apperror.Wrap("scan tag", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// GetAllTagCounts returns all unique tags with their usage count.
func (d *DB) GetAllTagCounts() ([]TagCount, error) {
	rows, err := d.Query(`
		SELECT t.Name, COUNT(*) as cnt
		FROM Tag t
		INNER JOIN MediaTag mt ON t.TagId = mt.TagId
		GROUP BY t.Name
		ORDER BY cnt DESC, t.Name ASC`)
	if err != nil {
		return nil, apperror.Wrap("query tag counts", err)
	}
	defer rows.Close()

	var counts []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, apperror.Wrap("scan tag count", err)
		}
		counts = append(counts, tc)
	}
	return counts, rows.Err()
}
