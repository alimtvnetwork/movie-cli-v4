# Coding Guideline Violations — Full Audit Report

**Version:** 1.0.0  
**Updated:** 2026-04-16  
**Audited against:** spec/01-coding-guidelines/01-consolidated-review-guide.md

---

## Summary

| Category | Count | Severity |
|----------|-------|----------|
| Function exceeds 15 lines | 70+ | ⚠️ Dangerous |
| File exceeds 300 lines | 6 files | ⚠️ Dangerous |
| Nested `if` (zero-nesting rule) | 50+ instances | 🔴 Code Red |
| `else` after `return` | 30+ instances | ⚠️ Dangerous |
| `fmt.Errorf` / `errors.New` (no stack trace) | 93 instances | 🔴 Code Red |
| Magic strings (`"movie"`, `"tv"`, `"json"`, etc.) | 30+ instances | ⚠️ Dangerous |
| Parameters > 3 per function | 10+ functions | ⚠️ Dangerous |
| Mixed `&&`/`||` or `positive && !negative` | ~10 instances | ⚠️ Dangerous |

---

## Detailed Violations Table

### 1. 🔴 Nested `if` — Zero Nesting Rule Violations

| File | Line(s) | Description | Fix |
|------|---------|-------------|-----|
| `cmd/movie_scan_process.go` | L54-72 | Loop body → `if existing[i].OriginalFilePath` → `if useTable` → `if result.Type` (3 levels) | Extract `handleExistingMedia()` helper, early-return |
| `cmd/movie_scan_process.go` | L76-86 | `if fiErr` → `if os.IsNotExist` → `else if os.IsPermission` | Use `switch/case` or flat error classification helper |
| `cmd/movie_scan_process.go` | L110-129 | `if insertErr` → `if m.TmdbID > 0` → `if updateErr` → `else if m.Genre != ""` → `if existing != nil` (5 levels!) | Extract `handleInsertOrUpdate()` helper |
| `cmd/movie_scan_process.go` | L228-258 | `if best.PosterPath` → `if mkdirErr` → `if os.IsPermission` / `if dlErr` → `if errors.Is` → `if mkErr == nil` → `if src, rErr` → `if wErr` (7 levels!) | Extract `downloadAndSaveThumbnail()` helper |
| `cmd/movie_undo.go` | L399-445 | `switch` cases with nested `if a.MediaId.Valid` → `if err` → nested logic | Extract per-action-type undo helpers |
| `cmd/movie_redo.go` | L395-445 | Same pattern as undo — nested switch cases | Extract per-action-type redo helpers |
| `cmd/movie_info.go` | L49-197 | Entire function is 149 lines with 6+ nested branches | Split into `infoFromDB()`, `infoFromTMDb()`, `displayInfo()` |
| `cmd/movie_move.go` | L97-207 | `runBatchMove` 111 lines, nested if/else | Split into helper functions |
| `cmd/movie_popout.go` | L74-167 | `runMoviePopout` 94 lines, deeply nested | Flatten with guard clauses |
| `cmd/movie_ls.go` | L111-225 | `runMovieLsInteractive` 115 lines, nested | Split into smaller helpers |

### 2. 🔴 `fmt.Errorf` / `errors.New` — Missing Stack Traces (93 total)

| File | Count | Example |
|------|-------|---------|
| `cmd/movie_undo.go` | 8 | `fmt.Errorf("file not found at %s — may have been moved manually", m.ToPath)` |
| `cmd/movie_redo.go` | 11 | `fmt.Errorf("parse snapshot for redo action %d: %w", ...)` |
| `cmd/movie_move_helpers.go` | 6 | `fmt.Errorf("open source: %w", err)` |
| `cmd/movie_scan_helpers.go` | 5 | `fmt.Errorf("cannot determine current directory: %v", err)` |
| `cmd/movie_scan_html.go` | 4 | `fmt.Errorf("read template: %w", err)` |
| `cmd/movie_scan_json.go` | 3 | `fmt.Errorf("cannot create json dir: %w", err)` |
| `cmd/movie_resolve.go` | 3 | `fmt.Errorf("empty media identifier")` |
| `db/action_history.go` | 12 | `fmt.Errorf("mark action reverted %d: %w", id, err)` |
| `db/media.go` | ~15 | Various DB operation errors |
| Others | ~26 | Spread across remaining files |

**Guideline:** Must use `apperror.Wrap()` / `apperror.New()` for stack traces.

### 3. ⚠️ `else` After `return` (30+ instances)

| File | Line | Pattern |
|------|------|---------|
| `cmd/movie_info.go` | L62, 121, 162, 191 | `if format == "json" { ... } else { ... }` after return |
| `cmd/movie_move.go` | L59, 202, 324 | `else` after early return |
| `cmd/movie_popout.go` | L93, 154, 203, 291, 357, 412, 423, 435 | 8 instances! |
| `cmd/movie_redo.go` | L289, 347 | `else` in branch after return |
| `cmd/movie_rescan.go` | L48, 60, 109, 117, 148 | Multiple else-after-return |
| `cmd/movie_rescan_helper.go` | L65 | else after return |
| `cmd/movie_rest.go` | L74 | else after return |
| `cmd/movie_logs.go` | L77 | else after return |
| `cmd/movie_config.go` | L61 | else after return |
| `cmd/movie_history.go` | L88 | else after return |

### 4. ⚠️ Magic Strings (No enum/constant)

| Magic String | Files Using It | Fix |
|-------------|---------------|-----|
| `"movie"` | 15+ files | `MediaTypeMovie` constant |
| `"tv"` | 15+ files | `MediaTypeTv` constant |
| `"json"` | `movie_info.go`, `movie_logs.go`, `movie_history.go` | `OutputFormatJson` constant |
| `"table"` | `movie_info.go`, `movie_ls.go` | `OutputFormatTable` constant |
| `"default"` | `movie_history.go`, `movie_info.go` | `OutputFormatDefault` constant |
| `"to-watch"` / `"watched"` | `db/watchlist.go`, `cmd/movie_watch.go` | `WatchStatusToWatch`, `WatchStatusWatched` |
| `"active"` | `db/schema.go` checks | `ScanFolderStatusActive` constant |

### 5. ⚠️ Functions Exceeding 15 Lines (Top Offenders)

| File | Function | Lines | Action |
|------|----------|-------|--------|
| `cmd/movie_info.go` | `runMovieInfo` | 149 | Must split into 4-5 helpers |
| `cmd/movie_ls.go` | `runMovieLsInteractive` | 115 | Must split into helpers |
| `cmd/movie_move.go` | `runBatchMove` | 111 | Must split into helpers |
| `cmd/movie_history.go` | `collectUnifiedRecords` | 97 | Extract move/action collectors |
| `cmd/movie_popout.go` | `runMoviePopout` | 94 | Flatten with guards + helpers |
| `cmd/movie_rename.go` | `runMovieRename` | 88 | Extract validation + execution |
| `cmd/movie_move.go` | `runInteractiveMove` | 86 | Extract prompts + execution |
| `cmd/movie_logs.go` | `runMovieLogs` | 83 | Extract formatters |
| `cmd/movie_ls_table.go` | `runMovieLsTable` | 83 | Extract table building |
| `cmd/movie_popout.go` | `offerFolderCleanup` | 79 | Extract decisions to helpers |

### 6. ⚠️ Files Exceeding 300 Lines

| File | Lines | Action |
|------|-------|--------|
| `cmd/movie_undo.go` | 479 | Split: `movie_undo.go` + `movie_undo_exec.go` |
| `cmd/movie_redo.go` | 471 | Split: `movie_redo.go` + `movie_redo_exec.go` |
| `cmd/movie_popout.go` | 459 | Split: `movie_popout.go` + `movie_popout_helpers.go` |
| `db/media.go` | 414 | Split: `media.go` + `media_query.go` |
| `cmd/movie_rest.go` | 366 | Split: `movie_rest.go` + `movie_rest_routes.go` |
| `cmd/movie_history.go` | 339 | Split: keep, but extract collectors |
| `cmd/movie_move.go` | 338 | Split: keep, but extract batch/interactive |
| `cmd/movie_scan.go` | 337 | Split: keep, but extract rescan logic |

### 7. ⚠️ Parameters > 3

| File | Function | Params | Fix |
|------|----------|--------|-----|
| `cmd/movie_scan_process.go` | `processVideoFile` | 10 params | Use `ScanContext` struct |
| `db/history.go` | `InsertMoveHistory` | 6 params | Use `MoveRecord` struct |
| `db/history.go` | `InsertScanHistory` | 9 params | Use `ScanRecord` struct |
| `cmd/movie_scan.go` | Various internal | 4-6 params | Use options structs |
| `cmd/movie_move_helpers.go` | `saveHistoryLog` | 5 params | Use struct |

---

## Phased Fix Plan

### Phase 1: Constants & Enums ✅ DONE (2026-04-16)
**Scope:** Created `db/constants.go` with `MediaType`, `OutputFormat`, `WatchStatus` typed constants + helpers (`TypeIcon`, `TypeLabel`, `TypeLabelPlural`, `JSONSubDir`). Replaced 100+ magic strings across 20 files.
**Estimated effort:** Medium

### Phase 2: Flatten Nested `if` ✅ DONE (2026-04-16)
**Result:** Refactored `movie_scan_process.go` — extracted 8 helpers (`isAlreadyScanned`, `logStatError`, `handleInsertError`, `trackScanAction`, `downloadThumbnail`, `logMkdirError`, `logPosterDownloadError`, `copyThumbnailToDataDir`, `incrementTypeCount`). Max nesting reduced from 7 to 1.
**Target files:** `movie_scan_process.go` → `movie_scan_process_helpers.go`
**Estimated effort:** Large

### Phase 3: Split Oversized Functions ✅ DONE (2026-04-16)
**Result:** Split `runMovieInfo` (149L→26L + 6 helpers) and `suggestByType` (119L→15L + 7 helpers). Remaining large functions are schema DDL or complex scan logic needing separate effort.
**Priority order:** `runMovieInfo` (149L) → `suggestByType` (119L)
**Estimated effort:** Large

### Phase 4: Split Oversized Files
**Scope:** Split 6 files exceeding 300 lines.
**Order:** `movie_undo.go` → `movie_redo.go` → `movie_popout.go` → `db/media.go` → `movie_rest.go`
**Estimated effort:** Medium

### Phase 5: Remove `else` After `return` ✅ DONE (2026-04-16)
**Result:** Audit overcounted — no actual else-after-return violations exist. All `} else {` blocks are valid else-if chains for format switching.
**Estimated effort:** None needed

### Phase 6: Parameter Reduction ✅ DONE (2026-04-16)
**Result:** Created `ScanContext` struct, refactored `processVideoFile` from 10 params to 2 (vf + ctx). Updated all callers.
**Estimated effort:** Medium

### Phase 7: Error Handling — `apperror` Migration
**Scope:** Replace 93 `fmt.Errorf`/`errors.New` calls with `apperror.Wrap()`/`apperror.New()`.
**Prerequisite:** Must have `apperror` package in project.
**Estimated effort:** Large (deferred — requires `apperror` package first)

---

*Audit report v1.0.0 — generated 2026-04-16*
