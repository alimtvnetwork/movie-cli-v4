# Suggestions Tracker

> **Last Updated**: 16-Apr-2026

## Status Legend
- ✅ Done — implemented and verified
- 🔲 Open — not started

---

## ✅ Completed

| # | Suggestion | Completed | Notes |
|---|-----------|-----------|-------|
| S01 | Fix timestamp bug in move-log.json | 17-Mar-2026 | Replaced `"now"` with `time.Now().Format(time.RFC3339)` |
| S02 | Refactor large files (>200 lines) | 17-Mar-2026 | Split `movie_move.go` and `db/sqlite.go` |
| S03 | Extract shared TMDb fetch logic | 17-Mar-2026 | `fetchMovieDetails()`/`fetchTVDetails()` in `movie_info.go` |
| S04 | Cross-drive move fallback (copy+delete) | 05-Apr-2026 | `MoveFile()` detects EXDEV, falls back to copy+remove |
| S05 | Add confirmation prompt to `movie undo` | 10-Apr-2026 | Already implemented with `[y/N]` prompt |
| S06 | Add GIVEN/WHEN/THEN acceptance criteria | 10-Apr-2026 | 16 ACs covering all commands + export + batch move |
| S07 | Document shared helper locations | 10-Apr-2026 | Annotated movie_info.go, movie_resolve.go, movie_move_helpers.go, movie_scan_json.go |
| S08 | Clarify `movie ls` filter rule | 09-Apr-2026 | Only file-backed (scanned) items shown |
| S09 | Implement `movie tag` command | 06-Apr-2026 | `cmd/movie_tag.go` + `db/tags.go` |
| S10 | Add file size stats to `movie stats` | 10-Apr-2026 | Total, largest, smallest, average |
| S11 | Add error handling spec | 10-Apr-2026 | TMDb rate limits, DB locks, offline mode, filesystem errors |
| S12 | Update README.md with full docs | 10-Apr-2026 | 620+ lines, all commands, install, build |
| S13 | Batch move (`--all` flag) | 09-Apr-2026 | Move all video files from source at once |
| S14 | JSON metadata per movie/TV on scan | 09-Apr-2026 | `cmd/movie_scan_json.go` |
| S15 | Use `DiscoverByGenre` in suggest | 09-Apr-2026 | Genre-based discovery integrated |
| S16 | CI pipeline (lint, test, vuln scan) | 10-Apr-2026 | ci.yml + vulncheck.yml |
| S17 | Retry logic with exponential backoff | 11-Apr-2026 | 429 rate-limit handling, 3 retries |
| S18 | Add `movie duplicates` command | 10-Apr-2026 | Detect by TMDb ID, filename, or size |
| S19 | Add `movie cleanup` command | 10-Apr-2026 | Find/remove stale DB entries |
| S20 | Integration tests with SQLite fixtures | 11-Apr-2026 | db/db_test.go + db/testhelper_test.go |
| S21 | Apply error log spec v2 to ci.yml | 10-Apr-2026 | Per-stage error logs, summary assembly |
| S22 | Add `movie watch` / watchlist | 11-Apr-2026 | to-watch/watched tracking |
| S23 | Console-safe updater handoff | 16-Apr-2026 | Synchronous execution, exit code propagation, gitmap pattern |
| S24 | Guideline violations audit | 16-Apr-2026 | 280+ violations catalogued, 7-phase remediation plan |
| S25 | Nested-if refactoring (top 20 files) | 16-Apr-2026 | Early returns, guard clauses, extracted helpers |

---

## 🔲 Open — Priority Order

| # | Suggestion | Priority | Description |
|---|-----------|----------|-------------|
| S26 | Magic strings → constants | High | Replace all hardcoded strings with constants/enums (guideline Phase 3) |
| S27 | fmt.Errorf → apperror.Wrap() | High | Replace all fmt.Errorf calls (guideline Phase 4) |
| S28 | Oversized functions split | Medium | Split functions >15 lines (guideline Phase 5) |
| S29 | Oversized files split | Medium | Split files >300 lines (guideline Phase 6) |

---

*Tracker updated: 16-Apr-2026*
