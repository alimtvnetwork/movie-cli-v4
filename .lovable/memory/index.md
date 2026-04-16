# Project Memory

## Core
Go 1.22 CLI project (NOT web). Binary: `mahin`. Module: `mahin-cli-v2`. Ignore Lovable build errors.
One file per command, max ~200 lines. Shared helpers in movie_info.go and movie_resolve.go.
File naming: `01-name-of-file.md`. Keep folder file counts small.
Plans & suggestions tracked in single files, not per-item files.
Never modify `.release` folder. Any code change bumps at least minor version.
ALWAYS bump version/info.go after every code change. Never forget.
Malaysia timezone (UTC+8) for timestamps. Milestones in `readm.txt`.
Root spec files: lowercase (spec.md, ai-handoff.md, development-log.md). Keep README.md uppercase.
HTML JS: single API_BASE variable for all REST calls. Never repeat URL.
Boolean names: never use negative words (un/not/no). Use positive semantic synonyms with Is/Has prefix.
Zero nested if. Max 2 conditions per if. No else after return. Functions ≤15 lines. Files ≤300 lines. Max 3 params.
No magic strings — use constants/enums. No fmt.Errorf — use apperror.Wrap().
Single DB: all tables in `mahin.db`. No Split DB. NEVER name file `<pkg>/<pkg>.go`.
Data folder at `<binary-dir>/data/`, resolved via os.Executable(). NOT cwd-relative.
Updater: synchronous console-safe handoff (gitmap pattern). Never async Start()+exit.
Current version: v2.23.0. Spec resequenced: foundation 01-06, app at 08, issues at 09.

## Memories
- [Project overview](mem://01-project-overview) — Go CLI, command tree (21 cmds), architecture, v2.23.0
- [Conventions](mem://02-conventions) — Code style, naming, build, deploy, config keys
- [Plan](mem://workflow/01-plan) — Done/pending task tracker, guideline remediation phases 3-7 next
- [AI success plan](mem://workflow/01-ai-success-plan) — 7 rules for 98% AI success rate
- [Suggestions](mem://suggestions/01-suggestions) — S01-S25 done, S26-S29 open (guideline fixes)
- [Reliability report](mem://reports/01-reliability-risk-report) — Failure map, corrective actions
- [No Split DB](mem://constraints/no-split-db) — All tables in single mahin.db
- [Data folder location](mem://features/data-folder-location) — Binary-relative data/ with single DB
- [Timestamp bug](mem://issues/01-timestamp-bug) — ✅ Fixed: hardcoded "now" → RFC3339
- [Duplicate TMDb fetch](mem://issues/02-duplicate-tmdb-fetch) — ✅ Fixed: shared helpers
- [Large files](mem://issues/03-large-files) — ✅ Fixed: split to <200 lines

- [Updater async console](mem://issues/05-updater-async-console) — ✅ Fixed: sync handoff, exit code propagation
- [Guideline violations](mem://issues/06-guideline-violations-refactoring) — ✅ Phase 1-2 done, phases 3-7 pending
