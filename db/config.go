package db

// GetConfig returns a config value by key.
func (d *DB) GetConfig(key string) (string, error) {
	var val string
	err := d.QueryRow("SELECT ConfigValue FROM Config WHERE ConfigKey = ?", key).Scan(&val)
	return val, err
}

// SetConfig sets a config value (upsert).
func (d *DB) SetConfig(key, value string) error {
	_, err := d.Exec("INSERT OR REPLACE INTO Config (ConfigKey, ConfigValue) VALUES (?, ?)", key, value)
	return err
}
