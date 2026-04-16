# Self-Update Specification

## Purpose

This folder defines the **update command architecture** for the movie CLI tool.
The `movie update` command pulls the latest source, rebuilds the binary via
`run.ps1`, and deploys it — all without the user leaving the terminal.

The design follows the **copy-and-handoff** pattern from
[gitmap-v2](https://github.com/alimtvnetwork/gitmap-v2/blob/main/spec/03-general/03-self-update-mechanism.md)
to solve the Windows file-lock problem where a running binary cannot overwrite
itself.

> **Reference implementation**: gitmap-v2
> ([self-update mechanism](https://github.com/alimtvnetwork/gitmap-v2/blob/main/spec/03-general/03-self-update-mechanism.md),
> [update.go](https://github.com/alimtvnetwork/gitmap-v2/blob/main/gitmap/cmd/update.go),
> [updatescript.go](https://github.com/alimtvnetwork/gitmap-v2/blob/main/gitmap/cmd/updatescript.go),
> [updatecleanup.go](https://github.com/alimtvnetwork/gitmap-v2/blob/main/gitmap/cmd/updatecleanup.go))

---

## Documents

| File | Topic |
|------|-------|
| [01-update-overview.md](./01-update-overview.md) | Architecture, flow diagram, platform behavior |
| [02-repo-path-resolution.md](./02-repo-path-resolution.md) | How the binary finds its source repo |
| [03-copy-and-handoff.md](./03-copy-and-handoff.md) | The handoff mechanism that bypasses file locks |
| [04-update-script-generation.md](./04-update-script-generation.md) | Generated PowerShell/Bash scripts that run `run.ps1` |
| [05-version-comparison.md](./05-version-comparison.md) | Before/after version check and skip-if-current |
| [06-cleanup.md](./06-cleanup.md) | `update-cleanup` subcommand for temp artifacts |
| [07-acceptance-criteria.md](./07-acceptance-criteria.md) | GIVEN/WHEN/THEN test cases |

---

## Core Principle

A running binary **cannot overwrite itself** on Windows. The entire update
architecture exists to work around this constraint while maintaining a
seamless user experience.

## Command Surface

| Command | Description |
|---------|-------------|
| `movie update` | Pull latest code, rebuild, and deploy the binary |
| `movie update-cleanup` | Remove leftover temp binaries and `.old` backups |
| `movie update-runner` | **Hidden** — the worker command run by the handoff copy |

## Flow Summary

```
User runs: movie update
  │
  ├─ Resolves repo path (binary dir → CWD → sibling clone → prompt)
  ├─ Copies self → movie-update-<pid>.exe (same dir, fallback %TEMP%)
  ├─ Launches copy with: movie update-runner --repo-path <path> (foreground/blocking)
  ├─ Parent waits for worker to complete (terminal stays attached)
  │
  └─ Worker (update-runner) starts
      ├─ Captures current deployed version: movie version
      ├─ On Windows: writes temp .ps1 script → runs run.ps1 -NoPull first, then handles pull
      ├─ On Unix: runs run.sh --update (or run.ps1 via pwsh)
      │   ├─ git pull --ff-only
      │   ├─ go mod tidy
      │   ├─ go build with ldflags
      │   ├─ Deploy with rename-first (backup .old → copy new → rollback on failure)
      │   └─ Verify deployed binary
      ├─ Compares old vs new version
      │   ├─ Same version → warn (version constant not bumped?)
      │   └─ Different → print "Updated: v1.60.0 → v1.61.0"
      ├─ Runs: movie changelog --latest (show what changed)
      ├─ Runs: movie update-cleanup (auto)
      └─ Cleans up temp script
```

## Placeholders

| Placeholder | Meaning | Movie CLI Value |
|-------------|---------|-----------------|
| `<binary>` | CLI binary name | `movie` (or `movie.exe`) |
| `<deploy-dir>` | Install directory | From `powershell.json` `deployPath` |
| `<repo-root>` | Source repository root | Directory containing `go.mod` |
| `<module>` | Go module path | `github.com/alimtvnetwork/movie-cli-v4` |

---

*Self-update spec — updated: 2026-04-16*
