# Strictly Avoided Patterns

**Updated:** 2026-04-16

---

- **Split DB (multiple .db files):** All tables must live in single `mahin.db`. Never split into media.db, watchlist.db, config.db, etc. See: `.lovable/memory/constraints/no-split-db.md`
- **Negative boolean names:** Never use `un/not/no` in boolean names. Use positive semantic synonyms with Is/Has prefix (e.g., IsReverted not IsUndone). See: memory index `boolean-no-negative-words`
- **CI log commit-back:** CI must NEVER commit/push back to the repo. Kill feature entirely if loops occur. See: `.lovable/memory/issues/04-ci-log-commit-loop`
- **Async updater handoff:** Never use `cmd.Start()` + parent exit for self-update. Must be synchronous with exit code propagation. See: gitmap console-safe-handoff spec.
- **Placeholder values in production:** Never use `"now"`, `"TODO"`, `"test"` as literal values in code. Always use real data. See: `.lovable/memory/issues/01-timestamp-bug.md`
- **Copy-paste TMDb fetch logic:** Always use shared helpers `fetchMovieDetails()` / `fetchTVDetails()`. Never duplicate. See: `.lovable/memory/issues/02-duplicate-tmdb-fetch.md`
- **Files >200 lines:** Split at natural boundaries early. See: `.lovable/memory/issues/03-large-files.md`
- **Nested if statements:** Zero nesting rule. Max 2 conditions per if. No else after return. Use early returns and guard clauses.
- **Magic strings:** Use constants/enums. Never hardcode string literals for repeated values.
- **fmt.Errorf:** Use `apperror.Wrap()` instead.
- **Named file `<package>/<package>.go`:** Never name a file same as its package (e.g., `db/db.go`). Use descriptive names like `db/open.go`.
