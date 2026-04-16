# Project Overview

> **Last Updated**: 16-Apr-2026

## Project

- **Name**: Mahin CLI (formerly Movie CLI)
- **Type**: Go CLI application (NOT a web app)
- **Binary**: `mahin`
- **Language**: Go 1.22
- **Module**: `github.com/alimtvnetwork/movie-cli-v4`
- **Framework**: Cobra (CLI), SQLite (storage), TMDb API (metadata)
- **Current Version**: v2.23.0

## Purpose

A cross-platform CLI tool for managing a personal movie and TV show library. It scans local folders for video files, cleans messy filenames, fetches metadata from TMDb, stores everything in SQLite, and organizes files into configured directories.

## Key Architecture Decisions

1. **Pure-Go SQLite** (`modernc.org/sqlite`) — no CGo dependency
2. **WAL mode** for SQLite concurrency
3. **Single DB** — all tables in `mahin.db` (no Split DB)
4. **TMDb API** for metadata (requires user-provided API key)
5. **Console-safe self-update** — synchronous handoff via gitmap pattern, exit code propagation
6. **Data folder** at `<binary-dir>/data/` resolved via `os.Executable()`
7. **apperror.Wrap()** for all error wrapping (never fmt.Errorf)
8. **Zero nesting rule** — no nested ifs, max 2 conditions per if, no else after return

## Command Tree (21 commands)

```
mahin
├── hello                      # Greeting with version
├── version                    # Version/commit/build date + Go/OS info
├── update                     # Console-safe self-update (gitmap handoff)
├── changelog                  # Show changelog
└── movie
    ├── config                 # View/set configuration
    ├── scan                   # Scan folder → DB + TMDb + JSON metadata
    ├── ls                     # Paginated library list (file-backed only)
    ├── search                 # Live TMDb search → save
    ├── info                   # Local DB → TMDb fallback
    ├── suggest                # Recommendations/trending + genre discover
    ├── move                   # Browse + move + track (--all batch, cross-drive)
    ├── rename                 # Batch clean rename
    ├── undo                   # Revert last move/rename (with confirmation)
    ├── play                   # Open with default player
    ├── stats                  # Library statistics + file sizes
    ├── tag                    # Add/remove/list tags
    ├── export                 # Export library data
    ├── duplicates             # Detect duplicates by TMDb ID/filename/size
    ├── cleanup                # Find/remove stale DB entries
    └── watch                  # Watchlist: to-watch/watched tracking
```

## Important Notes for AI

- **This is NOT a web project** — no dev server, no preview
- Build errors in Lovable (`no package.json found`, `no command found for task "dev"`) are **expected and MUST be ignored**
- All file operations require a real OS/terminal to test
- Full specification lives in `spec/` folder
- **Always read memory files before making changes**
- **Always bump version/info.go after every code change**

## File Structure (as of 16-Apr-2026)

- `cmd/` — 21+ Go files (root, hello, version, update, changelog + movie parent + subcommands + helpers)
- `cleaner/` — filename cleaning
- `tmdb/` — API client (split: client.go, http.go, types.go, etc.)
- `db/` — 6+ files (open.go, media.go, config.go, history.go, helpers.go, tags.go, etc.)
- `updater/` — self-update: run.go, handoff.go, script.go, process.go
- `apperror/` — error wrapping (Wrap, New)
- `errlog/` — structured error logging
- `version/` — build-time vars (info.go)
- `.github/` — CI/CD pipelines
- `spec/` — structured specification docs
