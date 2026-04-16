// Package db manages the SQLite database for the movie CLI.
package db

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/alimtvnetwork/movie-cli-v3/apperror"
	_ "modernc.org/sqlite"
)

const dbFile = "mahin.db"

// legacyDBFiles are old database files that should be removed on startup.
var legacyDBFiles = []string{"movie.db", "movie.db-wal", "movie.db-shm"}

// DB wraps the sql.DB connection.
type DB struct {
	*sql.DB
	BasePath string // path to data directory
}

// exeDir returns the directory where the running binary is located.
func exeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", apperror.Wrap("cannot locate executable", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", apperror.Wrap("cannot resolve symlinks for executable", err)
	}
	return filepath.Dir(exe), nil
}

// removeLegacyDB deletes old database files (movie.db) if they exist.
func removeLegacyDB(base string) {
	for _, name := range legacyDBFiles {
		p := filepath.Join(base, name)
		if _, err := os.Stat(p); err == nil {
			os.Remove(p) // best-effort
		}
	}
}

// Open opens (or creates) the SQLite database and runs migrations.
// If a legacy database (movie.db) is found, it is deleted.
// The app version is stored in Config on every startup.
func Open() (*DB, error) {
	binDir, dirErr := exeDir()
	if dirErr != nil {
		return nil, dirErr
	}

	base := filepath.Join(binDir, "data")
	dirs := []string{
		base,
		filepath.Join(base, "json", string(MediaTypeMovie)),
		filepath.Join(base, "json", string(MediaTypeTV)),
		filepath.Join(base, "json", "history"),
		filepath.Join(base, "thumbnails"),
		filepath.Join(base, "config"),
		filepath.Join(base, "log"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, apperror.Wrapf(err, "cannot create directory %s", d)
		}
	}

	// Remove legacy database before opening new one
	removeLegacyDB(base)

	dbPath := filepath.Join(base, dbFile)
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, apperror.Wrap("cannot open database", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, apperror.Wrap("cannot set WAL mode", err)
	}

	// Set busy timeout — wait up to 5s for locked DB
	if _, err := conn.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		conn.Close()
		return nil, apperror.Wrap("cannot set busy_timeout", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, apperror.Wrap("cannot enable foreign keys", err)
	}

	d := &DB{DB: conn, BasePath: base}
	if err := d.migrateSchema(); err != nil {
		conn.Close()
		return nil, apperror.Wrap("migration failed", err)
	}

	return d, nil
}
