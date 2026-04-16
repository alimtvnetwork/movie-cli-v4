# 02 — Repo Path Resolution

## Purpose

Define how `movie update` locates the source repository on disk.

---

## Resolution Order

The update command checks these locations in order:

| Priority | Source | How |
|----------|--------|-----|
| 1 | `--repo-path` flag | User explicitly passes the path |
| 2 | Binary's own directory | `os.Executable()` → check for `.git` and `go.mod` |
| 3 | Sibling clone | `<binary-dir>/movie-cli-v4/` |
| 4 | Current working directory | `os.Getwd()` → check for `.git` and `go.mod` |
| 5 | Bootstrap clone | Clone fresh next to the binary |

---

## Validation

A candidate path is valid only if it contains BOTH:
- A `.git` directory (it's a git repo)
- A `go.mod` file with module `github.com/alimtvnetwork/movie-cli-v4`

This prevents false matches on unrelated git repos.

---

## Bootstrap Clone

If no repo is found at any location:

1. Print: `📥 No local repo found. Cloning to: <binary-dir>/movie-cli-v4/`
2. Run: `git clone --depth 1 https://github.com/alimtvnetwork/movie-cli-v4.git`
3. Report **bootstrap success** — NOT "already up to date"
4. Tell user to run `movie update` again to build

A fresh clone is a special case. The binary cannot rebuild itself on first
clone because the handoff copy was made from the OLD binary. The user must
run `movie update` a second time (or `pwsh run.ps1` manually).

---

## `--repo-path` Flag

```
movie update --repo-path C:\dev\movie-cli-v4
```

When provided, this flag overrides all automatic resolution. The path is:
1. Trimmed of whitespace and quotes
2. `~` expanded to home directory
3. Resolved to absolute path
4. Validated for `.git` + `go.mod`

---

## Pseudocode

```go
func resolveRepoPath() string {
    // 1. Flag
    if flagPath := getFlagValue("--repo-path"); len(flagPath) > 0 {
        return validateRepoPath(flagPath)
    }

    // 2. Binary directory
    exe, _ := os.Executable()
    exeDir := filepath.Dir(exe)
    if isValidRepo(exeDir) {
        return exeDir
    }

    // 3. Sibling clone
    sibling := filepath.Join(exeDir, "movie-cli-v4")
    if isValidRepo(sibling) {
        return sibling
    }

    // 4. CWD
    cwd, _ := os.Getwd()
    if isValidRepo(cwd) {
        return cwd
    }

    // 5. Bootstrap clone
    return bootstrapClone(exeDir)
}
```

---

*Repo path resolution — updated: 2026-04-16*
