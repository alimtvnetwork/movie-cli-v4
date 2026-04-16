// Package db manages the SQLite database for the movie CLI.
package db

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"database/sql"
	"os"
	"path/filepath"

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
	if err := createDataDirs(base); err != nil {
		return nil, err
	}

	removeLegacyDB(base)

	conn, err := openAndConfigureDB(base)
	if err != nil {
		return nil, err
	}

	d := &DB{DB: conn, BasePath: base}
	if err := d.migrateSchema(); err != nil {
		conn.Close()
		return nil, apperror.Wrap("migration failed", err)
	}

	return d, nil
}

func createDataDirs(base string) error {
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
			return apperror.Wrapf(err, "cannot create directory %s", d)
		}
	}
	return nil
}

func openAndConfigureDB(base string) (*sql.DB, error) {
	dbPath := filepath.Join(base, dbFile)
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, apperror.Wrap("cannot open database", err)
	}

	pragmas := []struct{ stmt, errMsg string }{
		{"PRAGMA journal_mode=WAL", "cannot set WAL mode"},
		{"PRAGMA busy_timeout = 5000", "cannot set busy_timeout"},
		{"PRAGMA foreign_keys = ON", "cannot enable foreign keys"},
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p.stmt); err != nil {
			conn.Close()
			return nil, apperror.Wrap(p.errMsg, err)
		}
	}

	return conn, nil
}
