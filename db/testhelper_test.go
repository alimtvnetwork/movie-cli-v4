package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// openTestDB creates an in-memory SQLite database with the full schema.
func openTestDB(t *testing.T) *DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		t.Fatalf("enable foreign keys: %v", err)
	}
	d := &DB{DB: conn, BasePath: t.TempDir()}
	if err := d.migrateSchema(); err != nil {
		conn.Close()
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return d
}

// seedMedia inserts a sample media record and returns the ID.
func seedMedia(t *testing.T, d *DB, title string, tmdbID int) int64 {
	t.Helper()
	id, err := d.InsertMedia(&Media{
		Title:            title,
		CleanTitle:       title,
		Year:             2024,
		Type:             "movie",
		TmdbID:           tmdbID,
		OriginalFileName: title + ".mkv",
		OriginalFilePath: "/movies/" + title + ".mkv",
		CurrentFilePath:  "/movies/" + title + ".mkv",
		FileExtension:    ".mkv",
		FileSizeMb:       700.0, // 700 MB
	})
	if err != nil {
		t.Fatalf("seed media %q: %v", title, err)
	}
	return id
}
