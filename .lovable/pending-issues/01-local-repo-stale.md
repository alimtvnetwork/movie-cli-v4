# Local Repository Out of Sync

## Description
User's local repo is stuck at v2.14.0 (commit 3e4f3c3) while remote HEAD is at v2.23.0 (commit ccc7605). `git pull` doesn't update files — likely on a different branch or has local commits.

## Root Cause
Local branch is diverged from `origin/main`. Standard `git pull` does fast-forward only and fails silently when branches have diverged.

## Steps to Reproduce
1. Run `git log -1 --oneline` → shows old commit
2. Run `.\run.ps1` → build fails with unused imports that were already fixed in remote
3. `git pull` reports "Already up to date" despite being behind

## Attempted Solutions
- [x] Advised `git fetch origin && git reset --hard origin/main && git clean -fd` — user hasn't confirmed execution yet
- [ ] Manual file edit of tmdb/client.go and tmdb/http.go as workaround

## Priority
High — blocks all builds

## Blocked By
User action required — must run the git reset command.
