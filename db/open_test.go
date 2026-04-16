package db

import (
	"testing"
)

// ─── Schema / Migration ─────────────────────────────────────

func TestMigrate(t *testing.T) {
	d := openTestDB(t)
	for _, table := range []string{
		"Media", "MoveHistory", "Config", "ScanHistory", "ScanFolder",
		"Tag", "MediaTag", "Watchlist", "ErrorLog", "ActionHistory",
		"Genre", "Cast", "MediaGenre", "MediaCast", "Language",
		"FileAction", "Collection",
	} {
		var name string
		err := d.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	d := openTestDB(t)
	val, err := d.GetConfig("MoviesDir")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if val != "~/Movies" {
		t.Errorf("MoviesDir = %q, want ~/Movies", val)
	}
}

func TestAppVersionInConfig(t *testing.T) {
	d := openTestDB(t)
	val, err := d.GetConfig("AppVersion")
	if err != nil {
		t.Fatalf("get AppVersion: %v", err)
	}
	if val == "" {
		t.Error("AppVersion should not be empty")
	}
}

// ─── Scan History ───────────────────────────────────────────

func TestScanHistory(t *testing.T) {
	d := openTestDB(t)
	folderId, err := d.UpsertScanFolder("/downloads")
	if err != nil {
		t.Fatalf("upsert scan folder: %v", err)
	}
	err = d.InsertScanHistory(ScanHistoryInput{
		ScanFolderID: int(folderId), TotalFiles: 50, Movies: 30, TV: 20,
		NewFiles: 10, Removed: 0, Updated: 0, Errors: 0, DurationMs: 500,
	})
	if err != nil {
		t.Fatalf("insert scan history: %v", err)
	}
}

// ─── Top Genres ─────────────────────────────────────────────

func TestTopGenres(t *testing.T) {
	d := openTestDB(t)
	id1 := seedMedia(t, d, "A", 1)
	id2 := seedMedia(t, d, "B", 2)

	d.Exec("INSERT INTO Genre (Name) VALUES ('Action')")
	d.Exec("INSERT INTO Genre (Name) VALUES ('Drama')")
	d.Exec("INSERT INTO Genre (Name) VALUES ('Comedy')")

	d.Exec("INSERT INTO MediaGenre (MediaId, GenreId) SELECT ?, GenreId FROM Genre WHERE Name = 'Action'", id1)
	d.Exec("INSERT INTO MediaGenre (MediaId, GenreId) SELECT ?, GenreId FROM Genre WHERE Name = 'Drama'", id1)
	d.Exec("INSERT INTO MediaGenre (MediaId, GenreId) SELECT ?, GenreId FROM Genre WHERE Name = 'Action'", id2)
	d.Exec("INSERT INTO MediaGenre (MediaId, GenreId) SELECT ?, GenreId FROM Genre WHERE Name = 'Comedy'", id2)

	genres, err := d.TopGenres(10)
	if err != nil {
		t.Fatalf("genres: %v", err)
	}
	if genres["Action"] != 2 {
		t.Errorf("Action = %d, want 2", genres["Action"])
	}
	if genres["Drama"] != 1 {
		t.Errorf("Drama = %d", genres["Drama"])
	}
}

// ─── FileAction Seed ────────────────────────────────────────

func TestFileActionSeeded(t *testing.T) {
	d := openTestDB(t)
	var count int
	if err := d.QueryRow("SELECT COUNT(*) FROM FileAction").Scan(&count); err != nil {
		t.Fatalf("count FileAction: %v", err)
	}
	if count != 14 {
		t.Errorf("FileAction count = %d, want 14", count)
	}
}

// ─── Views ──────────────────────────────────────────────────

func TestViewsExist(t *testing.T) {
	d := openTestDB(t)
	for _, view := range []string{
		"VwMediaDetail", "VwMediaGenreList", "VwMediaCastList",
		"VwMediaFull", "VwMoveHistoryDetail", "VwActionHistoryDetail",
		"VwScanHistoryDetail", "VwMediaTag",
	} {
		var name string
		err := d.QueryRow("SELECT name FROM sqlite_master WHERE type='view' AND name=?", view).Scan(&name)
		if err != nil {
			t.Errorf("view %q not found: %v", view, err)
		}
	}
}
