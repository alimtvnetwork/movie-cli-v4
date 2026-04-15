# App Database Design

**Version:** 1.0.0  
**Updated:** 2026-04-15  
**Status:** Active

---

## Overview

Complete database design documentation for the Movie CLI (`mahin`). All persistent state — media metadata, file operations, scan history, user tags, watchlists, and error logs — is stored in a single SQLite database (`movie.db`).

---

## Document Inventory

| # | File | Description | Status |
|---|------|-------------|--------|
| 01 | [01-db-schema-diagram.mmd](./01-db-schema-diagram.mmd) | Full ER diagram — all 7 tables with relationships | ✅ Active |
| 02 | [02-state-history-spec.md](./02-state-history-spec.md) | State tracking & undo/redo spec | ✅ Active |
| 03 | [03-popout-spec.md](./03-popout-spec.md) | `movie popout` command spec | ✅ Active |

---

## Tables Summary

| Table | Purpose | Records |
|-------|---------|---------|
| `media` | Core media metadata (title, TMDb data, file paths) | One per scanned file |
| `move_history` | All file move/rename operations with undo flag | One per move operation |
| `tags` | User-assigned tags per media item | Many per media |
| `scan_history` | Folder scan log (counts, timestamps) | One per scan run |
| `config` | Key-value settings (directories, page size) | System defaults |
| `watchlist` | To-watch / watched tracking linked to TMDb | One per tracked title |
| `error_logs` | Structured error/warning log entries | One per error event |

---

## Cross-References

- [DB Schema Diagram (legacy)](../../06-diagrams/15-db-schema.mmd) — Original 6-table diagram (pre-error_logs)
- [Error Handling Spec](../04-error-handling-spec.md) — Error logging architecture
- [State History Spec](./02-state-history-spec.md) — Undo/redo design
- [Popout Spec](./03-popout-spec.md) — File extraction command

---

*Database design docs — updated: 2026-04-15*
