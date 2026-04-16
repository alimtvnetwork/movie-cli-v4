// seed.go — seed data for FileAction and default Config.
package db

import "fmt"

// seedFileActions inserts the 14 predefined FileAction types.
func (d *DB) seedFileActions() error {
	actions := []string{
		"Move", "Rename", "Delete", "Popout", "Restore",
		"ScanAdd", "ScanRemove", "RescanUpdate",
		"TagAdd", "TagRemove",
		"WatchlistAdd", "WatchlistRemove", "WatchlistStatusChange",
		"ConfigChange",
	}
	for _, name := range actions {
		if _, err := d.Exec("INSERT OR IGNORE INTO FileAction (Name) VALUES (?)", name); err != nil {
			return fmt.Errorf("seed FileAction %q: %w", name, err)
		}
	}
	return nil
}

// seedDefaultConfig inserts default config values if not already present.
func (d *DB) seedDefaultConfig() error {
	defaults := [][2]string{
		{"MoviesDir", "~/Movies"},
		{"TvDir", "~/TVShows"},
		{"ArchiveDir", "~/Archive"},
		{"ScanDir", "~/Downloads"},
		{"PageSize", "20"},
	}
	for _, kv := range defaults {
		if _, err := d.Exec("INSERT OR IGNORE INTO Config (ConfigKey, ConfigValue) VALUES (?, ?)", kv[0], kv[1]); err != nil {
			return fmt.Errorf("seed config %q: %w", kv[0], err)
		}
	}
	return nil
}
