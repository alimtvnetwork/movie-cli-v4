# Movie Popout Command Specification

**Version:** 1.0.0  
**Updated:** 2026-04-15  
**Status:** Planned

---

## 1. Overview

`movie popout` extracts video files from nested subfolders and moves them to the parent (root) directory. This is common when downloaded movies come wrapped in folders with extras, subtitles, and sample files.

**Example:**

```
Before:
  ~/Downloads/
    Inception.2010.1080p.BluRay/
      Inception.2010.1080p.BluRay.mkv    ← video file
      Sample/
        sample.mkv
      Subs/
        english.srt
    The.Matrix.1999.x264/
      The.Matrix.1999.x264.mp4           ← video file
      nfo/
        movie.nfo

After (movie popout ~/Downloads):
  ~/Downloads/
    Inception (2010).mkv                  ← popped out + renamed
    The Matrix (1999).mp4                 ← popped out + renamed
    Inception.2010.1080p.BluRay/          ← empty folders listed for removal
    The.Matrix.1999.x264/
```

---

## 2. Command Interface

```
movie popout [directory]          # Pop out video files from subfolders
movie popout --dry-run            # Preview what would happen without moving
movie popout --no-rename          # Keep original filename, don't clean
movie popout --depth 2            # Max subfolder depth to search (default: 3)
```

### 2.1 Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Preview only, no file operations |
| `--no-rename` | bool | false | Keep original filename instead of cleaning |
| `--depth` | int | 3 | Maximum subfolder depth to search |

---

## 3. Command Flow

### 3.1 Discovery Phase

1. Accept target directory (argument or prompt from scan history)
2. Recursively find all video files in subfolders (not root-level files)
3. For each video file:
   - Parse filename with `cleaner.Clean()`
   - Determine clean filename
   - Calculate destination path (parent root directory)
4. Display preview table:

```
Movie Popout — 3 files found in subfolders

  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  1. Inception (2010)  [4.2 GB]
     From: ~/Downloads/Inception.2010.1080p.BluRay/Inception.2010.1080p.BluRay.mkv
     To:   ~/Downloads/Inception (2010).mkv

  2. The Matrix (1999)  [2.1 GB]
     From: ~/Downloads/The.Matrix.1999.x264/The.Matrix.1999.x264.mp4
     To:   ~/Downloads/The Matrix (1999).mp4

  3. Interstellar (2014)  [5.8 GB]
     From: ~/Downloads/Interstellar.2014.IMAX/video/Interstellar.2014.IMAX.mkv
     To:   ~/Downloads/Interstellar (2014).mkv

  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Pop out all 3 files? [y/N]:
```

### 3.2 Execution Phase

1. For each file (on confirmation):
   - Move file to root directory with clean name
   - Log in `move_history` (from_path, to_path)
   - Log in `action_history` with `action_type = 'popout'`
   - Update `media` table if entry exists
2. Display success summary

### 3.3 Folder Cleanup Phase

After all files are moved, identify now-empty or near-empty subfolders:

```
Empty folders after popout:

  1. Inception.2010.1080p.BluRay/
     └── (empty)

  2. The.Matrix.1999.x264/
     └── 1 file remaining: movie.nfo (2 KB)

  3. Interstellar.2014.IMAX/
     └── 2 files remaining:
         - sample.mkv (50 MB)
         - english.srt (120 KB)

  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Options:
    [a] Remove all empty folders
    [s] Select folders to remove one by one
    [n] Keep all folders
    [l] List files in each folder before deciding

  Choose [a/s/n/l]:
```

**Option `s` (select) flow:**
```
  Inception.2010.1080p.BluRay/ — empty
    Remove? [y/N]: y
    Removed.

  The.Matrix.1999.x264/ — 1 file (movie.nfo, 2 KB)
    Files:
      - movie.nfo (2 KB)
    Remove folder and contents? [y/N]: n
    Kept.

  Interstellar.2014.IMAX/ — 2 files (50.1 MB)
    Files:
      - sample.mkv (50 MB)
      - english.srt (120 KB)
    Remove folder and contents? [y/N]: y
    Removed.
```

---

## 4. State Tracking

All popout operations are fully tracked for undo support:

### 4.1 `move_history` Entry

Each file move creates a `move_history` record:

| Field | Value |
|-------|-------|
| `media_id` | FK to media (if exists, else insert new) |
| `from_path` | Original nested path |
| `to_path` | New root-level path |
| `original_file_name` | Original filename |
| `new_file_name` | Clean filename |
| `undone` | 0 |

### 4.2 `action_history` Entry

Each popout creates an `action_history` record:

| Field | Value |
|-------|-------|
| `action_type` | `popout` |
| `media_id` | FK to media |
| `detail` | `"Popped out: Inception (2010).mkv from Inception.2010.1080p.BluRay/"` |
| `batch_id` | Shared UUID for this popout session |
| `undone` | 0 |

### 4.3 Folder Deletion Tracking

Deleted folders are logged so undo can recreate them:

| Field | Value |
|-------|-------|
| `action_type` | `delete` |
| `media_snapshot` | JSON with folder path and file listing |
| `detail` | `"Removed folder: Inception.2010.1080p.BluRay/"` |
| `batch_id` | Same batch as the popout |

---

## 5. Undo Behavior

`movie undo` for a popout batch:

1. Move files back to their original subfolder paths
2. Recreate deleted folders if needed (`os.MkdirAll`)
3. Restore `media.current_file_path` to original
4. Mark `move_history` and `action_history` records as `undone = 1`

---

## 6. History View

`movie history --type popout` shows:

```
Popout History

  Batch: abc123 — 2026-04-15 14:30 MYT
    1. Inception (2010).mkv ← Inception.2010.1080p.BluRay/
    2. The Matrix (1999).mp4 ← The.Matrix.1999.x264/
    3 folders removed

  Batch: def456 — 2026-04-14 09:15 MYT
    1. Interstellar (2014).mkv ← Interstellar.2014.IMAX/video/
```

---

## 7. Error Handling

- **File conflict**: If destination already exists, skip with warning (don't overwrite)
- **Permission denied**: Log to `error_logs`, continue with next file
- **Partial failure**: Report `N/M moved, K failed` — all successful moves are still tracked
- **Folder not empty**: Only offer deletion for folders that had their video files popped out

---

## 8. Implementation Priority

| Phase | Items |
|-------|-------|
| Phase 1 | Basic popout: discover + move + rename |
| Phase 2 | Folder cleanup (list, select, remove) |
| Phase 3 | State tracking via `move_history` + `action_history` |
| Phase 4 | `--dry-run` and `--no-rename` flags |
| Phase 5 | Undo support integration |

---

*Popout spec — updated: 2026-04-15*
