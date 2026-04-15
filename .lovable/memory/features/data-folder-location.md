---
name: Data folder location
description: Data folder is relative to binary location, not cwd — uses os.Executable(). Split DB layout with config/ and log/ subfolders.
type: feature
---
The CLI's data folder (`data/`) is always created relative to where the binary physically resides, not the current working directory.

- Resolved via `os.Executable()` + `filepath.EvalSymlinks()` at runtime
- If binary is at `E:\bin-run\movie.exe`, data lives in `E:\bin-run\data\`
- `run.ps1` deploys binary to the configured deploy path; data is auto co-located
- No environment variables needed — purely resolved from binary location

### Split DB Files
- `data/media.db` — Media & operations (core tables, lookups, history)
- `data/watchlist.db` — Watchlist tracking
- `data/config.db` — Key-value CLI settings
- `data/error-log.db` — Structured error logs

### Subfolders
- `data/config/` — CLI configuration files (preserved across DB resets)
- `data/log/log.txt` — General application log
- `data/log/error.log` — Error-only log (see error handling spec)
- `data/thumbnails/<slug>/` — Downloaded poster images
- `data/json/` — JSON metadata and move history logs
