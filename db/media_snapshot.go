// media_snapshot.go — JSON snapshot helpers for media records.
package db

import (
	"encoding/json"
	"fmt"
)

// MediaToJSON serialises a Media record to JSON for ActionHistory snapshots.
func MediaToJSON(m *Media) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshal media snapshot: %w", err)
	}
	return string(data), nil
}

// MediaFromJSON deserialises a JSON snapshot back into a Media struct.
func MediaFromJSON(snapshot string) (*Media, error) {
	var m Media
	if err := json.Unmarshal([]byte(snapshot), &m); err != nil {
		return nil, fmt.Errorf("unmarshal media snapshot: %w", err)
	}
	return &m, nil
}

// DeleteMediaByID deletes a single media record by primary key.
func (d *DB) DeleteMediaByID(id int64) error {
	_, err := d.Exec("DELETE FROM Media WHERE MediaId = ?", id)
	if err != nil {
		return fmt.Errorf("delete media %d: %w", id, err)
	}
	return nil
}
