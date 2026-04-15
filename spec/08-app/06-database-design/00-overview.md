# App Database Design

**Version:** 2.0.0  
**Updated:** 2026-04-15  
**Status:** Active

---

## Overview

Complete database design documentation for the Movie CLI (`mahin`). All persistent state — media metadata, file operations, scan history, user tags, watchlists, and error logs — is stored across multiple SQLite databases following the **Split DB** pattern.

### Split DB Layout

| Database File | Bounded Context | Tables |
|---------------|----------------|--------|
| `media.db` | Media & Operations | Media, Genre, MediaGenre, Cast, MediaCast, Language, Tag, ScanFolder, ScanHistory, MoveHistory, ActionHistory, FileAction |
| `watchlist.db` | Watch Tracking | Watchlist |
| `config.db` | Configuration | Config |
| `error-log.db` | Error Logging | ErrorLog |

### Data Folder Structure

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
        ├── log.txt
        └── error.log
```

> The `data/` folder is resolved relative to the binary's physical location via `os.Executable()` + `filepath.EvalSymlinks()`. See [Data Folder Location](../../../.lovable/memory/features/data-folder-location.md).

---

## Document Inventory

| # | File | Description | Status |
|---|------|-------------|--------|
| 01 | [01-db-schema-diagram.mmd](./01-db-schema-diagram.mmd) | Full ER diagram — all tables with relationships (Split DB) | ✅ Active |
| 02 | [02-state-history-spec.md](./02-state-history-spec.md) | State tracking & undo/redo spec | ✅ Active |
| 03 | [03-popout-spec.md](./03-popout-spec.md) | `movie popout` command spec | ✅ Active |
| 04 | [04-database-design-spec.md](./04-database-design-spec.md) | Full database design spec — tables, views, enums, conventions | ✅ Active |
| 05 | [05-database-migration-spec.md](./05-database-migration-spec.md) | Migration logic — versioning, drop-and-recreate, view creation | ✅ Active |

---

## Tables Summary

| Table | Database | Purpose | Records |
|-------|----------|---------|---------|
| `Media` | media.db | Core media metadata (title, TMDb data, file paths) | One per scanned file |
| `Genre` | media.db | Genre lookup table | One per unique genre |
| `MediaGenre` | media.db | Media-to-Genre join (1-N) | Many per media |
| `Cast` | media.db | Cast member lookup table | One per unique person |
| `MediaCast` | media.db | Media-to-Cast join (N-M) | Many per media |
| `Language` | media.db | Language lookup table | One per unique language |
| `Tag` | media.db | User-assigned tags per media item | Many per media |
| `ScanFolder` | media.db | Registered scan folder paths (root entity) | One per folder |
| `ScanHistory` | media.db | Folder scan log (detailed counts, duration) | One per scan run |
| `MoveHistory` | media.db | File move/rename operations with undo flag | One per move operation |
| `ActionHistory` | media.db | Unified audit log for all reversible operations | One per action |
| `FileAction` | media.db | Action type lookup table | Predefined action types |
| `Watchlist` | watchlist.db | To-watch / watched tracking linked to TMDb | One per tracked title |
| `Config` | config.db | Key-value settings (directories, page size) | System defaults |
| `ErrorLog` | error-log.db | Structured error/warning log entries | One per error event |

---

## Database Views

All views are created during migration and use the `Vw` prefix per naming conventions.

| View | Database | Pre-Joined Tables |
|------|----------|-------------------|
| `VwMediaDetail` | media.db | Media + Language |
| `VwMediaGenreList` | media.db | Media + MediaGenre + Genre |
| `VwMediaCastList` | media.db | Media + MediaCast + Cast |
| `VwMediaFull` | media.db | Media + Language + aggregated genres + aggregated cast |
| `VwMoveHistoryDetail` | media.db | MoveHistory + Media + FileAction |
| `VwActionHistoryDetail` | media.db | ActionHistory + Media + FileAction |
| `VwScanHistoryDetail` | media.db | ScanHistory + ScanFolder |
| `VwMediaTag` | media.db | Media + Tag |

---

## Enum Types (Go Code)

| Enum | Values | Used In |
|------|--------|---------|
| `MediaType` | `Movie`, `Tv` | Media.Type, Watchlist.Type |
| `WatchStatus` | `ToWatch`, `Watched` | Watchlist.Status |
| `LogLevel` | `Error`, `Warn`, `Info` | ErrorLog.Level |
| `FileActionType` | `Move`, `Rename`, `Delete`, `Popout`, `Restore`, `ScanAdd`, `ScanRemove`, `RescanUpdate` | FileAction.Name |

---

## Cross-References

- [DB Schema Diagram (legacy — deprecated)](../../06-diagrams/15-db-schema.mmd) — Redirects to canonical diagram
- [Error Handling Spec](../04-error-handling-spec.md) — Error logging architecture
- [State History Spec](./02-state-history-spec.md) — Undo/redo design
- [Popout Spec](./03-popout-spec.md) — File extraction command
- [Database Conventions](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/00-overview.md) — Naming, schema design, Split DB
- [Split DB Pattern](../../01-coding-guidelines/03-coding-guidelines-spec/10-database-conventions/07-split-db-pattern.md) — Split DB architecture

---

*Database design docs — updated: 2026-04-15*
