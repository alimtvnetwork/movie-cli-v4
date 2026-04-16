# 07 — Acceptance Criteria

## Purpose

GIVEN/WHEN/THEN test cases for the update command.

---

## AC-01: Basic Update

- GIVEN a clean repo with new commits on remote
- WHEN the user runs `movie update`
- THEN the binary pulls latest, rebuilds, deploys, and prints version change

## AC-02: Already Up to Date

- GIVEN the repo is at the latest commit
- WHEN the user runs `movie update`
- THEN it prints "Already up to date" and does NOT rebuild

## AC-03: No Git Installed

- GIVEN git is not in PATH
- WHEN the user runs `movie update`
- THEN it prints a clear error and exits 1

## AC-04: Local Changes Block Update

- GIVEN the repo has uncommitted local changes
- WHEN the user runs `movie update`
- THEN it refuses with "repository has local changes; commit or stash them"

## AC-05: Bootstrap Clone

- GIVEN no local repo exists anywhere
- WHEN the user runs `movie update`
- THEN it clones a fresh repo next to the binary
- AND reports bootstrap success (NOT "already up to date")
- AND tells the user to run `movie update` again to build

## AC-06: Handoff Copy Creation

- GIVEN any update scenario on Windows
- WHEN the update starts
- THEN a `movie-update-<pid>.exe` copy is created
- AND the worker runs from that copy (not the original)

## AC-07: Version Comparison

- GIVEN a successful update with version bump
- WHEN the build completes
- THEN it prints "Updated: v1.60.0 → v1.61.0"

## AC-08: Version Unchanged Warning

- GIVEN a successful update WITHOUT version bump
- WHEN the build completes
- THEN it warns "Version unchanged — was version/info.go bumped?"

## AC-09: Cleanup Auto-Run

- GIVEN a successful update
- WHEN the build and deploy complete
- THEN `movie update-cleanup` runs automatically
- AND removes temp binaries and .bak files

## AC-10: Manual Cleanup

- GIVEN leftover temp files from a previous update
- WHEN the user runs `movie update-cleanup`
- THEN it removes all `movie-update-*` and `*.bak` files
- AND does NOT delete the currently running binary

## AC-11: Repo Path Flag

- GIVEN the user passes `--repo-path /custom/path`
- WHEN the update starts
- THEN it uses that path instead of auto-resolution

## AC-12: Deploy Rollback

- GIVEN the deploy step fails (file locked, disk full, etc.)
- WHEN `run.ps1` cannot copy the new binary
- THEN it restores the `.bak` backup
- AND the user still has a working binary

## AC-13: Terminal Stays Attached

- GIVEN any update on any OS
- WHEN the handoff worker runs
- THEN all output appears in the user's terminal
- AND the user can see progress in real-time

## AC-14: Changelog Display

- GIVEN a successful update
- WHEN the version changes
- THEN `movie changelog --latest` output is displayed

---

*Acceptance criteria — updated: 2026-04-16*
