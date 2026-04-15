---
name: No Split DB
description: All tables in single mahin.db — never use multiple .db files
type: constraint
---
Do NOT use the Split DB pattern (multiple .db files per domain). All tables live in a single `mahin.db` file. **Why:** The system is small enough that splitting adds unnecessary complexity. Watchlist.MediaId is now a proper FK to Media, not a cross-DB reference.
