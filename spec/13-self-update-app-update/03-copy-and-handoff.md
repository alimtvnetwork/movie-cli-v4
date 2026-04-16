# 03 — Copy-and-Handoff Mechanism

## Purpose

Define how the running binary creates a temporary copy of itself and
delegates the update work to that copy, bypassing Windows file locks.

---

## Why Handoff is Needed

On Windows, a running `.exe` holds a file lock. If the build pipeline tries
to overwrite the binary while the original process is still alive, the OS
blocks it. The handoff solves this by:

1. The running binary is Process A (holds the lock on `movie.exe`)
2. Process A copies itself to `movie-update-<pid>.exe` (a different file)
3. Process A launches Process B from the copy (foreground/blocking)
4. Process B runs `run.ps1` which deploys the NEW binary to `movie.exe`
   - Since Process A is waiting (not terminated), it still holds the lock
   - But `run.ps1` uses rename-first: `movie.exe` → `movie.exe.bak`, then
     copies new binary. Rename is allowed even on locked files.
5. Process B finishes → Process A exits naturally

---

## Copy Location

The handoff copy is placed at:

1. **Same directory as the binary** (preferred):
   `<binary-dir>/movie-update-<pid>.exe`
2. **Fallback to temp directory**:
   `%TEMP%/movie-update-<pid>.exe` (Windows)
   `/tmp/movie-update-<pid>` (Unix)

The PID suffix ensures uniqueness if multiple updates run simultaneously.

---

## File Naming

| OS | Format |
|----|--------|
| Windows | `movie-update-<pid>.exe` |
| Linux/macOS | `movie-update-<pid>` |

---

## Launch Arguments

The parent launches the copy with:

```
movie-update-12345.exe update-runner --repo-path /path/to/repo
```

- `update-runner` is a **hidden command** (not shown in help)
- `--repo-path` passes the resolved repo path to avoid re-resolution

---

## Foreground Execution (CRITICAL)

The parent MUST use **`cmd.Run()`** (blocking) to launch the worker:

```go
cmd := exec.Command(copyPath, "update-runner", "--repo-path", repoPath)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Stdin = os.Stdin
err := cmd.Run()  // ← BLOCKING — terminal stays attached
```

**NEVER** use `cmd.Start()` + `os.Exit(0)`. This detaches the terminal and
breaks the user experience — they lose all output from the update process.

---

## Exit Code Propagation

If the worker exits with a non-zero code, the parent must propagate it:

```go
if err != nil {
    var exitErr *exec.ExitError
    if errors.As(err, &exitErr) {
        os.Exit(exitErr.ExitCode())
    }
    os.Exit(1)
}
```

---

## Unix Simplification

On Linux/macOS, the handoff is technically unnecessary because the OS
allows in-place binary replacement. However, we use the same handoff
pattern on all platforms for code simplicity and consistency.

---

## Pseudocode

```go
func createHandoffCopy(selfPath string) string {
    name := fmt.Sprintf("movie-update-%d", os.Getpid())
    if runtime.GOOS == "windows" {
        name += ".exe"
    }

    // Try same directory first
    copyPath := filepath.Join(filepath.Dir(selfPath), name)
    if err := copyFile(selfPath, copyPath); err == nil {
        makeExecutable(copyPath)
        return copyPath
    }

    // Fallback to temp
    copyPath = filepath.Join(os.TempDir(), name)
    if err := copyFile(selfPath, copyPath); err != nil {
        fmt.Fprintf(os.Stderr, "Cannot create handoff copy: %v\n", err)
        os.Exit(1)
    }
    makeExecutable(copyPath)
    return copyPath
}
```

---

*Copy-and-handoff mechanism — updated: 2026-04-16*
