// media_snapshot.go — JSON snapshot helpers for media records.
package db

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"encoding/json"
)

// MediaToJSON serialises a Media record to JSON for ActionHistory snapshots.
func MediaToJSON(m *Media) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", apperror.Wrap("marshal media snapshot", err)
	}
	return string(data), nil
}

// MediaFromJSON deserialises a JSON snapshot back into a Media struct.
func MediaFromJSON(snapshot string) (*Media, error) {
	var m Media
	if err := json.Unmarshal([]byte(snapshot), &m); err != nil {
		return nil, apperror.Wrap("unmarshal media snapshot", err)
	}
	return &m, nil
}

// DeleteMediaByID deletes a single media record by primary key.
func (d *DB) DeleteMediaByID(id int64) error {
	_, err := d.Exec("DELETE FROM Media WHERE MediaId = ?", id)
	if err != nil {
		return apperror.Wrapf(err, "delete media %d", id)
	}
	return nil
}
