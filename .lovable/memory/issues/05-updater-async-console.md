# Issue: Updater Async Console Breakage

> **Status**: ✅ Resolved  
> **Severity**: High  
> **Files**: `cmd/update.go`, `updater/handoff.go`, `updater/run.go`, `updater/process.go`, `updater/script.go`  
> **Iteration**: 3 (fixed 16-Apr-2026)

## Root Cause

The `movie update` command used `cmd.Start()` to launch the handoff process and immediately exited the parent. This caused:
1. Console detachment — child process lost stdout/stderr
2. No exit code propagation — parent always exited 0
3. Broken console on Windows — user sees garbled output

## Solution Applied

Refactored to **synchronous foreground execution** per gitmap console-safe-handoff spec:
- Created `updater/process.go` with `runAttached()` helper — attaches Stdout, Stderr, Stdin
- `handoff.go` and `run.go` return errors through call stack instead of calling `os.Exit` internally
- `cmd/update.go` has `exitOnUpdateError()` that catches `*exec.ExitError` and propagates exit code
- All child processes run via `cmd.Run()` (blocking), never `cmd.Start()` + parent exit

## Learning

- Self-update must be synchronous — the parent process must wait for the child
- Exit code propagation requires `errors.As(err, &exitErr)` pattern
- Console attachment requires explicit `cmd.Stdout = os.Stdout` etc.

## What Not to Repeat

- Never use `cmd.Start()` + `os.Exit(0)` for self-update handoff
- Never swallow child process exit codes
- Always follow gitmap console-safe-handoff spec for update commands
