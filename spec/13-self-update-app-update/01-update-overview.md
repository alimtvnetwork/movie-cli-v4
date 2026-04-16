# 01 — Update Overview

## Purpose

Explain the update architecture, why it is non-trivial, and how the
copy-and-handoff pattern solves the Windows file-lock problem.

---

## The Problem

When a user runs `movie update`, the tool must:

1. Find the source repository on disk.
2. Pull the latest code from GitHub.
3. Build a new binary with version metadata.
4. Replace the currently running binary with the new one.
5. Verify the update succeeded.
6. Clean up temporary artifacts.

**Step 4 is the hard part.** On Windows, a running `.exe` file is locked by
the OS — it cannot be overwritten or deleted while the process is alive.

---

## Platform Behavior

| Operation | Windows | Linux / macOS |
|-----------|---------|---------------|
| Overwrite running binary | ❌ Blocked (file lock) | ✅ Works |
| Rename running binary | ✅ Allowed | ✅ Works |
| Delete running binary | ❌ Blocked | ✅ Works |
| Replace after rename | ✅ Works | ✅ Works |

**Key insight**: Windows allows **renaming** a running executable but not
**overwriting** or **deleting** it.

---

## Solution: Copy-and-Handoff (Three Layers)

### Layer 1 — Copy and Re-launch

1. The running binary copies itself to a temp location:
   `movie-update-<pid>.exe` (same directory as binary, fallback to `%TEMP%`).
2. It launches the temp copy with a hidden `update-runner` command.
3. The parent **waits for the worker** using foreground/blocking execution
   (`cmd.Run()`, NOT `cmd.Start()` + `os.Exit()`).
4. This keeps the terminal session stable — no detached processes.

```
func runUpdate():
    repoPath = resolveRepoPath()
    selfPath = os.Executable()
    copyPath = createHandoffCopy(selfPath)
    launchHandoff(copyPath, repoPath)  // Blocking — keeps terminal stable
    // Parent exits naturally after worker completes
```

### Layer 2 — Skip-if-Current + Build Pipeline

The temp copy (worker) orchestrates the actual update:

1. **Capture current version** from the deployed binary.
2. **Generate a temp script** that calls `run.ps1` (or `run.sh`).
3. The script runs: `git pull` → checks "Already up to date" → if no changes,
   exit early (no rebuild). If changes exist → `run.ps1 -NoPull` (already
   pulled).
4. **Wait 1.2 seconds** for the parent process to fully release file handles.
5. **Compare versions** — warn if unchanged, confirm if different.

```
# Generated PowerShell script (simplified)
$oldVersion = & movie version 2>&1

cd $repoPath
$pullOutput = git pull 2>&1
if ($pullOutput -match "Already up to date") {
    Write-Host "No update needed — already running latest version"
    exit 0
}

Start-Sleep -Seconds 1.2
& ./run.ps1 -NoPull    # Already pulled above

$newVersion = & movie version 2>&1
if ($oldVersion -eq $newVersion) {
    Write-Host "WARNING: Version unchanged — was version/info.go bumped?"
} else {
    Write-Host "Updated: $oldVersion -> $newVersion"
}
```

### Layer 3 — Deploy with Rollback (handled by run.ps1)

The existing `run.ps1` already implements rename-first deploy:

1. **Backup** existing binary: `movie.exe` → `movie.exe.bak`
2. **Copy** new binary to deploy path
3. **On failure** → rename `.bak` back (rollback)
4. **On success** → delete `.bak`

This is already implemented in `run.ps1` lines 522–584.

---

## Command Rename: `self-update` → `update`

| Before | After |
|--------|-------|
| `movie self-update` | `movie update` |
| `movie update` (alias) | `movie update` (primary) |
| No cleanup command | `movie update-cleanup` |
| No hidden runner | `movie update-runner` (hidden) |

The command is renamed to match gitmap-v2 convention. `self-update` is
removed entirely — no alias.

---

## What Changes from Current Implementation

| Aspect | Current | New |
|--------|---------|-----|
| Command name | `self-update` (alias: `update`) | `update` (no alias) |
| What it does | `git pull` only, tells user to rebuild | Full pull → build → deploy |
| File lock handling | None | Copy-and-handoff |
| Version comparison | Shows git commit SHAs | Shows semver before/after |
| Bootstrap clone | Yes (clones if no repo) | Yes (kept) |
| Cleanup | None | `update-cleanup` subcommand |
| run.ps1 integration | None (manual rebuild) | Automatic via generated script |

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Git not installed | Print error, exit 1 |
| No repo path found | Prompt user, or clone fresh |
| Repo has local changes | Refuse with clear message |
| Already up to date | Print message, exit 0 (no rebuild) |
| Build fails | Script exits with error, backup remains |
| Deploy locked after retries | Restore backup, fail with message |
| Temp copy fails to create | Print error, exit 1 |
| Version unchanged after update | Warn user (version constant not bumped) |

---

## Key Learnings (from gitmap-v2)

1. **`exit()` doesn't release locks instantly** — add 1.2s delay before overwrite.
2. **The parent MUST use `cmd.Run()` (foreground/blocking)** — NEVER `cmd.Start()` + `os.Exit(0)`.
3. **Use rename-first, not copy-first** for PATH binary sync.
4. **Bump the version on every change** so the user can confirm the update applied.
5. **Never add `Read-Host`** to generated scripts — they run in non-interactive sessions.
6. **Always provide a rollback path** — `.bak` files serve as safety net.

---

*Update overview — updated: 2026-04-16*
