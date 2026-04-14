# 10 — CLI Output Format Specification

> Version: 1.0 | Updated: 2026-04-14

## Overview

This spec defines the visual output format for the `movie scan` command.
The format is inspired by [gitmap](https://github.com/alimtvnetwork/gitmap-v2)
and provides a structured, human-readable terminal experience with numbered
items, categorized icons, ratings, and a tree-style output file listing.

---

## 1. Header Block

A double-line box banner is printed at the top of every scan:

```
  ╔══════════════════════════════════════╗
  ║         🎬  Movie CLI Scanner        ║
  ╚══════════════════════════════════════╝

  📂 Scanning: /path/to/folder
  🔄 Mode: recursive (all subdirectories)
  📁 Output: /path/to/folder/.movie-output
```

### Rules
- Box width: fixed at 40 chars inner
- Mode line only appears if `--recursive` is used
- Output line is hidden in `--dry-run` mode
- All lines indented with 2 spaces

---

## 2. Scanned Items Section

```
  ■ Scanned Items
  ──────────────────────────────────────────
```

Each item is printed as a numbered entry with an icon indicating type:

```
  1. 🎬 Inception (2010) [movie]
     └─ Inception.2010.1080p.BluRay.x264.mkv
     ⭐ 8.4  Inception

  2. 📺 Breaking Bad (2008) [tv]
     └─ Breaking.Bad.S01E01.720p.mkv
     ⭐ 9.5  Breaking Bad

  3. 🎬 The Dark Knight (2008) [movie]
     └─ The.Dark.Knight.2008.REMUX.mkv
     ⏩ Already in database, skipping
```

### Item Format

```
  {index}. {icon} {clean_title} ({year}) [{type}]
     └─ {original_filename}
     ⭐ {rating}  {tmdb_title}
```

### Icons
| Type    | Icon |
|---------|------|
| Movie   | 🎬   |
| TV Show | 📺   |

### Metadata Lines (after `└─`)
| Condition         | Line                                 |
|-------------------|--------------------------------------|
| TMDb matched      | `⭐ {rating}  {title}`               |
| Thumbnail saved   | `🖼️  Thumbnail saved`               |
| Already in DB     | `⏩ Already in database, skipping`    |
| TMDb warning      | `⚠️  no TMDb match for '{query}'`    |

---

## 3. Summary Section

```
  ■ Summary
  ──────────────────────────────────────────
  📊 Scan Complete!
     Total files: 25
     Movies:      18
     TV Shows:    7
     Skipped:     3 (already in DB)
```

### Rules
- "Skipped" line only appears if `skipped > 0`
- In dry-run mode, title changes to "Dry Run Complete!"
- In dry-run mode, a tip line is appended:
  `💡 Run without --dry-run to actually scan and save.`

---

## 4. Output Files Section

Only shown when NOT in dry-run mode:

```
  ■ Output Files
  ──────────────────────────────────────────
  📁 /path/to/.movie-output/
  ├── 📄 summary.json      Scan report with metadata
  ├── 🌐 report.html       Interactive HTML report
  ├── 📁 json/movie/       Per-movie JSON metadata
  ├── 📁 json/tv/          Per-show JSON metadata
  └── 📁 thumbnails/       Movie poster thumbnails
```

### File Descriptions
| File/Dir         | Description                      |
|------------------|----------------------------------|
| `summary.json`   | Full scan report with counts     |
| `report.html`    | Interactive filterable HTML UI   |
| `json/movie/`    | One JSON file per movie          |
| `json/tv/`       | One JSON file per TV show        |
| `thumbnails/`    | Poster images: `{slug}-{id}.jpg` |

---

## 5. REST Server Section (--rest flag)

When `--rest` is specified, after the scan completes:

```
  🚀 Starting REST server on http://localhost:8086 ...
```

The server starts, the default browser opens the HTML report, and the CLI
blocks until Ctrl+C is pressed.

---

## 6. Thumbnail Naming Convention

Thumbnails are saved to `.movie-output/thumbnails/` with the format:

```
{slug}-{tmdb_id}.jpg
```

Where `slug` = `ToSlug(clean_title)` + optional `-{year}`.

Examples:
- `inception-2010-27205.jpg`
- `breaking-bad-2008-1396.jpg`
- `the-dark-knight-2008-155.jpg`

---

## 7. JSON Output Mode (`--format json`)

When `--format json` is used, NO visual output is printed. Instead, a
single JSON object is written to stdout:

```json
{
  "scan_dir": "/path/to/folder",
  "total": 25,
  "movies": 18,
  "tv_shows": 7,
  "skipped": 3,
  "items": [...]
}
```

---

## 8. Table Output Mode (`--format table`)

Table headers and rows are printed using fixed-width columns. See
`movie_scan_table.go` for the column definitions.

---

## 9. Error Handling in Output

- Errors during scan are logged via `errlog` to:
  1. `.movie-output/logs/error.txt` (flat file with stack traces)
  2. `error_logs` table in SQLite database
- Errors are NOT printed to the main scan output unless they are
  user-facing (e.g., "folder not found")
- Warnings (TMDb miss, thumbnail fail) print inline with the item

---

## 10. REST API Request Logging

When the REST server is running (via `movie rest` or `movie scan --rest`),
every HTTP request is logged via `errlog.Info`:

```
[REST] GET /api/media → 200 (2.3ms)
[REST] PATCH /api/media/5/watched → 200 (1.1ms)
```

Format: `[REST] {METHOD} {PATH} → {STATUS} ({DURATION})`
