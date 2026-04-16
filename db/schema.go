// schema.go — Full PascalCase schema creation, seed data, and views.
package db

import (
	"fmt"

	"github.com/alimtvnetwork/movie-cli-v3/version"
)

// migrateSchema creates all tables, indexes, views, and seed data.
func (d *DB) migrateSchema() error {
	if err := d.createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	if err := d.seedFileActions(); err != nil {
		return fmt.Errorf("seed FileAction: %w", err)
	}
	if err := d.seedDefaultConfig(); err != nil {
		return fmt.Errorf("seed Config: %w", err)
	}
	if err := d.createViews(); err != nil {
		return fmt.Errorf("create views: %w", err)
	}
	// Always stamp current app version
	if err := d.SetConfig("AppVersion", version.Short()); err != nil {
		return fmt.Errorf("stamp app version: %w", err)
	}
	return nil
}

// createTables creates all PascalCase tables and indexes.
func (d *DB) createTables() error {
	_, err := d.Exec(`
	-- Lookup Tables
	CREATE TABLE IF NOT EXISTS Language (
		LanguageId INTEGER PRIMARY KEY AUTOINCREMENT,
		Code       TEXT NOT NULL UNIQUE,
		Name       TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS Genre (
		GenreId INTEGER PRIMARY KEY AUTOINCREMENT,
		Name    TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS Cast (
		CastId       INTEGER PRIMARY KEY AUTOINCREMENT,
		Name         TEXT NOT NULL,
		TmdbPersonId INTEGER UNIQUE
	);

	CREATE TABLE IF NOT EXISTS FileAction (
		FileActionId INTEGER PRIMARY KEY AUTOINCREMENT,
		Name         TEXT NOT NULL UNIQUE
	);

	-- Core Entities
	CREATE TABLE IF NOT EXISTS Collection (
		CollectionId     INTEGER PRIMARY KEY AUTOINCREMENT,
		TmdbCollectionId INTEGER NOT NULL UNIQUE,
		Name             TEXT NOT NULL,
		Overview         TEXT,
		PosterPath       TEXT,
		BackdropPath     TEXT,
		CreatedAt        TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS ScanFolder (
		ScanFolderId INTEGER PRIMARY KEY AUTOINCREMENT,
		FolderPath   TEXT NOT NULL UNIQUE,
		IsActive     BOOLEAN NOT NULL DEFAULT 1,
		CreatedAt    TEXT NOT NULL DEFAULT (datetime('now')),
		UpdatedAt    TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS ScanHistory (
		ScanHistoryId INTEGER PRIMARY KEY AUTOINCREMENT,
		ScanFolderId  INTEGER NOT NULL,
		TotalFiles    INTEGER NOT NULL DEFAULT 0,
		MoviesFound   INTEGER NOT NULL DEFAULT 0,
		TvFound       INTEGER NOT NULL DEFAULT 0,
		NewFiles      INTEGER NOT NULL DEFAULT 0,
		RemovedFiles  INTEGER NOT NULL DEFAULT 0,
		UpdatedFiles  INTEGER NOT NULL DEFAULT 0,
		ErrorCount    INTEGER NOT NULL DEFAULT 0,
		DurationMs    INTEGER NOT NULL DEFAULT 0,
		ScannedAt     TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (ScanFolderId) REFERENCES ScanFolder(ScanFolderId)
	);

	CREATE TABLE IF NOT EXISTS Media (
		MediaId          INTEGER PRIMARY KEY AUTOINCREMENT,
		Title            TEXT NOT NULL,
		CleanTitle       TEXT NOT NULL,
		Year             SMALLINT,
		Type             TEXT NOT NULL CHECK(Type IN ('movie', 'tv')),
		TmdbId           INTEGER UNIQUE,
		ImdbId           TEXT,
		Description      TEXT,
		ImdbRating       REAL,
		TmdbRating       REAL,
		Popularity       REAL,
		LanguageId       INTEGER,
		CollectionId     INTEGER,
		Director         TEXT,
		ThumbnailPath    TEXT,
		OriginalFileName TEXT,
		OriginalFilePath TEXT,
		CurrentFilePath  TEXT,
		FileExtension    TEXT,
		FileSizeMb       REAL,
		Runtime          INTEGER NOT NULL DEFAULT 0,
		Budget           INTEGER NOT NULL DEFAULT 0,
		Revenue          INTEGER NOT NULL DEFAULT 0,
		TrailerUrl       TEXT,
		Tagline          TEXT,
		ScanHistoryId    INTEGER,
		ScannedAt        TEXT NOT NULL DEFAULT (datetime('now')),
		UpdatedAt        TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (LanguageId) REFERENCES Language(LanguageId),
		FOREIGN KEY (CollectionId) REFERENCES Collection(CollectionId),
		FOREIGN KEY (ScanHistoryId) REFERENCES ScanHistory(ScanHistoryId)
	);

	-- Join Tables
	CREATE TABLE IF NOT EXISTS Director (
		DirectorId   INTEGER PRIMARY KEY AUTOINCREMENT,
		Name         TEXT NOT NULL UNIQUE,
		TmdbPersonId INTEGER UNIQUE
	);

	CREATE TABLE IF NOT EXISTS MediaDirector (
		MediaDirectorId INTEGER PRIMARY KEY AUTOINCREMENT,
		MediaId         INTEGER NOT NULL,
		DirectorId      INTEGER NOT NULL,
		UNIQUE (MediaId, DirectorId),
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE,
		FOREIGN KEY (DirectorId) REFERENCES Director(DirectorId)
	);

	CREATE TABLE IF NOT EXISTS MediaGenre (
		MediaGenreId INTEGER PRIMARY KEY AUTOINCREMENT,
		MediaId      INTEGER NOT NULL,
		GenreId      INTEGER NOT NULL,
		UNIQUE (MediaId, GenreId),
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE,
		FOREIGN KEY (GenreId) REFERENCES Genre(GenreId)
	);

	CREATE TABLE IF NOT EXISTS MediaCast (
		MediaCastId INTEGER PRIMARY KEY AUTOINCREMENT,
		MediaId     INTEGER NOT NULL,
		CastId      INTEGER NOT NULL,
		Role        TEXT,
		CastOrder   INTEGER,
		UNIQUE (MediaId, CastId),
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE,
		FOREIGN KEY (CastId) REFERENCES Cast(CastId)
	);

	CREATE TABLE IF NOT EXISTS Tag (
		TagId     INTEGER PRIMARY KEY AUTOINCREMENT,
		Name      TEXT NOT NULL UNIQUE,
		CreatedAt TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS MediaTag (
		MediaTagId INTEGER PRIMARY KEY AUTOINCREMENT,
		MediaId    INTEGER NOT NULL,
		TagId      INTEGER NOT NULL,
		CreatedAt  TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE (MediaId, TagId),
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE,
		FOREIGN KEY (TagId) REFERENCES Tag(TagId) ON DELETE CASCADE
	);

	-- History / Operations
	CREATE TABLE IF NOT EXISTS MoveHistory (
		MoveHistoryId    INTEGER PRIMARY KEY AUTOINCREMENT,
		MediaId          INTEGER NOT NULL,
		FileActionId     INTEGER NOT NULL,
		FromPath         TEXT NOT NULL,
		ToPath           TEXT NOT NULL,
		OriginalFileName TEXT,
		NewFileName      TEXT,
		IsReverted       BOOLEAN NOT NULL DEFAULT 0,
		MovedAt          TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId),
		FOREIGN KEY (FileActionId) REFERENCES FileAction(FileActionId)
	);

	CREATE TABLE IF NOT EXISTS ActionHistory (
		ActionHistoryId INTEGER PRIMARY KEY AUTOINCREMENT,
		FileActionId    INTEGER NOT NULL,
		MediaId         INTEGER,
		MediaSnapshot   TEXT,
		Detail          TEXT,
		BatchId         TEXT,
		IsReverted      BOOLEAN NOT NULL DEFAULT 0,
		CreatedAt       TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (FileActionId) REFERENCES FileAction(FileActionId),
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE SET NULL
	);

	-- Watchlist
	CREATE TABLE IF NOT EXISTS Watchlist (
		WatchlistId INTEGER PRIMARY KEY AUTOINCREMENT,
		MediaId     INTEGER,
		TmdbId      INTEGER NOT NULL UNIQUE,
		Title       TEXT NOT NULL,
		Year        SMALLINT,
		Type        TEXT CHECK(Type IN ('movie', 'tv')),
		Status      TEXT NOT NULL CHECK(Status IN ('to-watch', 'watched')) DEFAULT 'to-watch',
		AddedAt     TEXT NOT NULL DEFAULT (datetime('now')),
		WatchedAt   TEXT,
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE SET NULL
	);

	-- TV Tracking
	CREATE TABLE IF NOT EXISTS Season (
		SeasonId     INTEGER PRIMARY KEY AUTOINCREMENT,
		MediaId      INTEGER NOT NULL,
		SeasonNumber INTEGER NOT NULL,
		TmdbSeasonId INTEGER,
		Name         TEXT,
		Overview     TEXT,
		PosterPath   TEXT,
		AirDate      TEXT,
		EpisodeCount INTEGER NOT NULL DEFAULT 0,
		CreatedAt    TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE (MediaId, SeasonNumber),
		FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS Episode (
		EpisodeId      INTEGER PRIMARY KEY AUTOINCREMENT,
		SeasonId       INTEGER NOT NULL,
		EpisodeNumber  INTEGER NOT NULL,
		TmdbEpisodeId  INTEGER,
		Name           TEXT,
		Overview       TEXT,
		AirDate        TEXT,
		Runtime        INTEGER NOT NULL DEFAULT 0,
		StillPath      TEXT,
		VoteAvg        REAL NOT NULL DEFAULT 0,
		IsWatched      BOOLEAN NOT NULL DEFAULT 0,
		WatchedAt      TEXT,
		CreatedAt      TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE (SeasonId, EpisodeNumber),
		FOREIGN KEY (SeasonId) REFERENCES Season(SeasonId) ON DELETE CASCADE
	);

	-- Config
	CREATE TABLE IF NOT EXISTS Config (
		ConfigKey   TEXT PRIMARY KEY NOT NULL,
		ConfigValue TEXT NOT NULL
	);

	-- Error Log
	CREATE TABLE IF NOT EXISTS ErrorLog (
		ErrorLogId INTEGER PRIMARY KEY AUTOINCREMENT,
		Timestamp  TEXT NOT NULL,
		Level      TEXT NOT NULL CHECK(Level IN ('ERROR', 'WARN', 'INFO')),
		Source     TEXT NOT NULL,
		Function   TEXT,
		Command    TEXT,
		WorkDir    TEXT,
		Message    TEXT NOT NULL,
		StackTrace TEXT,
		CreatedAt  TEXT NOT NULL DEFAULT (datetime('now'))
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS IdxCast_TmdbPersonId       ON Cast(TmdbPersonId);
	CREATE INDEX IF NOT EXISTS IdxCollection_TmdbCollectionId ON Collection(TmdbCollectionId);
	CREATE INDEX IF NOT EXISTS IdxScanHistory_ScanFolderId ON ScanHistory(ScanFolderId);
	CREATE INDEX IF NOT EXISTS IdxMedia_TmdbId             ON Media(TmdbId);
	CREATE INDEX IF NOT EXISTS IdxMedia_Type               ON Media(Type);
	CREATE INDEX IF NOT EXISTS IdxMedia_LanguageId         ON Media(LanguageId);
	CREATE INDEX IF NOT EXISTS IdxMedia_CollectionId       ON Media(CollectionId);
	CREATE INDEX IF NOT EXISTS IdxMedia_ScanHistoryId      ON Media(ScanHistoryId);
	CREATE INDEX IF NOT EXISTS IdxDirector_TmdbPersonId    ON Director(TmdbPersonId);
	CREATE INDEX IF NOT EXISTS IdxMediaDirector_MediaId    ON MediaDirector(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaDirector_DirectorId ON MediaDirector(DirectorId);
	CREATE INDEX IF NOT EXISTS IdxMediaGenre_MediaId       ON MediaGenre(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaGenre_GenreId        ON MediaGenre(GenreId);
	CREATE INDEX IF NOT EXISTS IdxMediaCast_MediaId        ON MediaCast(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaCast_CastId         ON MediaCast(CastId);
	CREATE INDEX IF NOT EXISTS IdxMediaTag_MediaId         ON MediaTag(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaTag_TagId           ON MediaTag(TagId);
	CREATE INDEX IF NOT EXISTS IdxMoveHistory_MediaId      ON MoveHistory(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMoveHistory_FileActionId ON MoveHistory(FileActionId);
	CREATE INDEX IF NOT EXISTS IdxMoveHistory_IsReverted   ON MoveHistory(IsReverted);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_FileActionId ON ActionHistory(FileActionId);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_MediaId      ON ActionHistory(MediaId);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_BatchId      ON ActionHistory(BatchId);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_IsReverted   ON ActionHistory(IsReverted);
	CREATE INDEX IF NOT EXISTS IdxErrorLog_Level           ON ErrorLog(Level);
	CREATE INDEX IF NOT EXISTS IdxErrorLog_Command         ON ErrorLog(Command);
	CREATE INDEX IF NOT EXISTS IdxErrorLog_Timestamp       ON ErrorLog(Timestamp);
	CREATE INDEX IF NOT EXISTS IdxSeason_MediaId           ON Season(MediaId);
	CREATE INDEX IF NOT EXISTS IdxSeason_TmdbSeasonId      ON Season(TmdbSeasonId);
	CREATE INDEX IF NOT EXISTS IdxEpisode_SeasonId         ON Episode(SeasonId);
	CREATE INDEX IF NOT EXISTS IdxEpisode_TmdbEpisodeId    ON Episode(TmdbEpisodeId);
	CREATE INDEX IF NOT EXISTS IdxEpisode_IsWatched        ON Episode(IsWatched);
	`)
	return err
}

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
