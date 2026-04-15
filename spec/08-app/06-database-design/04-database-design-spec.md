# Database Design Specification

**Version:** 1.0.0  
**Updated:** 2026-04-15  
**Status:** Active  
**Scope:** Complete database design for the Movie CLI (`mahin`)

---

## 1. Overview

The Movie CLI uses **SQLite** with the **Split DB** pattern — multiple small database files per bounded context. All naming follows **PascalCase** for tables and columns, with `{TableName}Id` primary keys.

### 1.1 Split DB Layout

| Database File | Bounded Context | Description |
|---------------|----------------|-------------|
| `media.db` | Media & Operations | Core media data, lookups, tags, scan tracking, file operations, action history |
| `watchlist.db` | Watch Tracking | User watchlist independent of local media library |
| `config.db` | Configuration | Key-value CLI settings |
| `error-log.db` | Error Logging | Structured error/warning log entries |

### 1.2 Data Folder Structure

```
<cli-binary-location>/
└── data/
    ├── media.db
    ├── watchlist.db
    ├── config.db
    ├── error-log.db
    ├── config/
    │   └── (CLI configuration files)
    └── log/
        ├── log.txt       — general application log
        └── error.log     — error-only log (see error handling spec)
```

> Path resolved via `os.Executable()` + `filepath.EvalSymlinks()` at runtime. No environment variables needed.

---

## 2. Naming Conventions

| Object | Convention | Example |
|--------|-----------|---------|
| Table names | PascalCase, singular or plural per domain | `Media`, `Genre`, `MediaCast` |
| Column names | PascalCase | `CleanTitle`, `FileSizeMb` |
| Primary key | `{TableName}Id` INTEGER AUTOINCREMENT | `MediaId`, `GenreId` |
| Foreign key column | Same name as referenced PK | `LanguageId` references `Language.LanguageId` |
| Boolean columns | `Is` or `Has` prefix, positive only, NOT NULL DEFAULT | `IsActive`, `IsUndone` |
| Index names | `Idx{Table}_{Column}` | `IdxMedia_TmdbId` |
| View names | PascalCase with `Vw` prefix | `VwMediaDetail` |
| Timestamps | TEXT with ISO 8601 / RFC 3339 format | `CreatedAt`, `ScannedAt` |

> Full reference: [Database Naming Conventions](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/01-naming-conventions.md)

---

## 3. Table Definitions — `media.db`

### 3.1 Language (Lookup)

**Purpose:** Normalized language codes and names.  
**Expected volume:** ~200 rows (ISO 639-1 languages)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| LanguageId | INTEGER | PK, AUTOINCREMENT | Primary key |
| Code | TEXT | NOT NULL, UNIQUE | ISO 639-1 code (e.g., `en`, `ms`, `ja`) |
| Name | TEXT | NOT NULL | Human-readable name (e.g., `English`) |

```sql
CREATE TABLE Language (
    LanguageId INTEGER PRIMARY KEY AUTOINCREMENT,
    Code       TEXT NOT NULL UNIQUE,
    Name       TEXT NOT NULL
);
```

---

### 3.2 Genre (Lookup)

**Purpose:** Normalized genre names. Sourced from TMDb genre list.  
**Expected volume:** ~30 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| GenreId | INTEGER | PK, AUTOINCREMENT | Primary key |
| Name | TEXT | NOT NULL, UNIQUE | Genre name (e.g., `Action`, `Comedy`) |

```sql
CREATE TABLE Genre (
    GenreId INTEGER PRIMARY KEY AUTOINCREMENT,
    Name    TEXT NOT NULL UNIQUE
);
```

---

### 3.3 Cast (Lookup)

**Purpose:** Cast members (actors/directors). Sourced from TMDb credits.  
**Expected volume:** ~10,000 rows (grows with library)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| CastId | INTEGER | PK, AUTOINCREMENT | Primary key |
| Name | TEXT | NOT NULL | Person's name |
| TmdbPersonId | INTEGER | UNIQUE, NULLABLE | TMDb person ID for deduplication |

```sql
CREATE TABLE Cast (
    CastId       INTEGER PRIMARY KEY AUTOINCREMENT,
    Name         TEXT NOT NULL,
    TmdbPersonId INTEGER UNIQUE
);

CREATE INDEX IdxCast_TmdbPersonId ON Cast(TmdbPersonId);
```

---

### 3.4 FileAction (Lookup)

**Purpose:** Predefined action types for history tracking. Seeded during migration.  
**Expected volume:** 14 rows (fixed set)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| FileActionId | INTEGER | PK, AUTOINCREMENT | Primary key |
| Name | TEXT | NOT NULL, UNIQUE | Action type name |

```sql
CREATE TABLE FileAction (
    FileActionId INTEGER PRIMARY KEY AUTOINCREMENT,
    Name         TEXT NOT NULL UNIQUE
);

-- Seed data (inserted during migration)
INSERT INTO FileAction (Name) VALUES
    ('Move'),
    ('Rename'),
    ('Delete'),
    ('Popout'),
    ('Restore'),
    ('ScanAdd'),
    ('ScanRemove'),
    ('RescanUpdate'),
    ('TagAdd'),
    ('TagRemove'),
    ('WatchlistAdd'),
    ('WatchlistRemove'),
    ('WatchlistStatusChange'),
    ('ConfigChange');
```

#### FileAction Enum (Go Code)

```go
type FileActionType int

const (
    FileActionMove                FileActionType = 1
    FileActionRename              FileActionType = 2
    FileActionDelete              FileActionType = 3
    FileActionPopout              FileActionType = 4
    FileActionRestore             FileActionType = 5
    FileActionScanAdd             FileActionType = 6
    FileActionScanRemove          FileActionType = 7
    FileActionRescanUpdate        FileActionType = 8
    FileActionTagAdd              FileActionType = 9
    FileActionTagRemove           FileActionType = 10
    FileActionWatchlistAdd        FileActionType = 11
    FileActionWatchlistRemove     FileActionType = 12
    FileActionWatchlistStatusChange FileActionType = 13
    FileActionConfigChange        FileActionType = 14
)
```

> Enum values match database row IDs. IDs are stable — never reorder or delete.

---

### 3.5 ScanFolder (Root Entity)

**Purpose:** Registered folders that the CLI scans for media files. Serves as the root entity for all scan operations.  
**Expected volume:** ~5-20 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| ScanFolderId | INTEGER | PK, AUTOINCREMENT | Primary key |
| FolderPath | TEXT | NOT NULL, UNIQUE | Absolute path to the scan folder |
| IsActive | BOOLEAN | NOT NULL, DEFAULT 1 | Whether this folder is actively scanned |
| CreatedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | When folder was first registered |
| UpdatedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | Last modification time |

```sql
CREATE TABLE ScanFolder (
    ScanFolderId INTEGER PRIMARY KEY AUTOINCREMENT,
    FolderPath   TEXT NOT NULL UNIQUE,
    IsActive     BOOLEAN NOT NULL DEFAULT 1,
    CreatedAt    TEXT NOT NULL DEFAULT (datetime('now')),
    UpdatedAt    TEXT NOT NULL DEFAULT (datetime('now'))
);
```

---

### 3.6 ScanHistory

**Purpose:** Detailed log of every scan operation. Connected to ScanFolder as parent and to Media as discoverer.  
**Expected volume:** ~500-2,000 rows (grows with usage)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| ScanHistoryId | INTEGER | PK, AUTOINCREMENT | Primary key |
| ScanFolderId | INTEGER | FK, NOT NULL | Which folder was scanned |
| TotalFiles | INTEGER | DEFAULT 0 | Total media files found in folder |
| MoviesFound | INTEGER | DEFAULT 0 | Number of movie files found |
| TvFound | INTEGER | DEFAULT 0 | Number of TV show files found |
| NewFiles | INTEGER | DEFAULT 0 | Files added in this scan |
| RemovedFiles | INTEGER | DEFAULT 0 | Files removed (no longer on disk) |
| UpdatedFiles | INTEGER | DEFAULT 0 | Files with updated metadata |
| ErrorCount | INTEGER | DEFAULT 0 | Number of errors during scan |
| DurationMs | INTEGER | DEFAULT 0 | Scan duration in milliseconds |
| ScannedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | When the scan was executed |

```sql
CREATE TABLE ScanHistory (
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

CREATE INDEX IdxScanHistory_ScanFolderId ON ScanHistory(ScanFolderId);
```

---

### 3.7 Media (Core Entity)

**Purpose:** Core media metadata — one row per scanned media file. Connected to ScanHistory to track which scan discovered it.  
**Expected volume:** ~1,000-50,000 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| MediaId | INTEGER | PK, AUTOINCREMENT | Primary key |
| Title | TEXT | NOT NULL | Display title |
| CleanTitle | TEXT | NOT NULL | Cleaned/normalized title for matching |
| Year | SMALLINT | NULLABLE | Release year |
| Type | TEXT | NOT NULL, CHECK | `movie` or `tv` — enum `MediaType` in code |
| TmdbId | INTEGER | UNIQUE, NULLABLE | TMDb ID for API lookups |
| ImdbId | TEXT | NULLABLE | IMDb ID |
| Description | TEXT | NULLABLE | Plot summary |
| ImdbRating | REAL | NULLABLE | IMDb rating (0.0-10.0) |
| TmdbRating | REAL | NULLABLE | TMDb rating (0.0-10.0) |
| Popularity | REAL | NULLABLE | TMDb popularity score |
| LanguageId | INTEGER | FK, NULLABLE | Original language |
| Director | TEXT | NULLABLE | Primary director name |
| ThumbnailPath | TEXT | NULLABLE | Local path to cached poster image |
| OriginalFileName | TEXT | NULLABLE | File name at time of scan |
| OriginalFilePath | TEXT | NULLABLE | Full path at time of scan |
| CurrentFilePath | TEXT | NULLABLE | Current file location (updated on move/rename) |
| FileExtension | TEXT | NULLABLE | e.g., `.mkv`, `.mp4` |
| FileSizeMb | REAL | NULLABLE | File size in **megabytes** |
| Runtime | INTEGER | DEFAULT 0 | Runtime in minutes |
| Budget | INTEGER | DEFAULT 0 | Production budget (USD) |
| Revenue | INTEGER | DEFAULT 0 | Box office revenue (USD) |
| TrailerUrl | TEXT | NULLABLE | YouTube/TMDb trailer URL |
| Tagline | TEXT | NULLABLE | Movie tagline |
| ScanHistoryId | INTEGER | FK, NULLABLE | Which scan discovered this file |
| ScannedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | When first scanned |
| UpdatedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | Last metadata update |

```sql
CREATE TABLE Media (
    MediaId         INTEGER PRIMARY KEY AUTOINCREMENT,
    Title           TEXT NOT NULL,
    CleanTitle      TEXT NOT NULL,
    Year            SMALLINT,
    Type            TEXT NOT NULL CHECK(Type IN ('movie', 'tv')),
    TmdbId          INTEGER UNIQUE,
    ImdbId          TEXT,
    Description     TEXT,
    ImdbRating      REAL,
    TmdbRating      REAL,
    Popularity      REAL,
    LanguageId      INTEGER,
    Director        TEXT,
    ThumbnailPath   TEXT,
    OriginalFileName TEXT,
    OriginalFilePath TEXT,
    CurrentFilePath TEXT,
    FileExtension   TEXT,
    FileSizeMb      REAL,
    Runtime         INTEGER NOT NULL DEFAULT 0,
    Budget          INTEGER NOT NULL DEFAULT 0,
    Revenue         INTEGER NOT NULL DEFAULT 0,
    TrailerUrl      TEXT,
    Tagline         TEXT,
    ScanHistoryId   INTEGER,
    ScannedAt       TEXT NOT NULL DEFAULT (datetime('now')),
    UpdatedAt       TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (LanguageId) REFERENCES Language(LanguageId),
    FOREIGN KEY (ScanHistoryId) REFERENCES ScanHistory(ScanHistoryId)
);

CREATE INDEX IdxMedia_TmdbId ON Media(TmdbId);
CREATE INDEX IdxMedia_Type ON Media(Type);
CREATE INDEX IdxMedia_LanguageId ON Media(LanguageId);
CREATE INDEX IdxMedia_ScanHistoryId ON Media(ScanHistoryId);
```

#### MediaType Enum (Go Code)

```go
type MediaType string

const (
    MediaTypeMovie MediaType = "movie"
    MediaTypeTv    MediaType = "tv"
)

func (m MediaType) IsEqual(other MediaType) bool {
    return m == other
}
```

---

### 3.8 MediaGenre (Join Table — 1-N)

**Purpose:** Links media to genres. One media can have many genres.  
**Expected volume:** ~3,000-150,000 rows (avg 3 genres per media)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| MediaGenreId | INTEGER | PK, AUTOINCREMENT | Primary key |
| MediaId | INTEGER | FK, NOT NULL | References Media |
| GenreId | INTEGER | FK, NOT NULL | References Genre |

```sql
CREATE TABLE MediaGenre (
    MediaGenreId INTEGER PRIMARY KEY AUTOINCREMENT,
    MediaId      INTEGER NOT NULL,
    GenreId      INTEGER NOT NULL,
    UNIQUE (MediaId, GenreId),
    FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE,
    FOREIGN KEY (GenreId) REFERENCES Genre(GenreId)
);

CREATE INDEX IdxMediaGenre_MediaId ON MediaGenre(MediaId);
CREATE INDEX IdxMediaGenre_GenreId ON MediaGenre(GenreId);
```

---

### 3.9 MediaCast (Join Table — N-M)

**Purpose:** Links media to cast members with role and ordering. Many-to-many relationship.  
**Expected volume:** ~10,000-500,000 rows (avg 10 cast per media)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| MediaCastId | INTEGER | PK, AUTOINCREMENT | Primary key |
| MediaId | INTEGER | FK, NOT NULL | References Media |
| CastId | INTEGER | FK, NOT NULL | References Cast |
| Role | TEXT | NULLABLE | Character name played |
| CastOrder | INTEGER | NULLABLE | Billing order (1 = lead) |

```sql
CREATE TABLE MediaCast (
    MediaCastId INTEGER PRIMARY KEY AUTOINCREMENT,
    MediaId     INTEGER NOT NULL,
    CastId      INTEGER NOT NULL,
    Role        TEXT,
    CastOrder   INTEGER,
    UNIQUE (MediaId, CastId),
    FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE,
    FOREIGN KEY (CastId) REFERENCES Cast(CastId)
);

CREATE INDEX IdxMediaCast_MediaId ON MediaCast(MediaId);
CREATE INDEX IdxMediaCast_CastId ON MediaCast(CastId);
```

---

### 3.10 Tag

**Purpose:** User-assigned tags per media item.  
**Expected volume:** ~500-5,000 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| TagId | INTEGER | PK, AUTOINCREMENT | Primary key |
| MediaId | INTEGER | FK, NOT NULL | References Media |
| Name | TEXT | NOT NULL | Tag text |
| CreatedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | When tag was added |

```sql
CREATE TABLE Tag (
    TagId     INTEGER PRIMARY KEY AUTOINCREMENT,
    MediaId   INTEGER NOT NULL,
    Name      TEXT NOT NULL,
    CreatedAt TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (MediaId, Name),
    FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE
);

CREATE INDEX IdxTag_MediaId ON Tag(MediaId);
```

---

### 3.11 MoveHistory

**Purpose:** Tracks all file move/rename operations with undo support. Each entry references a FileAction type.  
**Expected volume:** ~500-10,000 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| MoveHistoryId | INTEGER | PK, AUTOINCREMENT | Primary key |
| MediaId | INTEGER | FK, NOT NULL | Which media file was moved |
| FileActionId | INTEGER | FK, NOT NULL | Action type (Move, Rename, Popout) |
| FromPath | TEXT | NOT NULL | Source path before operation |
| ToPath | TEXT | NOT NULL | Destination path after operation |
| OriginalFileName | TEXT | NULLABLE | File name before operation |
| NewFileName | TEXT | NULLABLE | File name after operation |
| IsUndone | BOOLEAN | NOT NULL, DEFAULT 0 | Whether this operation has been undone |
| MovedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | When the operation occurred |

```sql
CREATE TABLE MoveHistory (
    MoveHistoryId    INTEGER PRIMARY KEY AUTOINCREMENT,
    MediaId          INTEGER NOT NULL,
    FileActionId     INTEGER NOT NULL,
    FromPath         TEXT NOT NULL,
    ToPath           TEXT NOT NULL,
    OriginalFileName TEXT,
    NewFileName      TEXT,
    IsUndone         BOOLEAN NOT NULL DEFAULT 0,
    MovedAt          TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (MediaId) REFERENCES Media(MediaId),
    FOREIGN KEY (FileActionId) REFERENCES FileAction(FileActionId)
);

CREATE INDEX IdxMoveHistory_MediaId ON MoveHistory(MediaId);
CREATE INDEX IdxMoveHistory_FileActionId ON MoveHistory(FileActionId);
CREATE INDEX IdxMoveHistory_IsUndone ON MoveHistory(IsUndone);
```

---

### 3.12 ActionHistory

**Purpose:** Unified audit log for all reversible operations beyond file moves (scan adds, deletes, restores, metadata updates).  
**Expected volume:** ~1,000-50,000 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| ActionHistoryId | INTEGER | PK, AUTOINCREMENT | Primary key |
| FileActionId | INTEGER | FK, NOT NULL | Action type from FileAction lookup |
| MediaId | INTEGER | FK, NULLABLE | Affected media (NULL if deleted) |
| MediaSnapshot | TEXT | NULLABLE | Full JSON snapshot of media record before change |
| Detail | TEXT | NULLABLE | Human-readable description |
| BatchId | TEXT | NULLABLE | Groups related actions (one scan = one batch) |
| IsUndone | BOOLEAN | NOT NULL, DEFAULT 0 | Whether this action has been undone |
| CreatedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | When the action occurred |

```sql
CREATE TABLE ActionHistory (
    ActionHistoryId INTEGER PRIMARY KEY AUTOINCREMENT,
    FileActionId    INTEGER NOT NULL,
    MediaId         INTEGER,
    MediaSnapshot   TEXT,
    Detail          TEXT,
    BatchId         TEXT,
    IsUndone        BOOLEAN NOT NULL DEFAULT 0,
    CreatedAt       TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (FileActionId) REFERENCES FileAction(FileActionId),
    FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE SET NULL
);

CREATE INDEX IdxActionHistory_FileActionId ON ActionHistory(FileActionId);
CREATE INDEX IdxActionHistory_MediaId ON ActionHistory(MediaId);
CREATE INDEX IdxActionHistory_BatchId ON ActionHistory(BatchId);
CREATE INDEX IdxActionHistory_IsUndone ON ActionHistory(IsUndone);
```

---

## 4. Table Definitions — `watchlist.db`

### 4.1 Watchlist

**Purpose:** User's to-watch and watched list. References TMDb IDs. Optional cross-DB reference to Media (not FK-enforced).  
**Expected volume:** ~100-1,000 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| WatchlistId | INTEGER | PK, AUTOINCREMENT | Primary key |
| MediaId | INTEGER | NULLABLE | Cross-DB ref to media.db Media (no FK) |
| TmdbId | INTEGER | NOT NULL, UNIQUE | TMDb ID |
| Title | TEXT | NOT NULL | Display title |
| Year | SMALLINT | NULLABLE | Release year |
| Type | TEXT | CHECK | `movie` or `tv` — enum `MediaType` |
| Status | TEXT | CHECK | `to-watch` or `watched` — enum `WatchStatus` |
| AddedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | When added to watchlist |
| WatchedAt | TEXT | NULLABLE | When marked as watched |

```sql
CREATE TABLE Watchlist (
    WatchlistId INTEGER PRIMARY KEY AUTOINCREMENT,
    MediaId     INTEGER,
    TmdbId      INTEGER NOT NULL UNIQUE,
    Title       TEXT NOT NULL,
    Year        SMALLINT,
    Type        TEXT CHECK(Type IN ('movie', 'tv')),
    Status      TEXT NOT NULL CHECK(Status IN ('to-watch', 'watched')) DEFAULT 'to-watch',
    AddedAt     TEXT NOT NULL DEFAULT (datetime('now')),
    WatchedAt   TEXT
);
```

#### WatchStatus Enum (Go Code)

```go
type WatchStatus string

const (
    WatchStatusToWatch WatchStatus = "to-watch"
    WatchStatusWatched WatchStatus = "watched"
)
```

> **Cross-DB note:** `MediaId` stores the ID from `media.db` but has no foreign key constraint. Validated at the application layer per Split DB rules.

---

## 5. Table Definitions — `config.db`

### 5.1 Config

**Purpose:** Key-value store for CLI settings (TMDb API key, default directories, page size, etc.).  
**Expected volume:** ~10-30 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| ConfigKey | TEXT | PK | Setting key (e.g., `TmdbApiKey`, `DefaultPageSize`) |
| ConfigValue | TEXT | NOT NULL | Setting value |

```sql
CREATE TABLE Config (
    ConfigKey   TEXT PRIMARY KEY NOT NULL,
    ConfigValue TEXT NOT NULL
);
```

---

## 6. Table Definitions — `error-log.db`

### 6.1 ErrorLog

**Purpose:** Structured error/warning log entries. See [Error Handling Spec](../04-error-handling-spec.md) for logging behavior.  
**Expected volume:** ~100-10,000 rows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| ErrorLogId | INTEGER | PK, AUTOINCREMENT | Primary key |
| Timestamp | TEXT | NOT NULL | ISO 8601 timestamp of the event |
| Level | TEXT | NOT NULL, CHECK | `ERROR`, `WARN`, or `INFO` — enum `LogLevel` |
| Source | TEXT | NOT NULL | Source module/package |
| Function | TEXT | NULLABLE | Function name where error occurred |
| Command | TEXT | NULLABLE | CLI command being executed |
| WorkDir | TEXT | NULLABLE | Working directory at time of error |
| Message | TEXT | NOT NULL | Error message |
| StackTrace | TEXT | NULLABLE | Full stack trace if available |
| CreatedAt | TEXT | DEFAULT CURRENT_TIMESTAMP | DB insertion time |

```sql
CREATE TABLE ErrorLog (
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

CREATE INDEX IdxErrorLog_Level ON ErrorLog(Level);
CREATE INDEX IdxErrorLog_Command ON ErrorLog(Command);
CREATE INDEX IdxErrorLog_Timestamp ON ErrorLog(Timestamp);
```

#### LogLevel Enum (Go Code)

```go
type LogLevel string

const (
    LogLevelError LogLevel = "ERROR"
    LogLevelWarn  LogLevel = "WARN"
    LogLevelInfo  LogLevel = "INFO"
)
```

---

## 7. Database Views

All views are created during migration and live in `media.db`. Application queries MUST use views instead of direct table joins. Views use the `Vw` prefix.

> Full reference: [ORM and Views](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/03-orm-and-views.md)

### 7.1 VwMediaDetail

**Purpose:** Media with resolved language name.

```sql
CREATE VIEW VwMediaDetail AS
SELECT
    m.MediaId,
    m.Title,
    m.CleanTitle,
    m.Year,
    m.Type,
    m.TmdbId,
    m.ImdbId,
    m.Description,
    m.ImdbRating,
    m.TmdbRating,
    m.Popularity,
    l.Code        AS LanguageCode,
    l.Name        AS LanguageName,
    m.Director,
    m.ThumbnailPath,
    m.OriginalFileName,
    m.OriginalFilePath,
    m.CurrentFilePath,
    m.FileExtension,
    m.FileSizeMb,
    m.Runtime,
    m.Budget,
    m.Revenue,
    m.TrailerUrl,
    m.Tagline,
    m.ScanHistoryId,
    m.ScannedAt,
    m.UpdatedAt
FROM Media m
LEFT JOIN Language l ON m.LanguageId = l.LanguageId;
```

### 7.2 VwMediaGenreList

**Purpose:** Media with all associated genre names.

```sql
CREATE VIEW VwMediaGenreList AS
SELECT
    mg.MediaGenreId,
    mg.MediaId,
    g.GenreId,
    g.Name AS GenreName
FROM MediaGenre mg
INNER JOIN Genre g ON mg.GenreId = g.GenreId;
```

### 7.3 VwMediaCastList

**Purpose:** Media with all associated cast members, roles, and ordering.

```sql
CREATE VIEW VwMediaCastList AS
SELECT
    mc.MediaCastId,
    mc.MediaId,
    c.CastId,
    c.Name         AS CastName,
    c.TmdbPersonId,
    mc.Role,
    mc.CastOrder
FROM MediaCast mc
INNER JOIN Cast c ON mc.CastId = c.CastId;
```

### 7.4 VwMediaFull

**Purpose:** Full media detail with language, aggregated genres (comma-separated), and aggregated cast (comma-separated). Primary view for display commands.

```sql
CREATE VIEW VwMediaFull AS
SELECT
    m.MediaId,
    m.Title,
    m.CleanTitle,
    m.Year,
    m.Type,
    m.TmdbId,
    m.ImdbId,
    m.Description,
    m.ImdbRating,
    m.TmdbRating,
    m.Popularity,
    l.Code         AS LanguageCode,
    l.Name         AS LanguageName,
    m.Director,
    m.ThumbnailPath,
    m.CurrentFilePath,
    m.FileExtension,
    m.FileSizeMb,
    m.Runtime,
    m.TrailerUrl,
    m.Tagline,
    m.ScannedAt,
    m.UpdatedAt,
    COALESCE(
        (SELECT GROUP_CONCAT(g.Name, ', ')
         FROM MediaGenre mg
         INNER JOIN Genre g ON mg.GenreId = g.GenreId
         WHERE mg.MediaId = m.MediaId), ''
    ) AS Genres,
    COALESCE(
        (SELECT GROUP_CONCAT(c.Name, ', ')
         FROM MediaCast mc
         INNER JOIN Cast c ON mc.CastId = c.CastId
         WHERE mc.MediaId = m.MediaId
         ORDER BY mc.CastOrder), ''
    ) AS CastList
FROM Media m
LEFT JOIN Language l ON m.LanguageId = l.LanguageId;
```

### 7.5 VwMoveHistoryDetail

**Purpose:** Move history with media title and action type name.

```sql
CREATE VIEW VwMoveHistoryDetail AS
SELECT
    mh.MoveHistoryId,
    mh.MediaId,
    m.Title        AS MediaTitle,
    fa.Name        AS ActionName,
    mh.FromPath,
    mh.ToPath,
    mh.OriginalFileName,
    mh.NewFileName,
    mh.IsUndone,
    mh.MovedAt
FROM MoveHistory mh
INNER JOIN Media m ON mh.MediaId = m.MediaId
INNER JOIN FileAction fa ON mh.FileActionId = fa.FileActionId;
```

### 7.6 VwActionHistoryDetail

**Purpose:** Action history with media title and action type name.

```sql
CREATE VIEW VwActionHistoryDetail AS
SELECT
    ah.ActionHistoryId,
    ah.MediaId,
    m.Title        AS MediaTitle,
    fa.Name        AS ActionName,
    ah.MediaSnapshot,
    ah.Detail,
    ah.BatchId,
    ah.IsUndone,
    ah.CreatedAt
FROM ActionHistory ah
INNER JOIN FileAction fa ON ah.FileActionId = fa.FileActionId
LEFT JOIN Media m ON ah.MediaId = m.MediaId;
```

### 7.7 VwScanHistoryDetail

**Purpose:** Scan history with folder path.

```sql
CREATE VIEW VwScanHistoryDetail AS
SELECT
    sh.ScanHistoryId,
    sh.ScanFolderId,
    sf.FolderPath,
    sf.IsActive    AS FolderIsActive,
    sh.TotalFiles,
    sh.MoviesFound,
    sh.TvFound,
    sh.NewFiles,
    sh.RemovedFiles,
    sh.UpdatedFiles,
    sh.ErrorCount,
    sh.DurationMs,
    sh.ScannedAt
FROM ScanHistory sh
INNER JOIN ScanFolder sf ON sh.ScanFolderId = sf.ScanFolderId;
```

### 7.8 VwMediaTag

**Purpose:** Media with associated tags.

```sql
CREATE VIEW VwMediaTag AS
SELECT
    t.TagId,
    t.MediaId,
    m.Title AS MediaTitle,
    t.Name  AS TagName,
    t.CreatedAt
FROM Tag t
INNER JOIN Media m ON t.MediaId = m.MediaId;
```

---

## 8. Enum Types Summary

All enum-like columns use TEXT with CHECK constraints in SQLite and typed constants in Go code.

| Enum Name | Go Type | Values | Used In |
|-----------|---------|--------|---------|
| `MediaType` | `string` | `movie`, `tv` | Media.Type, Watchlist.Type |
| `WatchStatus` | `string` | `to-watch`, `watched` | Watchlist.Status |
| `LogLevel` | `string` | `ERROR`, `WARN`, `INFO` | ErrorLog.Level |
| `FileActionType` | `int` | 1-8 (mapped to FileAction table rows) | MoveHistory.FileActionId, ActionHistory.FileActionId |

> Enum guidelines: Use `isEqual()` method for comparisons, never raw `==`. PascalCase constant names. Type suffix on the enum type name.

---

## 9. Index Summary

| Index | Table | Column(s) | Purpose |
|-------|-------|-----------|---------|
| `IdxMedia_TmdbId` | Media | TmdbId | TMDb API lookups |
| `IdxMedia_Type` | Media | Type | Filter by movie/tv |
| `IdxMedia_LanguageId` | Media | LanguageId | Language filter queries |
| `IdxMedia_ScanHistoryId` | Media | ScanHistoryId | Find media from a specific scan |
| `IdxCast_TmdbPersonId` | Cast | TmdbPersonId | TMDb person dedup |
| `IdxMediaGenre_MediaId` | MediaGenre | MediaId | Genre lookup by media |
| `IdxMediaGenre_GenreId` | MediaGenre | GenreId | Media lookup by genre |
| `IdxMediaCast_MediaId` | MediaCast | MediaId | Cast lookup by media |
| `IdxMediaCast_CastId` | MediaCast | CastId | Media lookup by cast member |
| `IdxTag_MediaId` | Tag | MediaId | Tags for a media item |
| `IdxScanHistory_ScanFolderId` | ScanHistory | ScanFolderId | History for a folder |
| `IdxMoveHistory_MediaId` | MoveHistory | MediaId | Move history for a media item |
| `IdxMoveHistory_FileActionId` | MoveHistory | FileActionId | Filter by action type |
| `IdxMoveHistory_IsUndone` | MoveHistory | IsUndone | Find undoable operations |
| `IdxActionHistory_FileActionId` | ActionHistory | FileActionId | Filter by action type |
| `IdxActionHistory_MediaId` | ActionHistory | MediaId | Action history for a media item |
| `IdxActionHistory_BatchId` | ActionHistory | BatchId | Group actions by batch |
| `IdxActionHistory_IsUndone` | ActionHistory | IsUndone | Find undoable actions |
| `IdxErrorLog_Level` | ErrorLog | Level | Filter by severity |
| `IdxErrorLog_Command` | ErrorLog | Command | Filter by command |
| `IdxErrorLog_Timestamp` | ErrorLog | Timestamp | Time-based queries |

---

## 10. Cross-References

| Reference | Location |
|-----------|----------|
| ER Diagram | [01-db-schema-diagram.mmd](./01-db-schema-diagram.mmd) |
| State History & Undo/Redo | [02-state-history-spec.md](./02-state-history-spec.md) |
| Popout Spec | [03-popout-spec.md](./03-popout-spec.md) |
| Migration Spec | [05-database-migration-spec.md](./05-database-migration-spec.md) |
| Error Handling Spec | [../04-error-handling-spec.md](../04-error-handling-spec.md) |
| Database Naming Conventions | [../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/01-naming-conventions.md](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/01-naming-conventions.md) |
| Schema Design Rules | [../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/02-schema-design.md](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/02-schema-design.md) |
| ORM and Views | [../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/03-orm-and-views.md](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/03-orm-and-views.md) |
| Split DB Pattern | [../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/07-split-db-pattern.md](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/07-split-db-pattern.md) |
| Data Folder Location | [../../../.lovable/memory/features/data-folder-location.md](../../../.lovable/memory/features/data-folder-location.md) |

---

*Database design specification v1.0.0 — updated: 2026-04-15*
