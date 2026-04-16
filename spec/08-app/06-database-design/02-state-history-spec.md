# State History & Undo/Redo Specification

**Version:** 2.1.0  
**Updated:** 2026-04-16  
**Status:** Planned

---

## 1. Overview

Every state-changing operation in the CLI must be tracked in the database so that it can be undone (and re-done). This covers:

- **File moves** (`movie move`, `movie popout`)
- **File renames** (`movie rename`)
- **File deletions / removals** (`movie cleanup --remove`)
- **Scan operations** (`movie scan` — adding/removing entries)

The `MoveHistory` table tracks move and rename operations with an `IsReverted` flag. The `ActionHistory` table extends that pattern to cover **all** reversible actions via a `FileActionId` FK to the `FileAction` lookup table.

> **Naming convention:** Boolean columns use positive semantic names with `Is`/`Has` prefix. The flag `IsReverted` means "this action has been reversed" (1 = reverted, 0 = active). Never use negative words like "un", "not", "no" in boolean names.

---

## 2. Current State Tracking

### 2.1 What Is Already Tracked

| Action | Table | Tracked Fields | Undo Support |
|--------|-------|----------------|--------------|
| File move | `MoveHistory` | FromPath, ToPath, OriginalFileName, NewFileName, MovedAt | ✅ `IsReverted` flag |
| File rename | `MoveHistory` | Same as move (rename = move within same dir) | ✅ `IsReverted` flag |
| Folder scan | `ScanHistory` | ScanFolderId, TotalFiles, MoviesFound, TvFound, ScannedAt | ❌ Log only |
| Media insert | `Media` | All metadata fields, ScannedAt | ❌ No revert |
| Media delete | N/A | Not tracked | ❌ No revert |

### 2.2 Gaps to Fill

| Action | Gap | Solution |
|--------|-----|----------|
| Media deletion | No record of what was deleted | `ActionHistory` with FileAction = Delete |
| Scan additions | No per-file record of what was added | `ActionHistory` with FileAction = ScanAdd |
| Scan removals | No record of what was removed | `ActionHistory` with FileAction = ScanRemove |
| Popout operations | Not yet implemented | `MoveHistory` + `ActionHistory` with FileAction = Popout |

---

## 3. ActionHistory Table

A unified audit log for all reversible operations beyond file moves. Uses `FileActionId` FK to the `FileAction` lookup table (14 predefined action types).

```sql
CREATE TABLE ActionHistory (
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

CREATE INDEX IdxActionHistory_FileActionId ON ActionHistory(FileActionId);
CREATE INDEX IdxActionHistory_MediaId      ON ActionHistory(MediaId);
CREATE INDEX IdxActionHistory_BatchId      ON ActionHistory(BatchId);
CREATE INDEX IdxActionHistory_IsReverted   ON ActionHistory(IsReverted);
```

### 3.1 Field Descriptions

| Field | Purpose |
|-------|---------|
| `FileActionId` | FK to FileAction lookup — identifies the type of operation |
| `MediaId` | FK to the affected Media record (NULL if deleted) |
| `MediaSnapshot` | Full JSON of the Media row **before** the change — enables undo by restoring |
| `Detail` | e.g. `"Deleted: Scream (2022).mkv from /Movies"` |
| `BatchId` | UUID grouping all actions from one command invocation |
| `IsReverted` | 0 = active, 1 = reverted (action has been reversed) |

### 3.2 FileAction Types (relevant to state history)

| FileAction Name | When Created | Undo Behavior |
|-----------------|-------------|---------------|
| `ScanAdd` | New media inserted during scan | Delete the Media record |
| `ScanRemove` | Media removed during incremental scan | Re-insert from MediaSnapshot |
| `Delete` | `movie cleanup --remove` | Re-insert from MediaSnapshot |
| `Popout` | `movie popout` extracts file | Move file back (uses `MoveHistory`) |
| `Restore` | Undo of a delete/remove | Delete again |
| `RescanUpdate` | `movie rescan` updates metadata | Restore old metadata from MediaSnapshot |

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
1. Query latest record where `IsReverted = 0` from `MoveHistory` or `ActionHistory`
2. Display what will be undone, ask confirmation
3. Reverse the operation:
   - Move/rename → move file back, update `Media.CurrentFilePath`
   - Delete → re-insert from `MediaSnapshot`
   - ScanAdd → delete the Media entry
4. Set `IsReverted = 1`

### 4.2 `movie redo`

```
movie redo              # Redo the last reverted action
movie redo --list       # Show recent redoable actions
movie redo --id <id>    # Redo specific action by ID
```

**Flow:**
1. Query latest record where `IsReverted = 1`
2. Re-apply the original operation
3. Set `IsReverted = 0`

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

1. Generate a `BatchId` (UUID) for this scan session
2. For each **new** file found:
   - Insert into `Media`
   - Insert `ActionHistory` with `FileActionId = ScanAdd`
3. For each **removed** file (in DB but not on disk):
   - Snapshot the Media record as JSON
   - Delete from `Media`
   - Insert `ActionHistory` with `FileActionId = ScanRemove`, `MediaSnapshot` = JSON
4. For each **existing** file with missing metadata:
   - Snapshot current state
   - Rescan via TMDb
   - Insert `ActionHistory` with `FileActionId = RescanUpdate`

### 5.2 `movie rename` Integration

Already uses `MoveHistory`. No changes needed — `movie undo` reads from `MoveHistory`.

### 5.3 `movie move` Integration

Already uses `MoveHistory`. No changes needed.

### 5.4 `movie cleanup` Integration

When removing stale entries:

1. Snapshot each Media record as JSON
2. Delete from `Media`
3. Insert `ActionHistory` with `FileActionId = Delete`

---

## 6. Error Handling

All state operations must follow the error management spec:

- Wrap DB errors with context: `fmt.Errorf("undo move %d: %w", id, err)`
- Log failures to `ErrorLog` table
- Never leave partial state: use transactions for batch operations
- Display user-friendly messages on failure

---

## 7. Implementation Priority

| Phase | Items | Depends On |
|-------|-------|------------|
| Phase 1 | `ActionHistory` table + migration | — |
| Phase 2 | `movie undo` (`MoveHistory` only) | Phase 1 |
| Phase 3 | `movie history` command | Phase 1 |
| Phase 4 | Integrate `ActionHistory` into scan/cleanup | Phase 1 |
| Phase 5 | `movie redo` command | Phase 2 |
| Phase 6 | Batch undo support | Phase 2 + 4 |

---

*State history spec — updated: 2026-04-16*
