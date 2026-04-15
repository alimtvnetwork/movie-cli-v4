# State History & Undo/Redo Specification

**Version:** 1.0.0  
**Updated:** 2026-04-15  
**Status:** Planned

---

## 1. Overview

Every state-changing operation in the CLI must be tracked in the database so that it can be undone (and re-done). This covers:

- **File moves** (`movie move`, `movie popout`)
- **File renames** (`movie rename`)
- **File deletions / removals** (`movie cleanup --remove`)
- **Scan operations** (`movie scan` — adding/removing entries)

The `move_history` table already tracks move and rename operations with an `undone` flag. This spec extends that pattern to cover **all** reversible actions.

---

## 2. Current State Tracking

### 2.1 What Is Already Tracked

| Action | Table | Tracked Fields | Undo Support |
|--------|-------|----------------|--------------|
| File move | `move_history` | from_path, to_path, original_file_name, new_file_name, moved_at | ✅ `undone` flag |
| File rename | `move_history` | Same as move (rename = move within same dir) | ✅ `undone` flag |
| Folder scan | `scan_history` | folder_path, total_files, movies_found, tv_found, scanned_at | ❌ Log only |
| Media insert | `media` | All metadata fields, scanned_at | ❌ No revert |
| Media delete | N/A | Not tracked | ❌ No revert |

### 2.2 Gaps to Fill

| Action | Gap | Solution |
|--------|-----|----------|
| Media deletion | No record of what was deleted | New: `action_history` table |
| Scan additions | No per-file record of what was added | New: `action_history` table |
| Scan removals | No record of what was removed | New: `action_history` table |
| Popout operations | Not yet implemented | Use `move_history` + `action_history` |

---

## 3. Proposed: `action_history` Table

A unified audit log for all reversible operations beyond file moves.

```sql
CREATE TABLE IF NOT EXISTS action_history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    action_type   TEXT NOT NULL CHECK(action_type IN (
        'scan_add', 'scan_remove', 'delete', 'popout', 'restore', 'rescan_update'
    )),
    media_id      INTEGER,
    media_snapshot TEXT,          -- JSON snapshot of media record before change
    detail        TEXT,           -- Human-readable description
    batch_id      TEXT,           -- Groups related actions (e.g., one scan = one batch)
    undone        INTEGER DEFAULT 0,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_action_history_type ON action_history(action_type);
CREATE INDEX IF NOT EXISTS idx_action_history_batch ON action_history(batch_id);
CREATE INDEX IF NOT EXISTS idx_action_history_undone ON action_history(undone);
```

### 3.1 Field Descriptions

| Field | Purpose |
|-------|---------|
| `action_type` | Category of operation performed |
| `media_id` | FK to the affected media record (NULL if deleted) |
| `media_snapshot` | Full JSON of the media row **before** the change — enables undo by restoring |
| `detail` | e.g. `"Deleted: Scream (2022).mkv from /Movies"` |
| `batch_id` | UUID grouping all actions from one command invocation |
| `undone` | 0 = active, 1 = undone |

### 3.2 Action Types

| Type | When Created | Undo Behavior |
|------|-------------|---------------|
| `scan_add` | New media inserted during scan | Delete the media record |
| `scan_remove` | Media removed during incremental scan | Re-insert from snapshot |
| `delete` | `movie cleanup --remove` | Re-insert from snapshot |
| `popout` | `movie popout` extracts file | Move file back (uses `move_history`) |
| `restore` | Undo of a delete/remove | Delete again |
| `rescan_update` | `movie rescan` updates metadata | Restore old metadata from snapshot |

---

## 4. Undo/Redo Commands

### 4.1 `movie undo`

```
movie undo              # Undo the last action
movie undo --list       # Show recent undoable actions
movie undo --batch      # Undo entire last batch (e.g., full scan)
movie undo --id <id>    # Undo specific action by ID
```

**Flow:**
1. Query latest un-undone record from `move_history` or `action_history`
2. Display what will be undone, ask confirmation
3. Reverse the operation:
   - Move/rename → move file back, update `media.current_file_path`
   - Delete → re-insert from `media_snapshot`
   - Scan add → delete the media entry
4. Set `undone = 1`

### 4.2 `movie redo`

```
movie redo              # Redo the last undone action
movie redo --list       # Show recent redoable actions
movie redo --id <id>    # Redo specific action by ID
```

**Flow:**
1. Query latest `undone = 1` record
2. Re-apply the original operation
3. Set `undone = 0`

### 4.3 `movie history`

```
movie history                  # Show last 20 actions (all types)
movie history --type move      # Filter by type
movie history --type scan      # Filter by type
movie history --limit 50       # Custom limit
movie history --batch <id>     # Show all actions in a batch
```

---

## 5. Integration Points

### 5.1 `movie scan` Integration

When running an incremental scan on a previously scanned folder:

1. Generate a `batch_id` (UUID) for this scan session
2. For each **new** file found:
   - Insert into `media`
   - Insert `action_history` with `action_type = 'scan_add'`
3. For each **removed** file (in DB but not on disk):
   - Snapshot the media record as JSON
   - Delete from `media`
   - Insert `action_history` with `action_type = 'scan_remove'`, `media_snapshot` = JSON
4. For each **existing** file with missing metadata:
   - Snapshot current state
   - Rescan via TMDb
   - Insert `action_history` with `action_type = 'rescan_update'`

### 5.2 `movie rename` Integration

Already uses `move_history`. No changes needed — `movie undo` reads from `move_history`.

### 5.3 `movie move` Integration

Already uses `move_history`. No changes needed.

### 5.4 `movie cleanup` Integration

When removing stale entries:

1. Snapshot each media record as JSON
2. Delete from `media`
3. Insert `action_history` with `action_type = 'delete'`

---

## 6. Error Handling

All state operations must follow the error management spec:

- Wrap DB errors with context: `fmt.Errorf("undo move %d: %w", id, err)`
- Log failures to `error_logs` table
- Never leave partial state: use transactions for batch operations
- Display user-friendly messages on failure

---

## 7. Implementation Priority

| Phase | Items | Depends On |
|-------|-------|------------|
| Phase 1 | `action_history` table + migration | — |
| Phase 2 | `movie undo` (move_history only) | Phase 1 |
| Phase 3 | `movie history` command | Phase 1 |
| Phase 4 | Integrate `action_history` into scan/cleanup | Phase 1 |
| Phase 5 | `movie redo` command | Phase 2 |
| Phase 6 | Batch undo support | Phase 2 + 4 |

---

*State history spec — updated: 2026-04-15*
