# Project Plan & Status

> **Last Updated**: 16-Apr-2026

## ✅ Completed

### Core CLI Structure
- [x] Root command with Cobra (`movie-cli`)
- [x] `hello` command with version display
- [x] `version` command with ldflags injection
- [x] `self-update` → migrated to `update` command (gitmap console-safe handoff)

### Movie Management Commands
- [x] `movie config` — get/set configuration with masked API key display
- [x] `movie scan` — folder scanning with TMDb metadata + poster download
- [x] `movie ls` — paginated list with interactive navigation + detail view
- [x] `movie search` — live TMDb search, select, save to DB
- [x] `movie info` — local DB lookup → TMDb fallback → auto-persist
- [x] `movie suggest` — genre-based recommendations + trending fallback
- [x] `movie move` — interactive browse, move, track history (cross-drive support)
- [x] `movie rename` — batch clean rename with undo tracking
- [x] `movie undo` — revert last move/rename operation (with confirmation prompt)
- [x] `movie play` — open file with system default player (cross-platform)
- [x] `movie stats` — counts, genre chart, average ratings, file sizes
- [x] `movie tag` — add/remove/list tags
- [x] `movie export` — export library data
- [x] `movie duplicates` — detect by TMDb ID, filename, or size
- [x] `movie cleanup` — find/remove stale DB entries
- [x] `movie watch` — watchlist: to-watch/watched tracking

### Infrastructure
- [x] SQLite database with migrations (single mahin.db)
- [x] TMDb API client (search, details, credits, recommendations, trending, posters, retry with backoff)
- [x] Filename cleaner (junk removal, year extraction, TV detection, slugs)
- [x] Makefile with build + cross-compile targets
- [x] build.ps1 / run.ps1 PowerShell deploy script
- [x] spec.md — full project specification
- [x] Shared resolver helper (`movie_resolve.go`)
- [x] apperror package for error wrapping
- [x] errlog package for structured error logging

### Bug Fixes & Refactoring
- [x] Fixed timestamp bug — `saveHistoryLog` now uses RFC3339
- [x] Deduplicated TMDb fetch logic — shared helpers
- [x] Split large files to <200 lines
- [x] Cross-drive move fallback (copy+delete for EXDEV)
- [x] Undo confirmation prompt
- [x] Console-safe updater handoff (sync, exit code propagation) ✅ 16-Apr-2026
- [x] Nested-if refactoring — top 20 files cleaned ✅ 16-Apr-2026
- [x] Guideline violations audit — 280+ violations catalogued ✅ 16-Apr-2026

### Documentation & CI
- [x] README.md, spec.md, ai-handoff.md, development-log.md
- [x] .lovable/memory structure
- [x] AI success rate plan
- [x] Reliability risk report
- [x] CI pipeline (lint, test, vuln scan)
- [x] Release pipeline with cross-compile + install scripts
- [x] Integration tests with SQLite fixtures

### Spec Restructuring (Phase 1-5) ✅
- [x] All 5 phases complete

### PowerShell Automation (Phase 1-8) ✅
- [x] All 8 phases complete

### Database Redesign v2.0.0 (15-Apr-2026) ✅
- [x] Schema diagram, design spec, state/history, popout, migration spec
- [x] Collection table, Tag M-N via MediaTag, 14 FileAction types
- [x] Removed Split DB — single mahin.db

---

## 🔄 In Progress

### Guideline Violations Remediation (Phases 3-7)
- [x] Phase 1: Full audit (280+ violations) ✅ 16-Apr-2026
- [x] Phase 2: Nested-if elimination (top 20 files) ✅ 16-Apr-2026
- [ ] Phase 3: Magic strings → constants/enums
- [ ] Phase 4: fmt.Errorf → apperror.Wrap()
- [ ] Phase 5: Oversized functions (>15 lines) split
- [ ] Phase 6: Oversized files (>300 lines) split
- [ ] Phase 7: Final consistency pass

---

## 🔲 Pending — Prioritized Backlog

### Phase 1: Database Implementation (P0)
- [ ] Implement new schema in Go (`db/` package) — single `mahin.db`, PascalCase tables
- [ ] Implement SchemaVersion tracking + migration runner in Go
- [ ] Seed FileAction with 14 predefined rows
- [ ] Create 8 database views (VwMediaFull, VwMoveHistoryDetail, etc.)

### Phase 2: Code Alignment (P1)
- [ ] Update all commands to use new PascalCase column names
- [ ] Update `movie_info.go` / `movie_resolve.go` for new Media table structure

### Phase 3: Spec Completeness (P2)
- [ ] Acceptance criteria (GIVEN/WHEN/THEN) for all commands
- [ ] Shared helper docs — code comments marking shared helpers

### Phase 4: Future Enhancements (P3)
- [ ] Director normalization table
- [ ] Season/Episode tables for TV series
- [ ] REST API server mode with HTML dashboard
- [ ] Watchlist sync with TMDb account

---

## 🚫 Known Issues

- **User's local repo stuck at v2.14.0** — needs `git reset --hard origin/main`. See `.lovable/pending-issues/01-local-repo-stale.md`

---

## Next Task Selection

Pick one of these to implement next:

1. **Guideline Phase 3** — Replace magic strings with constants
2. **Guideline Phase 4** — Replace fmt.Errorf with apperror.Wrap()
3. **Single DB implementation** — Create mahin.db with PascalCase schema
4. **Migration runner** — SchemaVersion + sequential migration system
