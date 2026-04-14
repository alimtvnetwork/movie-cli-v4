---
name: Data folder location
description: Data folder is relative to binary location, not cwd — uses os.Executable()
type: feature
---
The CLI's data folder (`data/`) is always created relative to where the binary physically resides, not the current working directory.

- Resolved via `os.Executable()` + `filepath.EvalSymlinks()` at runtime
- If binary is at `E:\bin-run\movie.exe`, data lives in `E:\bin-run\data\`
- `run.ps1` deploys binary to the configured deploy path; data is auto co-located
- No environment variables needed — purely resolved from binary location
