# Project Plan & Status

> **Last Updated**: 15-Apr-2026

## ✅ Completed

### Core CLI Structure
- [x] Root command with Cobra (`movie-cli`)
- [x] `hello` command with version display
- [x] `version` command with ldflags injection
- [x] `self-update` command via git pull --ff-only

### Movie Management Commands
- [x] `movie config` — get/set configuration with masked API key display
- [x] `movie scan` — folder scanning with TMDb metadata + poster download
- [x] `movie ls` — paginated list with interactive navigation + detail view
- [x] `movie search` — live TMDb search, select, save to DB
- [x] `movie info` — local DB lookup → TMDb fallback → auto-persist
- [x] `movie suggest` — genre-based recommendations + trending fallback
- [x] `movie move` — interactive browse, move, track history
- [x] `movie rename` — batch clean rename with undo tracking
- [x] `movie undo` — revert last move/rename operation
- [x] `movie play` — open file with system default player (cross-platform)
- [x] `movie stats` — counts, genre chart, average ratings

### Infrastructure
- [x] SQLite database with migrations (5 tables, 7 indexes)
- [x] TMDb API client (search, details, credits, recommendations, trending, posters)
- [x] Filename cleaner (junk removal, year extraction, TV detection, slugs)
- [x] Makefile with build + cross-compile targets
- [x] build.ps1 PowerShell deploy script
- [x] spec.md — full project specification
- [x] Shared resolver helper (`movie_resolve.go`)

### Bug Fixes
- [x] Fixed timestamp bug — `saveHistoryLog` now uses `time.Now().Format(time.RFC3339)`
- [x] Deduplicated TMDb fetch logic — shared `fetchMovieDetails()`/`fetchTVDetails()`

### Refactoring
- [x] Split `cmd/movie_move.go` → `movie_move.go` + `movie_move_helpers.go`
- [x] Split `db/sqlite.go` → 5 focused files

### Documentation
- [x] README.md (basic), spec.md, ai-handoff.md, development-log.md
- [x] .lovable/memory structure with suggestions, issues, workflow
- [x] AI success rate plan
- [x] Reliability risk report (05-Apr-2026)

### Spec Restructuring (Phase 1-5)
- [x] Phase 1: Spec authoring guideline review
- [x] Phase 2: Spec folder audit
- [x] Phase 3: Naming/placement normalization (root lowercase, merge 02-app, flatten error spec)
- [x] Phase 4: Ignore rule verification (.gitignore audit)
- [x] Phase 5: Final consistency pass (N1-N4 renames, C1-C5 missing files created)

### PowerShell Automation (Phase 1-8) ✅
- [x] Phase 1: Core parameters & environment detection
- [x] Phase 2: Git operations (pull, conflict resolution, force-pull)
- [x] Phase 3: Go build pipeline integration
- [x] Phase 4: Deployment with backup & rollback
- [x] Phase 5: Logging & colored output helpers
- [x] Phase 6: Error handling audit (no swallowed errors)
- [x] Phase 7: install.ps1 bootstrap script
- [x] Phase 8: README.md automation docs update

### Release Pipeline & Install Scripts (09-Apr-2026) ✅
- [x] GitHub Actions release.yml — triggers on `release/**` branches and `v*` tags
- [x] Cross-compiled binaries for 6 targets (windows/linux/darwin × amd64/arm64)
- [x] Version-specific install.ps1 (Windows) with SHA256 verification
- [x] Version-specific install.sh (Linux/macOS) with SHA256 verification
- [x] Release page with changelog, checksums, install instructions, asset table
- [x] CHANGELOG.md created for release note extraction
- [x] Pipeline spec documentation (`spec/pipeline/`)

### CLI UX Improvements (09-Apr-2026) ✅
- [x] Root command shows version + comprehensive help with examples
- [x] `mahin --version` flag support
- [x] `mahin version` shows Go version and OS/arch
- [x] `mahin movie` shows categorized subcommand help with examples

### Database Redesign v2.0.0 (15-Apr-2026) ✅
- [x] Schema diagram — PascalCase, INTEGER AUTOINCREMENT, Split DB (4 databases)
- [x] Design spec — 16 tables, 8 views, all DDL + indexes documented
- [x] State & history spec — undo/redo via ActionHistory + batch operations
- [x] Popout spec — media file extraction with history tracking
- [x] Migration spec — fresh install, breaking upgrade, incremental; SchemaVersion table
- [x] Data folder structure — `<binary-dir>/data/` with config/ and log/ subfolders
- [x] FileAction expanded to 14 types (added TagAdd, TagRemove, WatchlistAdd, WatchlistRemove, WatchlistStatusChange, ConfigChange)
- [x] Collection table for TMDb movie collections ✅ 15-Apr-2026
- [x] Suggestions & proposals document

---

## 🔲 Pending — Prioritized Backlog

### Phase 1: Database Implementation (P0)
- [ ] Implement new Split DB schema in Go (`db/` package) — 4 databases, PascalCase tables
- [ ] Migrate `action_history.go` to use FileAction FK instead of inline action_type CHECK
- [ ] Implement SchemaVersion tracking + migration runner in Go
- [ ] Seed FileAction with 14 predefined rows
- [ ] Create 8 database views (VwMediaFull, VwMoveHistoryDetail, etc.)


### Phase 2: Code Alignment (P1)
- [ ] Update all commands to use new PascalCase column names
- [ ] Update `movie_info.go` / `movie_resolve.go` for new Media table structure
- [ ] Add Watchlist commands (add, remove, list, mark watched)
- [ ] Add Tag commands (add, remove, list by tag)

### Phase 3: Spec Completeness (P2)
- [ ] Acceptance criteria (GIVEN/WHEN/THEN) for all commands
- [ ] Shared helper docs — code comments marking shared helpers
- [ ] File size stats in `movie stats`

### Phase 4: Future Enhancements (P3)
- [ ] Director normalization table (separate from Media)
- [ ] Season/Episode tables for TV series
- [ ] REST API server mode with HTML dashboard
- [ ] Watchlist sync with TMDb account

---

## Next Task Selection

Pick one of these to implement next:

1. **Split DB implementation** — Create the 4 .db files with PascalCase schema in Go
2. **Migration runner** — SchemaVersion + sequential migration system
3. **Acceptance criteria** — Add GIVEN/WHEN/THEN to spec for all commands
