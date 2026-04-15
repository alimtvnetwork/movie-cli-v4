# Database Design — Suggestions & Proposals

**Version:** 1.0.0  
**Updated:** 2026-04-15  
**Status:** For Review  
**Scope:** Additional actions, alternative naming, missing relationships/tables

---

## 1. Additional Trackable Actions

### 1.1 Actions to Add to FileAction Table

The current FileAction lookup has 8 predefined types. The following additional actions are recommended based on CLI capabilities and future features:

| # | Action Name | Trigger | What to Store | Priority |
|---|------------|---------|---------------|----------|
| 1 | `TagAdd` | `movie tag add` | MediaId, tag name | High — tags are user data, should be undoable |
| 2 | `TagRemove` | `movie tag remove` | MediaId, tag name (+ snapshot) | High — undo restores the tag |
| 3 | `WatchlistAdd` | `movie watch add` | TmdbId, title | Medium — track watchlist changes |
| 4 | `WatchlistRemove` | `movie watch remove` | TmdbId, title (+ snapshot) | Medium — undo restores entry |
| 5 | `WatchlistStatusChange` | `movie watch mark` | WatchlistId, old status → new status | Medium — track watched transitions |
| 6 | `ConfigChange` | `movie config set` | Key, old value, new value | Medium — undo config changes |
| 7 | `MetadataRefresh` | `movie rescan --id` | MediaId, old snapshot | Low — already covered by `RescanUpdate` |
| 8 | `ThumbnailDownload` | During scan/rescan | MediaId, thumbnail path | Low — informational |
| 9 | `Duplicate` | `movie move` (duplicate detection) | MediaId, duplicate path | Low — log duplicate findings |
| 10 | `Archive` | Future `movie archive` | MediaId, archive path | Future — when archive command exists |

### 1.2 Recommended Additions (High + Medium)

Update the FileAction seed to include these 6 additional types:

```sql
INSERT INTO FileAction (Name) VALUES
    ('Move'),           -- 1
    ('Rename'),         -- 2
    ('Delete'),         -- 3
    ('Popout'),         -- 4
    ('Restore'),        -- 5
    ('ScanAdd'),        -- 6
    ('ScanRemove'),     -- 7
    ('RescanUpdate'),   -- 8
    ('TagAdd'),         -- 9
    ('TagRemove'),      -- 10
    ('WatchlistAdd'),   -- 11
    ('WatchlistRemove'),-- 12
    ('WatchlistStatusChange'), -- 13
    ('ConfigChange');   -- 14
```

> This brings the total to **14 action types** — covering all current commands that modify state.

---

## 2. Alternative Names for the `Media` Table

The current name `Media` is acceptable. Here are alternatives with trade-offs:

| Name | Pros | Cons | Recommendation |
|------|------|------|----------------|
| **`Media`** | Generic, covers movies + TV, familiar | Slightly abstract — could mean any media type | ✅ **Keep — current choice** |
| **`Title`** | Natural — "a title in the library" | Conflicts with the `Title` column name; ambiguous | ❌ Avoid |
| **`LibraryItem`** | Explicit — "an item in the user's library" | Verbose, 11 chars, long FK names (`LibraryItemId`) | ⚠️ Acceptable but wordy |
| **`MediaFile`** | Clarifies it's a file-backed entry | Loses meaning if file is deleted but record kept | ⚠️ Acceptable |
| **`Catalog`** | Clean, implies organized collection | Less intuitive as a single-item entity | ❌ Avoid |
| **`Entry`** | Short, generic | Too generic — meaningless without context | ❌ Avoid |

### Recommendation

**Keep `Media`.** It's concise, well-understood, and matches TMDb API terminology. The table stores metadata about media items, and all current and future code already references "media" consistently. Renaming would require touching every file with no functional benefit.

---

## 3. Missing Relationships & Tables

### 3.1 Missing Tables — Recommended

| # | Table | Purpose | Relationship | Priority |
|---|-------|---------|-------------|----------|
| 1 | **`Collection`** | TMDb movie collections (e.g., "The Dark Knight Trilogy") | Media N→1 Collection | Medium |
| 2 | **`Director`** | Normalized director names (currently TEXT on Media) | Media N→M Director via `MediaDirector` | Medium |
| 3 | **`ProductionCompany`** | Studio/production company (available from TMDb) | Media N→M ProductionCompany via join | Low |
| 4 | **`Country`** | Production countries (available from TMDb) | Media N→M Country via join | Low |
| 5 | **`Keyword`** | TMDb keywords/tags (different from user tags) | Media N→M Keyword via join | Low |
| 6 | **`ExternalId`** | Store multiple external IDs per media (TMDb, IMDb, TVDB, etc.) | Media 1→N ExternalId | Low |
| 7 | **`Season`** / **`Episode`** | TV-specific: season/episode tracking | Media 1→N Season 1→N Episode | Future |

### 3.2 Collection Table — Detailed Proposal

TMDb groups movies into collections (e.g., "Harry Potter Collection"). This is valuable for the `movie suggest` and `movie ls` commands.

```sql
CREATE TABLE Collection (
    CollectionId     INTEGER PRIMARY KEY AUTOINCREMENT,
    TmdbCollectionId INTEGER UNIQUE,
    Name             TEXT NOT NULL,
    PosterPath       TEXT,
    CreatedAt        TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Add FK column to Media
ALTER TABLE Media ADD COLUMN CollectionId INTEGER
    REFERENCES Collection(CollectionId);

CREATE INDEX IdxMedia_CollectionId ON Media(CollectionId);
```

**View addition:**

```sql
CREATE VIEW VwCollectionMedia AS
SELECT
    c.CollectionId,
    c.Name AS CollectionName,
    m.MediaId,
    m.Title,
    m.Year
FROM Collection c
INNER JOIN Media m ON m.CollectionId = c.CollectionId
ORDER BY c.Name, m.Year;
```

### 3.3 Director Normalization — Detailed Proposal

Currently `Director` is a TEXT column on Media. For a small CLI, this is pragmatic. However, normalizing enables:
- "Show all movies by Christopher Nolan" queries via index
- Director dedup across movies
- TMDb person ID linking (same as Cast)

```sql
CREATE TABLE Director (
    DirectorId   INTEGER PRIMARY KEY AUTOINCREMENT,
    Name         TEXT NOT NULL,
    TmdbPersonId INTEGER UNIQUE
);

CREATE TABLE MediaDirector (
    MediaDirectorId INTEGER PRIMARY KEY AUTOINCREMENT,
    MediaId         INTEGER NOT NULL,
    DirectorId      INTEGER NOT NULL,
    UNIQUE (MediaId, DirectorId),
    FOREIGN KEY (MediaId) REFERENCES Media(MediaId) ON DELETE CASCADE,
    FOREIGN KEY (DirectorId) REFERENCES Director(DirectorId)
);
```

> **Trade-off:** This adds 2 tables and a view for a feature that's currently a simple text column. Recommend implementing only if "browse by director" becomes a feature.

### 3.4 Missing Relationships on Existing Tables

| From | To | Type | Status | Issue |
|------|----|------|--------|-------|
| ScanHistory → Media | ScanHistoryId FK on Media | 1-N | ✅ Added in v2.0.0 | Was missing — scan couldn't track which media it discovered |
| ScanFolder → ScanHistory | ScanFolderId FK on ScanHistory | 1-N | ✅ Added in v2.0.0 | ScanFolder is now root entity |
| MoveHistory → FileAction | FileActionId FK on MoveHistory | 1-N | ✅ Added in v2.0.0 | Was missing — action type was implicit |
| ActionHistory → FileAction | FileActionId FK on ActionHistory | 1-N | ✅ Added in v2.0.0 | Was missing — action type was text |
| Watchlist → Media | Cross-DB ref (no FK) | Optional | ✅ Already exists | Cross-DB per Split DB rules |
| Media → Media | Self-ref for remakes/related | N-M | ❌ Not needed | Over-engineering for a CLI |
| ErrorLog → Media | MediaId on ErrorLog | Optional | ❌ Not needed | Errors aren't always media-related |

---

## 4. Summary of Recommendations

### Immediate (v2.0.0) — Already Implemented

- [x] Genre normalized → Genre + MediaGenre (1-N)
- [x] Cast normalized → Cast + MediaCast (N-M)
- [x] Language normalized → Language table (1-N)
- [x] FileAction lookup table with 8 predefined types
- [x] ScanFolder as root entity
- [x] ScanHistory enriched with detailed counts
- [x] FileSizeMb in megabytes
- [x] 8 database views created during migration
- [x] All IDs use INTEGER AUTOINCREMENT
- [x] PascalCase naming throughout

### Short-term (v2.1.0) — Recommended

- [ ] Add 6 more FileAction types (TagAdd, TagRemove, WatchlistAdd, WatchlistRemove, WatchlistStatusChange, ConfigChange)
- [ ] Add Collection table for TMDb movie collections
- [ ] Integrate ActionHistory into `movie tag` commands

### Medium-term (v2.2.0+) — Optional

- [ ] Normalize Director into separate table with N-M join
- [ ] Add Season/Episode tables for TV show tracking
- [ ] Add ExternalId table for multiple ID sources

### Not Recommended

- Renaming `Media` table — no benefit, high churn
- Self-referencing Media table — over-engineering
- ProductionCompany / Country tables — low value for CLI use case
- ErrorLog → Media relationship — errors aren't always media-specific

---

*Suggestions & proposals — updated: 2026-04-15*
