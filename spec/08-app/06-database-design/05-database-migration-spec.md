# Database Migration Specification

**Version:** 1.0.0  
**Updated:** 2026-04-15  
**Status:** Active  
**Scope:** Migration strategy, version tracking, drop-and-recreate logic, view creation, and seed data

---

## 1. Overview

The Movie CLI (`mahin`) uses an **embedded migration system** — no external migration tool is required. Migrations run automatically on startup before any command executes. The system supports two migration modes:

| Mode | When | Behavior |
|------|------|----------|
| **Fresh install** | No `data/` folder or no `mahin.db` file | Create database, tables, indexes, views, and seed data from scratch |
| **Breaking upgrade** | Database version < minimum compatible version | Delete `mahin.db` and recreate from scratch (data loss accepted) |
| **Incremental upgrade** | Database version is compatible but behind current | Run pending migrations sequentially |

> **v2.0.0 rule:** The first release with this schema (v2.0.0) is a **breaking migration**. Any database created by a prior version is deleted and recreated.

---

## 2. Version Tracking

### 2.1 Schema Version Table

The database file contains a `SchemaVersion` table to track its migration state:

```sql
CREATE TABLE SchemaVersion (
    SchemaVersionId   INTEGER PRIMARY KEY AUTOINCREMENT,
    Version           TEXT NOT NULL,
    MinCompatible     TEXT NOT NULL,
    AppliedAt         TEXT NOT NULL DEFAULT (datetime('now')),
    Description       TEXT
);
```

| Column | Purpose |
|--------|---------|
| Version | Current schema version (semver: `2.0.0`) |
| MinCompatible | Minimum app version this schema works with |
| AppliedAt | When this migration was applied |
| Description | Human-readable migration note |

### 2.2 Version Check Flow

```
App starts
  │
  ├── data/ folder exists?
  │   ├── NO → Fresh install (Section 4)
  │   └── YES
  │       ├── SchemaVersion table exists?
  │       │   ├── NO → Legacy database detected → Drop and recreate (Section 5)
  │       │   └── YES
  │       │       ├── Version >= current? → No migration needed
  │       │       ├── Version >= MinCompatible? → Incremental migration (Section 6)
   │       │       └── Version < MinCompatible? → Drop and recreate (Section 5)
   │       └── Done — app proceeds to execute command
  └── Done — app proceeds to execute command
```

### 2.3 Version Constants (Go Code)

```go
const (
    // Current schema version — bump on every migration
    DbSchemaVersion = "2.0.0"

    // Minimum version that can be incrementally upgraded
    // Set to current version for breaking changes (forces drop-and-recreate)
    DbMinCompatible = "2.0.0"
)
```

---

## 3. Data Folder Initialization

On first run, the migration system creates the full folder structure:

```
<cli-binary-location>/
└── data/
    ├── mahin.db          ← created by migration
    ├── config/           ← created by migration
    └── log/              ← created by migration
        ├── log.txt       ← created on first log write
        └── error.log     ← created on first error log write
```

### 3.1 Folder Creation (Go Code)

```go
func ensureDataFolders(dataDir string) error {
    folders := []string{
        dataDir,
        filepath.Join(dataDir, "config"),
        filepath.Join(dataDir, "log"),
    }

    for _, folder := range folders {
        if err := os.MkdirAll(folder, 0755); err != nil {
            return fmt.Errorf("create folder %s: %w", folder, err)
        }
    }

    return nil
}
```

---

## 4. Fresh Install — Full Schema Creation

When no database file exists, the system creates the database with all tables, indexes, views, and seed data in order.

### 4.1 Execution Order

```
1. Create database file
2. Enable WAL mode: PRAGMA journal_mode=WAL
3. Enable foreign keys: PRAGMA foreign_keys=ON
4. Create SchemaVersion table
5. Create lookup tables (no FK dependencies)
6. Seed lookup tables with predefined data
7. Create entity tables (with FK to lookups)
8. Create join tables (with FK to entities)
9. Create indexes
10. Create views
11. Insert SchemaVersion record
```

### 4.2 Table Creation Order

```
Step  Table/Object                 Dependencies
────  ─────────────────────────    ──────────────
  1   SchemaVersion                none
  2   Language                     none (lookup)
  3   Genre                        none (lookup)
  4   Cast                         none (lookup)
  5   FileAction                   none (lookup)
  6   → Seed FileAction            FileAction exists
  7   ScanFolder                   none
  8   ScanHistory                  ScanFolder
  9   Collection                   none
 10   Media                        Language, ScanHistory, Collection
 10   MediaGenre                   Media, Genre
 11   MediaCast                    Media, Cast
 12   Tag                          (none — lookup)
 13   MediaTag                     Media, Tag
 14   MoveHistory                  Media, FileAction
 15   ActionHistory                Media, FileAction
 16   Watchlist                    Media (optional FK)
 17   Config                       none
 18   ErrorLog                     none
 19   → Create all indexes         all tables exist
 20   → Create all views           all tables exist
 21   → Insert SchemaVersion       done
```

### 4.3 Seed Data

#### FileAction (Predefined — 14 rows)

```sql
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

> **Important:** FileAction IDs are stable and mapped to Go enum constants. Never reorder or delete rows. New actions are appended only.

---

## 5. Breaking Upgrade — Drop and Recreate

When the existing database version is below `DbMinCompatible`, the system performs a clean reset.

### 5.1 Drop and Recreate Flow

```
1. Log warning: "Database version X.X.X is incompatible with minimum Y.Y.Y — recreating"
2. Close all database connections
3. Delete all .db files in data/ folder
4. Delete all .db-wal and .db-shm files (WAL cleanup)
5. Run fresh install (Section 4)
6. Log info: "Database recreated at version Z.Z.Z"
```

### 5.2 Safety Rules

| Rule | Description |
|------|-------------|
| Never delete `data/config/` | User configuration files are preserved across resets |
| Never delete `data/log/` | Log history is preserved across resets |
| Only delete `.db` files | The `data/` folder itself and subfolders are kept |
| Log before delete | Always write to `error.log` before deleting databases |
| No user prompt | Drop-and-recreate is automatic — the CLI is the sole user |

### 5.3 Detection of Legacy Databases (Pre-v2.0.0)

Legacy databases from before the v2.0.0 schema redesign can be detected by:

1. **No `SchemaVersion` table** — legacy databases didn't have version tracking
2. **Table named `media` (lowercase)** — legacy used lowercase table names
3. **Column named `id` (not `MediaId`)** — legacy used generic `id` columns
4. **Single `movie.db` file** — legacy used one monolithic database

Any of these conditions trigger drop-and-recreate.

```go
func isLegacyDatabase(dataDir string) bool {
    legacyPath := filepath.Join(dataDir, "movie.db")
    if _, err := os.Stat(legacyPath); err == nil {
        return true
    }

    // Check for SchemaVersion table in any existing .db file
    for _, dbFile := range splitDbFiles {
        if !hasSchemaVersionTable(filepath.Join(dataDir, dbFile)) {
            return true
        }
    }

    return false
}
```

---

## 6. Incremental Migration

When the database version is compatible but behind the current version, pending migrations are applied sequentially.

### 6.1 Migration Registry

Each migration is a named function registered in order:

```go
type Migration struct {
    Version     string
    Database    string   // which .db file this targets
    Description string
    Up          func(db *sql.DB) error
}

var migrations = []Migration{
    {
        Version:     "2.0.0",
        Database:    "media.db",
        Description: "Initial schema — full redesign with Split DB",
        Up:          migrateMedia200,
    },
    // Future migrations appended here:
    // {
    //     Version:     "2.1.0",
    //     Database:    "media.db",
    //     Description: "Add ReleaseDate column to Media",
    //     Up:          migrateMedia210,
    // },
}
```

### 6.2 Migration Execution Flow

```
1. Read current version from SchemaVersion (latest row)
2. Filter migrations where Version > current version
3. Sort by version (ascending)
4. For each pending migration:
   a. Begin transaction
   b. Execute migration function
   c. Insert SchemaVersion record
   d. Commit transaction
   e. Log: "Applied migration X.X.X: description"
5. If any migration fails:
   a. Rollback transaction
   b. Log error to error.log
   c. Exit with error message
```

### 6.3 Migration Rules

| Rule | Description |
|------|-------------|
| Forward-only | No down migrations — rollback by creating a new forward migration |
| One database per migration | Each migration targets exactly one `.db` file |
| Transaction-wrapped | Every migration runs inside a transaction |
| Idempotent where possible | Use `IF NOT EXISTS` for CREATE, `IF EXISTS` for DROP |
| Version in SchemaVersion | Every migration inserts a row in SchemaVersion |
| Never modify seed data IDs | FileAction IDs are stable — only append new rows |
| Views recreated on change | If underlying tables change, drop and recreate dependent views |

---

## 7. View Creation and Maintenance

### 7.1 View Creation Timing

Views are created:
- During **fresh install** — after all tables and indexes
- During **drop-and-recreate** — same as fresh install
- During **incremental migration** — when underlying tables change

### 7.2 View Recreation Pattern

When a migration modifies a table that a view depends on, the view must be dropped and recreated:

```go
func recreateView(db *sql.DB, viewName string, viewSQL string) error {
    _, err := db.Exec("DROP VIEW IF EXISTS " + viewName)
    if err != nil {
        return fmt.Errorf("drop view %s: %w", viewName, err)
    }

    _, err = db.Exec(viewSQL)
    if err != nil {
        return fmt.Errorf("create view %s: %w", viewName, err)
    }

    return nil
}
```

### 7.3 View Dependency Map

| View | Depends On |
|------|------------|
| VwMediaDetail | Media, Language |
| VwMediaGenreList | MediaGenre, Genre |
| VwMediaCastList | MediaCast, Cast |
| VwMediaFull | Media, Language, MediaGenre, Genre, MediaCast, Cast |
| VwMoveHistoryDetail | MoveHistory, Media, FileAction |
| VwActionHistoryDetail | ActionHistory, Media, FileAction |
| VwScanHistoryDetail | ScanHistory, ScanFolder |
| VwMediaTag | MediaTag, Tag, Media |

> If `Media` table changes, recreate: VwMediaDetail, VwMediaFull, VwMoveHistoryDetail, VwActionHistoryDetail, VwMediaTag.

---

## 8. SQLite Pragmas

Applied to every database connection immediately after opening:

```sql
PRAGMA journal_mode = WAL;         -- Write-Ahead Logging for concurrent reads
PRAGMA foreign_keys = ON;          -- Enforce FK constraints
PRAGMA busy_timeout = 5000;        -- Wait up to 5s on lock contention
PRAGMA synchronous = NORMAL;       -- Balance between safety and speed
PRAGMA cache_size = -8000;         -- 8MB page cache
```

```go
func applyPragmas(db *sql.DB) error {
    pragmas := []string{
        "PRAGMA journal_mode = WAL",
        "PRAGMA foreign_keys = ON",
        "PRAGMA busy_timeout = 5000",
        "PRAGMA synchronous = NORMAL",
        "PRAGMA cache_size = -8000",
    }

    for _, pragma := range pragmas {
        if _, err := db.Exec(pragma); err != nil {
            return fmt.Errorf("apply %s: %w", pragma, err)
        }
    }

    return nil
}
```

---

## 9. Startup Migration Sequence

Complete flow executed before any CLI command:

```
App starts
  │
  ├─ 1. Resolve data directory (os.Executable + EvalSymlinks)
  ├─ 2. Ensure data/, data/config/, data/log/ folders exist
  ├─ 3. Check for legacy database (movie.db or missing SchemaVersion)
  │     ├── Legacy found → delete all .db files, log warning
  │     └── No legacy → continue
  ├─ 4. For each Split DB file (media.db, watchlist.db, config.db, error-log.db):
  │     ├── File missing? → Fresh install for this DB
  │     ├── Version < MinCompatible? → Drop and recreate this DB
  │     ├── Version < Current? → Run incremental migrations
  │     └── Version == Current? → No action
  ├─ 5. Apply pragmas to all connections
  ├─ 6. Register connections in DB registry
  └─ 7. Proceed to command execution
```

### 9.1 Go Implementation Sketch

```go
func InitDatabases(dataDir string) (*DbRegistry, error) {
    // Step 1-2: Folders
    if err := ensureDataFolders(dataDir); err != nil {
        return nil, err
    }

    // Step 3: Legacy check
    if isLegacyDatabase(dataDir) {
        logWarn("Legacy database detected — recreating all databases")
        if err := deleteAllDatabases(dataDir); err != nil {
            return nil, err
        }
    }

    // Step 4: Per-database migration
    registry := NewDbRegistry()
    dbConfigs := []struct {
        Name       string
        File       string
        Migrations []Migration
    }{
        {"media", "media.db", mediaMigrations},
        {"watchlist", "watchlist.db", watchlistMigrations},
        {"config", "config.db", configMigrations},
        {"error-log", "error-log.db", errorLogMigrations},
    }

    for _, cfg := range dbConfigs {
        dbPath := filepath.Join(dataDir, cfg.File)
        db, err := sql.Open("sqlite3", dbPath)
        if err != nil {
            return nil, fmt.Errorf("open %s: %w", cfg.File, err)
        }

        // Step 5: Pragmas
        if err := applyPragmas(db); err != nil {
            return nil, err
        }

        // Migrate
        if err := runMigrations(db, cfg.Migrations); err != nil {
            return nil, fmt.Errorf("migrate %s: %w", cfg.File, err)
        }

        // Step 6: Register
        registry.Register(cfg.Name, db)
    }

    return registry, nil
}
```

---

## 10. Error Handling During Migration

| Scenario | Behavior |
|----------|----------|
| Cannot create `data/` folder | Exit with error: `"failed to create data directory: <path>"` |
| Cannot create `.db` file | Exit with error: `"failed to create database: <file>"` |
| Migration SQL fails | Rollback transaction, log full error + stack trace to `error.log`, exit |
| Cannot delete legacy `.db` | Log error, attempt to rename to `.db.bak`, exit if rename fails |
| Pragma fails | Log warning, continue (non-fatal) |
| SchemaVersion insert fails | Rollback migration, exit with error |

> All migration errors are logged to both stderr and `data/log/error.log`. See [Error Handling Spec](../04-error-handling-spec.md).

---

## 11. Future Migration Examples

### 11.1 Adding a Column

```go
{
    Version:     "2.1.0",
    Database:    "media.db",
    Description: "Add ReleaseDate column to Media",
    Up: func(db *sql.DB) error {
        _, err := db.Exec("ALTER TABLE Media ADD COLUMN ReleaseDate TEXT")
        if err != nil {
            return err
        }
        // Recreate views that depend on Media
        return recreateMediaViews(db)
    },
}
```

### 11.2 Adding a New Lookup Value

```go
{
    Version:     "2.2.0",
    Database:    "media.db",
    Description: "Add Merge action type to FileAction",
    Up: func(db *sql.DB) error {
        _, err := db.Exec("INSERT INTO FileAction (Name) VALUES ('Merge')")
        return err
    },
}
```

### 11.3 Adding a New Table

```go
{
    Version:     "2.3.0",
    Database:    "media.db",
    Description: "Add Collection table for movie collections",
    Up: func(db *sql.DB) error {
        _, err := db.Exec(`
            CREATE TABLE Collection (
                CollectionId INTEGER PRIMARY KEY AUTOINCREMENT,
                Name         TEXT NOT NULL,
                TmdbCollectionId INTEGER UNIQUE,
                CreatedAt    TEXT NOT NULL DEFAULT (datetime('now'))
            )
        `)
        return err
    },
}
```

---

## 12. Checklist — New Migration

When adding a new migration:

- [ ] Migration version follows semver and is greater than all existing
- [ ] Migration targets exactly one database file
- [ ] Migration function is wrapped in a transaction
- [ ] `IF NOT EXISTS` / `IF EXISTS` used where applicable
- [ ] SchemaVersion row is inserted automatically by the runner
- [ ] Dependent views are dropped and recreated if underlying tables change
- [ ] FileAction seed IDs are never modified — only append
- [ ] Migration is tested with in-memory DB
- [ ] Error messages include context (table name, column, migration version)
- [ ] Entry added to this document's future migration section (if notable)

---

## 13. Cross-References

| Reference | Location |
|-----------|----------|
| Database Design Spec | [04-database-design-spec.md](./04-database-design-spec.md) |
| ER Diagram | [01-db-schema-diagram.mmd](./01-db-schema-diagram.mmd) |
| State History & Undo/Redo | [02-state-history-spec.md](./02-state-history-spec.md) |
| Error Handling Spec | [../04-error-handling-spec.md](../04-error-handling-spec.md) |
| Split DB Pattern | [../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/07-split-db-pattern.md](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/07-split-db-pattern.md) |
| Data Folder Location | [../../../.lovable/memory/features/data-folder-location.md](../../../.lovable/memory/features/data-folder-location.md) |

---

*Database migration specification v1.0.0 — updated: 2026-04-15*
