# Application Commands — Acceptance Criteria

**Version:** 2.0.0  
**Last Updated:** 2026-04-16  
**Format:** GIVEN/WHEN/THEN (E2E-test-ready)

---

## AC-01: Hello Command

**GIVEN** the CLI is installed  
**WHEN** the user runs `movie hello`  
**THEN** the output contains `👋 Hello from Movie CLI!` followed by the version string

---

## AC-02: Version Command

**GIVEN** the binary was built with `-ldflags` injecting version, commit, and build date  
**WHEN** the user runs `movie version`  
**THEN** the output shows `vX.Y.Z (commit: <hash>, built: <date>)`

**Edge Cases:**
- **GIVEN** the binary was built without ldflags **WHEN** `movie version` is run **THEN** defaults are shown: `v0.0.1-dev`, `none`, `unknown`

---

## AC-03: Self-Update Command

**GIVEN** the CLI resolves an existing clean git repository  
**WHEN** the user runs `movie self-update`  
**THEN** `git pull --ff-only` is executed and the old→new commit hashes are displayed

**Edge Cases:**
- **GIVEN** no local repo exists **WHEN** self-update runs **THEN** a fresh clone is created next to the binary and the output reports bootstrap success
- **GIVEN** git is not in PATH **WHEN** self-update runs **THEN** a clear error is shown: git not found
- **GIVEN** the working tree has uncommitted changes **WHEN** self-update runs **THEN** a clear error is shown: dirty working tree
- **GIVEN** there are no new commits in an existing repo **WHEN** self-update runs **THEN** the output says already up-to-date

---

## AC-04: Config Command

### AC-04a: Show All Config

**GIVEN** config values exist in the database  
**WHEN** `movie config` is run with no arguments  
**THEN** all config keys and values are printed  
**AND** the `tmdb_api_key` value is masked (first 4 + `...` + last 4 chars)

### AC-04b: Get Single Key

**GIVEN** a config key `movies_dir` exists  
**WHEN** `movie config get movies_dir`  
**THEN** the value of `movies_dir` is printed

### AC-04c: Set Key

**GIVEN** a valid config key  
**WHEN** `movie config set movies_dir /new/path`  
**THEN** the value is updated in the database  
**AND** a confirmation message is printed

**Edge Cases:**
- **GIVEN** an invalid config key **WHEN** `config get unknown_key` **THEN** an error is shown listing valid keys

---

## AC-05: Scan Command

**GIVEN** a directory contains video files and a TMDb API key is configured  
**WHEN** `movie scan /path/to/folder`  
**THEN** each video file is cleaned, metadata is fetched from TMDb, a poster is downloaded, and a media record is inserted into the database  
**AND** a summary is printed: total files, movies, TV shows, skipped  
**AND** a `scan_history` record is logged

**Edge Cases:**
- **GIVEN** no folder argument and no `scan_dir` config **WHEN** scan runs **THEN** an error is shown
- **GIVEN** a file's `original_file_path` already exists in the DB **WHEN** scan encounters it **THEN** the file is skipped (dedup)
- **GIVEN** no TMDb API key is available **WHEN** scan runs **THEN** a warning is printed but scanning continues without metadata
- **GIVEN** a directory entry is a subfolder containing a video file **WHEN** scan processes it **THEN** the directory name is used for title cleaning

---

## AC-06: List Command

**GIVEN** media records exist in the database  
**WHEN** `movie ls` is run  
**THEN** only records with a non-empty `current_file_path` are shown (file-backed items only)  
**AND** records created via `search` or `info` without local files are excluded  
**AND** a paginated list is shown with: number, clean title, year, rating (TMDb→IMDb fallback), type icon  
**AND** page size is read from config (default 20)

**Edge Cases:**
- **GIVEN** no file-backed media records exist **WHEN** `ls` is run **THEN** a "no media found" message is shown
- **GIVEN** 5 records exist but only 2 have `current_file_path` **WHEN** `ls` is run **THEN** only 2 items are listed
- **GIVEN** the user enters `N` **WHEN** on the last page **THEN** nothing happens or wraps to first page
- **GIVEN** the user enters a number **WHEN** on the list page **THEN** the detail view shows full metadata card

---

## AC-07: Search Command

**GIVEN** a TMDb API key is configured  
**WHEN** `movie search "The Matrix"`  
**THEN** up to 15 results are displayed with: number, icon, title, year, rating, type  
**AND** the user can select a result to fetch full details, download poster, and save to DB

**Edge Cases:**
- **GIVEN** no TMDb API key **WHEN** search runs **THEN** the command exits with "API key required" error
- **GIVEN** the user selects 0 **WHEN** at the selection prompt **THEN** the search is cancelled
- **GIVEN** search returns no results **WHEN** query has no matches **THEN** a "no results found" message is shown

---

## AC-08: Info Command

### AC-08a: By Numeric ID

**GIVEN** a media record with ID 5 exists in the DB  
**WHEN** `movie info 5`  
**THEN** the full detail card is displayed (title, year, type, ratings, genres, director, cast, description, paths)

### AC-08b: By Title String

**GIVEN** a media record titled "Inception" exists locally  
**WHEN** `movie info inception`  
**THEN** the match is resolved using priority: exact match → prefix match → first result  
**AND** the detail card is displayed

### AC-08c: Fallback to TMDb

**GIVEN** the title is not found in the local DB and a TMDb API key exists  
**WHEN** `movie info "Unknown Title"`  
**THEN** TMDb is searched, full details + credits + poster are fetched, the record is auto-saved, and the detail card is displayed

**Edge Cases:**
- **GIVEN** the TMDb result's `tmdb_id` already exists in the DB **WHEN** fallback runs **THEN** the existing record is returned (no duplicate)

---

## AC-09: Suggest Command

**GIVEN** a TMDb API key is configured  
**WHEN** `movie suggest 5`  
**THEN** the user is prompted to choose: Movie / TV / Random  
**AND** 5 suggestions are displayed with: title, year, rating, genre names

### AC-09a: Genre-Based Suggestions

**GIVEN** the user selects "Movie" or "TV" and the library has genre data  
**WHEN** suggestions are generated  
**THEN** `TopGenres()` is used, recommendations are fetched from TMDb, remaining slots filled with trending  
**AND** existing library items are excluded from results

### AC-09b: Random Suggestions

**GIVEN** the user selects "Random"  
**WHEN** suggestions are generated  
**THEN** trending movies + TV are merged, shuffled, and deduplicated

**Edge Cases:**
- **GIVEN** the library has no media **WHEN** genre-based suggestions run **THEN** fallback to trending only

---

## AC-10: Move Command

**GIVEN** a source directory contains video files  
**WHEN** `movie move /path/to/source`  
**THEN** video files are listed with clean titles, type icons, and file sizes  
**AND** the user selects a file and destination (Movies / TV / Archive / Custom)  
**AND** the file is renamed to `Title (Year).ext` and moved  
**AND** `move_history` is logged and media record is updated

**Edge Cases:**
- **GIVEN** source and destination are on different filesystems **WHEN** move runs **THEN** `io.Copy` + `os.Remove` fallback is used instead of `os.Rename`
- **GIVEN** the copy fails mid-transfer **WHEN** fallback is active **THEN** the source file is NOT deleted and an error is reported
- **GIVEN** no argument is provided **WHEN** move starts **THEN** interactive prompt offers: Downloads / Scan Dir / Custom path

### AC-10b: Batch Move

**GIVEN** a source directory with multiple video files  
**WHEN** `movie move /path --all`  
**THEN** all video files are moved to their destination using the selected category  
**AND** each file is renamed to `Title (Year).ext`  
**AND** all moves are logged to `move_history`

**Edge Cases:**
- **GIVEN** `--all` is used with a directory containing 0 video files **WHEN** move runs **THEN** "no video files found" is shown

---

## AC-11: Rename Command

**GIVEN** media records exist with `current_file_path` values  
**WHEN** `movie rename` is run  
**THEN** files whose names differ from `ToCleanFileName(cleanTitle, year, ext)` are listed as a preview  
**AND** user confirms with `y/N`  
**AND** confirmed renames are executed, DB paths updated, and `move_history` logged  
**AND** a summary is printed: `X/Y files renamed`

**Edge Cases:**
- **GIVEN** all files already have clean names **WHEN** rename runs **THEN** "nothing to rename" is shown
- **GIVEN** user enters `N` at confirmation **WHEN** prompted **THEN** no files are renamed

---

## AC-12: Undo Command

**GIVEN** a `move_history` record exists with `undone=0`  
**WHEN** `movie undo` is run  
**THEN** the most recent move is displayed (from → to paths)  
**AND** user confirms with `y/N`  
**AND** the file is moved back, the record is marked `undone=1`, and `current_file_path` is restored

**Edge Cases:**
- **GIVEN** no un-undone move records exist **WHEN** undo runs **THEN** "nothing to undo" is shown
- **GIVEN** the file no longer exists at `to_path` **WHEN** undo runs **THEN** an error is shown: file not found
- **GIVEN** user enters `N` at confirmation **WHEN** prompted **THEN** undo is cancelled with a message

---

## AC-13: Play Command

**GIVEN** a media record with ID 3 exists and `current_file_path` points to an existing file  
**WHEN** `movie play 3`  
**THEN** the file is opened with the platform-specific command (`open` / `xdg-open` / `cmd /c start`)

**Edge Cases:**
- **GIVEN** the media ID does not exist **WHEN** play runs **THEN** "media not found" error is shown
- **GIVEN** `current_file_path` does not exist on disk **WHEN** play runs **THEN** "file not found" error is shown

---

## AC-14: Stats Command

**GIVEN** media records exist in the database  
**WHEN** `movie stats` is run  
**THEN** the output shows: total movies, total TV shows, total count  
**AND** storage section shows: total file size, largest file, smallest file, average file size (human-readable format)  
**AND** top 10 genres with visual bar chart (`█` characters, max 30 width)  
**AND** average IMDb rating and average TMDb rating (if available)

**Edge Cases:**
- **GIVEN** no media records exist **WHEN** stats runs **THEN** all counts show 0, no storage section, and no genre chart
- **GIVEN** media exists but all have `file_size = 0` **WHEN** stats runs **THEN** the storage section is not displayed

---

## AC-15: Tag Command

### AC-15a: Add Tag

**GIVEN** media with ID 1 exists  
**WHEN** `movie tag add 1 favorite`  
**THEN** the tag is inserted into the `tags` table  
**AND** confirmation: `Tag "favorite" added to "Title (Year)"`

### AC-15b: Duplicate Tag

**GIVEN** tag "favorite" already exists on media ID 1  
**WHEN** `movie tag add 1 favorite`  
**THEN** error: `tag already exists`

### AC-15c: Remove Tag

**GIVEN** tag "favorite" exists on media ID 1  
**WHEN** `movie tag remove 1 favorite`  
**THEN** the tag is deleted and confirmation is printed

### AC-15d: Remove Non-Existent Tag

**GIVEN** tag "unknown" does not exist on media ID 1  
**WHEN** `movie tag remove 1 unknown`  
**THEN** error: `tag not found`

### AC-15e: List Tags for Media

**GIVEN** tags exist on media ID 1  
**WHEN** `movie tag list 1`  
**THEN** all tags for that media are shown

### AC-15f: List All Tags

**GIVEN** tags exist across multiple media  
**WHEN** `movie tag list` (no ID)  
**THEN** all unique tags are shown with media count, e.g., `favorite (3)`

---

## AC-16: Export Command

**GIVEN** media records exist in the database  
**WHEN** `movie export` is run  
**THEN** all media records are serialized to JSON and written to `./data/json/export/media.json`  
**AND** output shows: `Exported N items → <path>`

### AC-16a: Custom Output Path

**GIVEN** media records exist  
**WHEN** `movie export -o ~/Desktop/library.json`  
**THEN** the JSON is written to the specified path instead of the default

**Edge Cases:**
- **GIVEN** no media records exist **WHEN** export runs **THEN** message: "No media to export"
- **GIVEN** the output directory does not exist **WHEN** export runs **THEN** the directory is created automatically
- **GIVEN** the output path is read-only **WHEN** export runs **THEN** an error is shown

---

## AC-17: Redo Command

**GIVEN** a move or action has been reverted via `movie undo`  
**WHEN** `movie redo` is run  
**THEN** the most recent reverted operation is displayed with details  
**AND** user confirms with `y/N`  
**AND** the operation is re-applied (file moved forward, or media re-inserted/re-deleted)  
**AND** the record is marked as restored

**Edge Cases:**
- **GIVEN** `--list` flag is used **WHEN** redo runs **THEN** all redoable operations are listed (moves + actions)
- **GIVEN** `--batch` flag is used **WHEN** redo runs **THEN** the entire last reverted batch is redone in original order
- **GIVEN** `--id <id>` flag is used **WHEN** redo runs **THEN** the specific action_history record is redone
- **GIVEN** no reverted operations exist **WHEN** redo runs **THEN** "nothing to redo" is shown
- **GIVEN** the file no longer exists at `from_path` **WHEN** redo runs **THEN** an error is shown

---

## AC-18: Popout Command

**GIVEN** a directory contains subfolders with nested video files  
**WHEN** `movie popout /path/to/folder`  
**THEN** nested video files are discovered up to `--depth` (default 3)  
**AND** files are moved to the root directory with clean filenames  
**AND** all moves are tracked in `move_history` with a shared batch ID  
**AND** after extraction, the user is offered folder cleanup

**Edge Cases:**
- **GIVEN** `--dry-run` flag is used **WHEN** popout runs **THEN** files are listed but not moved
- **GIVEN** `--no-rename` flag is used **WHEN** popout runs **THEN** original filenames are preserved
- **GIVEN** no nested video files found **WHEN** popout runs **THEN** "no nested videos found" is shown
- **GIVEN** a subfolder becomes empty after popout **WHEN** cleanup is offered **THEN** user can choose to delete empty folders

---

## AC-19: History Command

**GIVEN** state-changing operations have been performed  
**WHEN** `movie history` is run  
**THEN** a unified timeline of moves and actions is displayed, sorted by timestamp descending  
**AND** each entry shows: type icon, action type, detail, timestamp, revert status

**Edge Cases:**
- **GIVEN** `--type move` is used **WHEN** history runs **THEN** only move_history records are shown
- **GIVEN** `--type scan` is used **WHEN** history runs **THEN** only scan-related actions are shown
- **GIVEN** `--batch <id>` is used **WHEN** history runs **THEN** only records from that batch are shown
- **GIVEN** `--since 2026-04-01` is used **WHEN** history runs **THEN** only records after that date are shown
- **GIVEN** `--format json` is used **WHEN** history runs **THEN** output is valid JSON array
- **GIVEN** no history records exist **WHEN** history runs **THEN** "no history found" is shown

---

## AC-20: Rescan Command

**GIVEN** media entries exist with missing genre, rating, or description  
**WHEN** `movie rescan` is run  
**THEN** entries with incomplete metadata are re-fetched from TMDb  
**AND** each updated entry is logged in action_history as `RescanUpdate` with a pre-update snapshot  
**AND** a summary is printed: `X entries updated`

**Edge Cases:**
- **GIVEN** `--all` flag is used **WHEN** rescan runs **THEN** all media entries are re-fetched, not just incomplete ones
- **GIVEN** `--limit 10` is used **WHEN** rescan runs **THEN** at most 10 entries are processed
- **GIVEN** no TMDb API key is configured **WHEN** rescan runs **THEN** an error is shown
- **GIVEN** all entries already have complete metadata **WHEN** rescan runs (without `--all`) **THEN** "no entries need updating" is shown

---

## AC-21: REST Command

**GIVEN** media records exist in the database  
**WHEN** `movie rest` is run  
**THEN** an HTTP server starts on port 8086 (or `--port`)  
**AND** endpoints are available: `GET /api/media`, `GET /api/media/:id`, `GET /api/stats`  
**AND** a browser is automatically opened to the HTML report page  
**AND** CORS headers are set for local development

**Edge Cases:**
- **GIVEN** port 8086 is already in use **WHEN** rest starts **THEN** an error is shown
- **GIVEN** `--port 9000` is used **WHEN** rest starts **THEN** the server binds to port 9000

---

## AC-22: Cleanup Command

**GIVEN** media entries exist in the database  
**WHEN** `movie cleanup` is run  
**THEN** entries whose `current_file_path` no longer exists on disk are listed (dry-run preview)

**Edge Cases:**
- **GIVEN** `--remove` flag is used **WHEN** cleanup runs **THEN** stale entries are deleted from the database with confirmation
- **GIVEN** all entries have valid file paths **WHEN** cleanup runs **THEN** "no stale entries found" is shown

---

## AC-23: Duplicates Command

**GIVEN** media entries exist in the database  
**WHEN** `movie duplicates` is run  
**THEN** duplicate entries are detected (default: by TMDb ID) and displayed in groups

**Edge Cases:**
- **GIVEN** `--by filename` is used **WHEN** duplicates runs **THEN** duplicates are matched by original filename
- **GIVEN** `--by filesize` is used **WHEN** duplicates runs **THEN** duplicates are matched by file size
- **GIVEN** no duplicates exist **WHEN** duplicates runs **THEN** "no duplicates found" is shown

---

## AC-24: Logs Command

**GIVEN** error log entries exist in the database  
**WHEN** `movie logs` is run  
**THEN** recent log entries are displayed with: timestamp, level, source, message  
**AND** most recent entries shown first

**Edge Cases:**
- **GIVEN** `--level ERROR` is used **WHEN** logs runs **THEN** only ERROR-level entries are shown
- **GIVEN** no log entries exist **WHEN** logs runs **THEN** "no log entries found" is shown

---

## AC-25: Watch Command

### AC-25a: Add to Watchlist

**GIVEN** media with ID 5 exists  
**WHEN** `movie watch add 5`  
**THEN** the media is added to the watchlist with status "to-watch"

### AC-25b: Mark as Watched

**GIVEN** media ID 5 is on the watchlist  
**WHEN** `movie watch done 5`  
**THEN** the watchlist status is updated to "watched"

### AC-25c: Undo Watch

**GIVEN** media ID 5 is marked "watched"  
**WHEN** `movie watch undo 5`  
**THEN** the status reverts to "to-watch"

### AC-25d: Remove from Watchlist

**GIVEN** media ID 5 is on the watchlist  
**WHEN** `movie watch rm 5`  
**THEN** the media is removed from the watchlist

### AC-25e: List Watchlist

**GIVEN** watchlist entries exist  
**WHEN** `movie watch ls`  
**THEN** all watchlist items are shown with: title, year, status, date added

**Edge Cases:**
- **GIVEN** media ID does not exist **WHEN** `watch add` runs **THEN** "media not found" error is shown
- **GIVEN** media is already on watchlist **WHEN** `watch add` runs again **THEN** "already on watchlist" error is shown
- **GIVEN** no watchlist entries exist **WHEN** `watch ls` runs **THEN** "watchlist is empty" is shown

---

## AC-26: CD Command

**GIVEN** a folder has been previously scanned  
**WHEN** `movie cd Movies`  
**THEN** the full resolved path matching "Movies" is printed to stdout  
**AND** can be used with `cd $(movie cd Movies)` for shell navigation

**Edge Cases:**
- **GIVEN** no folder matches the query **WHEN** cd runs **THEN** an error is shown
- **GIVEN** multiple folders match **WHEN** cd runs **THEN** the best match is printed

---

## AC-27: DB Command

**GIVEN** the CLI is installed  
**WHEN** `movie db` is run  
**THEN** the full resolved path to `mahin.db` and the data directory is printed

---

## AC-28: Changelog Command

**GIVEN** `CHANGELOG.md` exists next to the binary  
**WHEN** `movie changelog` is run  
**THEN** the full changelog content is printed

**Edge Cases:**
- **GIVEN** `--latest` flag is used **WHEN** changelog runs **THEN** only the most recent version block is shown
- **GIVEN** `CHANGELOG.md` does not exist **WHEN** changelog runs **THEN** "changelog not found" is shown

---

## Cross-References

- [Overview](./00-overview.md)
- [Project Spec](./01-project-spec.md)
- [Error Management AC](../02-error-manage-spec/97-acceptance-criteria.md)
- [Seedable Config AC](../04-seedable-config-architecture/98-acceptance-criteria.md)
