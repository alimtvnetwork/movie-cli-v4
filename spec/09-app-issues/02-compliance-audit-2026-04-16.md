# Compliance Audit — 2026-04-16

Full codebase audit against all coding guidelines.

---

## Summary

| Rule | Status | Count | Trend vs Last Audit |
|------|--------|-------|---------------------|
| `fmt.Errorf` / `errors.New` (use `apperror`) | ⚠️ | 6 | ✅ Down from 93 → 6 |
| Files > 300 lines | ⚠️ | 3 | ✅ Down from 6 → 3 |
| Functions > 15 lines | ❌ | 219 | ⬆️ Up from 70+ (more files added) |
| Nested `} else {` | ❌ | 43 | ✅ Down from 50+ → 43 |
| Functions with > 3 params | ❌ | 20 | ⬆️ Up from 10+ → 20 |
| Magic strings | ⚠️ | 13 | ✅ Down from 30+ → 13 |
| Boolean negative naming | ✅ | 0 | ✅ Clean (os.IsNotExist is stdlib, OK) |
| Unused imports | ✅ | 0 | ✅ Clean |
| No `else` after `return` | ❌ | 43 | Same as nested-else count |

---

## 1. `fmt.Errorf` / `errors.New` — 6 remaining

All in `tmdb/client.go` as package-level sentinel errors. These are idiomatic Go sentinel errors — **acceptable exception** since they're `var` constants compared with `errors.Is()`.

```
tmdb/client.go:21  ErrAuthInvalid
tmdb/client.go:22  ErrAuthMissing
tmdb/client.go:23  ErrRateLimited
tmdb/client.go:24  ErrServerError
tmdb/client.go:25  ErrNetworkError
tmdb/client.go:26  ErrTimeout
```

**Verdict**: ✅ No action needed — sentinel errors are the correct pattern.

---

## 2. Files > 300 lines — 3 files

| File | Lines | Over by |
|------|-------|---------|
| `tmdb/client.go` | 323 | 23 |
| `db/schema.go` | 314 | 14 |
| `cmd/movie_suggest.go` | 314 | 14 |

**Action**: Split each into two files to get under 300.

---

## 3. Functions > 15 lines — 219 violations

Top offenders (>50 lines):

| File | Function | Lines |
|------|----------|-------|
| `db/schema.go` | `createTables` | 246 |
| `db/views.go` | `createViews` | 101 |
| `cmd/movie_ls_table.go` | `runMovieLsTable` | 81 |
| `cmd/movie_scan.go` | `runMovieScan` | 80 |
| `cmd/movie_info_table.go` | `printMediaDetailTable` | 75 |
| `cmd/movie_rescan.go` | `runMovieRescan` | 70 |
| `cmd/movie_ls_detail.go` | `printMediaDetail` | 69 |
| `updater/script.go` | `buildUpdateScriptContent` | 69 |
| `cmd/movie_cd.go` | `runMovieCd` | 67 |
| `cmd/movie_scan_html.go` | `writeHTMLReport` | 64 |
| `cmd/movie_scan_watch.go` | `runWatchLoop` | 64 |

**Action**: High priority. These need extraction into helper functions.

---

## 4. `} else {` violations — 43

Most are else-after-return patterns that should be flattened with guard clauses.

Top files:
- `cmd/movie_rescan.go` — 3 violations
- `cmd/movie_scan.go` — 3 violations
- `cmd/movie_suggest.go` — 3 violations
- `cmd/movie_undo.go` — 3 violations
- `cmd/movie_redo.go` — 2 violations

**Action**: Convert to early-return guard clauses.

---

## 5. Functions with > 3 params — 20

Notable:
- `saveHistoryLog` (5 params)
- `startPostScanServices` (6 params)
- `startRestWithOptionalWatch` (5 params)
- `applyTMDbResult` (5 params)
- `downloadThumbnail` (5 params)
- `discoverByGenres` (6 params)
- `fillFromRecommendations` (6 params)
- `fillFromTrending` (5 params)
- `appendUniqueResults` (5 params)

**Action**: Introduce option structs for functions with 4+ params.

---

## 6. Magic strings — 13 remaining

Mostly `"json"` and `"history"` path segments used in `filepath.Join`. These should be constants in `db/constants.go`.

---

## Priority Fix Order

| Priority | Category | Count | Effort |
|----------|----------|-------|--------|
| P1 | Flatten else-after-return | 43 | Medium |
| P2 | Split 3 oversized files | 3 | Low |
| P3 | Option structs for >3 params | 20 | Medium |
| P4 | Extract helpers from >50-line funcs | ~15 | High |
| P5 | Replace magic path strings | 13 | Low |
| P6 | Continue splitting 16-50 line funcs | ~200 | Very High |

---

## What's Clean ✅

- No unused imports
- No negative boolean names
- No `fmt.Errorf` in app code (only sentinel errors)
- `apperror` package fully adopted
- Pre-build validation in `run.ps1`
- Version bumping discipline maintained

---

*Audit completed: 2026-04-16*
