# 05 — Version Comparison

## Purpose

Define how the update process compares versions before and after the
build to give the user clear feedback.

---

## Version Source

The version is read by running:

```
movie version
```

Output format: `movie-cli v1.60.0 (commit: abc1234, built: 2026-04-16T10:00:00+08:00)`

The comparison uses the **full output string**, not just the semver tag.
This catches cases where the version tag is the same but the commit changed.

---

## Decision Table

| Old Version | New Version | Result | Message |
|-------------|-------------|--------|---------|
| Same string | Same string | Already up to date | `✔ Already up to date (v1.60.0)` |
| Different | Different | Success | `✨ Updated: v1.60.0 → v1.61.0` |
| Any | Build failed | Error | `❌ Build failed` (exit 1) |
| Any | Cannot read | Warning | `⚠ Cannot verify new version` |

---

## "Already Up to Date" Detection

The update checks `git pull` output BEFORE rebuilding:

```
if pullOutput matches "Already up to date":
    print "Already up to date"
    exit 0  # No rebuild needed
```

This saves the user from an unnecessary build cycle when there are no
new commits.

---

## Version Unchanged Warning

If the build succeeds but the version string is identical before and after:

```
⚠ WARNING: Version unchanged after update
  Was version/info.go bumped?
```

This catches the case where code changed but the developer forgot to bump
`version/info.go`. It is a **warning**, not an error — the binary was
still updated.

---

*Version comparison — updated: 2026-04-16*
