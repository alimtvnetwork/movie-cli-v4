# Runtime Error Handling — TMDb, Database, Network

**Version:** 1.0.0  
**Updated:** 2026-04-16  
**Format:** Error scenarios with handling strategy and GIVEN/WHEN/THEN criteria

---

## Purpose

Define how the CLI handles runtime errors from external dependencies (TMDb API, SQLite database, filesystem, network). This spec ensures graceful degradation, clear user messaging, and consistent error recovery across all commands.

---

## 1. TMDb API Errors

### 1.1 Rate Limiting (HTTP 429)

TMDb enforces rate limits (~40 requests per 10 seconds for API v3).

**Current behavior:** No retry logic. The HTTP client returns the error and the command prints it.

**Required behavior:**

| Scenario | Strategy |
|----------|----------|
| Single 429 response | Retry after `Retry-After` header value (or 2s default) |
| 3 consecutive 429s | Abort with clear message: "TMDb rate limit exceeded — try again in X seconds" |
| Batch operations (scan) | Add 250ms delay between requests to stay under limits |

**Implementation pattern:**

```go
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
    maxRetries := 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := c.httpClient.Do(req)
        if err != nil {
            return nil, err
        }
        if resp.StatusCode != 429 {
            return resp, nil
        }
        resp.Body.Close()

        retryAfter := resp.Header.Get("Retry-After")
        delay := 2 * time.Second
        if secs, err := strconv.Atoi(retryAfter); err == nil {
            delay = time.Duration(secs) * time.Second
        }
        time.Sleep(delay)
    }
    return nil, fmt.Errorf("TMDb rate limit exceeded after %d retries", maxRetries)
}
```

**Acceptance Criteria:**

- GIVEN a 429 response from TMDb WHEN a request is made THEN the client retries after `Retry-After` seconds
- GIVEN 3 consecutive 429 responses WHEN retries are exhausted THEN a clear error message is shown
- GIVEN a scan of 50 files WHEN TMDb requests are made THEN a 250ms delay is inserted between requests

---

### 1.2 Authentication Errors (HTTP 401)

| Scenario | Message |
|----------|---------|
| Invalid API key | `❌ TMDb API key is invalid. Run: movie config set tmdb_api_key YOUR_KEY` |
| Missing API key | `❌ No TMDb API key configured. Run: movie config set tmdb_api_key YOUR_KEY` |

**Acceptance Criteria:**

- GIVEN an invalid API key WHEN any TMDb request returns 401 THEN the error message includes the fix command
- GIVEN no API key configured WHEN a TMDb-dependent command runs THEN the error is shown before any network request

---

### 1.3 Server Errors (HTTP 5xx)

| Scenario | Strategy |
|----------|----------|
| 500 Internal Server Error | Retry once after 3s, then fail |
| 502/503 Bad Gateway | Retry once after 5s, then fail |
| 504 Gateway Timeout | Retry once after 5s, then fail |

**Message:** `⚠️ TMDb is temporarily unavailable. Try again later.`

---

### 1.4 Network Timeout

The HTTP client has a 15-second timeout (`tmdb/client.go:32`).

| Scenario | Strategy |
|----------|----------|
| Timeout on search/details | Show: `⚠️ TMDb request timed out. Check your internet connection.` |
| Timeout on poster download | Skip poster, continue with metadata: `⚠️ Poster download timed out — skipping` |

**Acceptance Criteria:**

- GIVEN no internet connection WHEN `movie search` runs THEN a network error message is shown (not a panic)
- GIVEN a poster download times out WHEN `movie scan` runs THEN the scan continues without the poster

---

## 2. SQLite Database Errors

### 2.1 Database Locked (SQLITE_BUSY)

SQLite allows only one writer at a time. WAL mode helps but doesn't eliminate contention.

| Scenario | Strategy |
|----------|----------|
| Write blocked by another connection | Retry with exponential backoff: 100ms, 200ms, 400ms, 800ms, 1.6s |
| 5 retries exhausted | Fail with: `❌ Database is busy — another process may be using it. Try again.` |

**Implementation pattern:**

```go
// In db.Open() — set busy_timeout pragma
db.Exec("PRAGMA busy_timeout = 5000")  // 5 second busy timeout
```

SQLite's built-in `busy_timeout` pragma handles most contention automatically. The 5-second timeout covers typical concurrent access scenarios.

**Acceptance Criteria:**

- GIVEN two CLI instances running simultaneously WHEN both write to the DB THEN the second waits up to 5 seconds
- GIVEN the DB is locked for >5 seconds WHEN a write is attempted THEN a clear error message is shown

---

### 2.2 Database Corruption

| Scenario | Strategy |
|----------|----------|
| `SQLITE_CORRUPT` error | Show: `❌ Database appears corrupted. Run: movie db repair` (future command) |
| Missing database file | Auto-create with migrations (current behavior in `db.Open()`) |
| Read-only filesystem | Show: `❌ Cannot write to database — check file permissions` |

---

### 2.3 Migration Failures

| Scenario | Strategy |
|----------|----------|
| New migration fails | Roll back the transaction, show the error, continue with old schema |
| Schema version mismatch | Log warning, attempt migration, fail gracefully if incompatible |

**Acceptance Criteria:**

- GIVEN a new version with schema changes WHEN migration fails THEN the CLI continues with the existing schema and warns the user

---

## 3. Filesystem Errors

### 3.1 File Not Found

| Command | Scenario | Message |
|---------|----------|---------|
| `movie play <id>` | `current_file_path` doesn't exist | `❌ File not found: /path/to/file.mkv` |
| `movie undo` | Source file at `to_path` is missing | `❌ Cannot undo — file no longer exists at: /path/to/file.mkv` |
| `movie scan <dir>` | Directory doesn't exist | `❌ Directory not found: /path/to/dir` |

---

### 3.2 Permission Denied

| Scenario | Message |
|----------|---------|
| Cannot read scan directory | `❌ Permission denied: /path/to/dir` |
| Cannot write to destination | `❌ Cannot write to destination — check permissions: /path/to/dir` |
| Cannot create thumbnails dir | `⚠️ Cannot create thumbnail dir — skipping poster download` |

---

### 3.3 Disk Full

| Scenario | Strategy |
|----------|----------|
| Copy fails mid-transfer | Delete partial file, keep source intact, show: `❌ Disk full — move aborted, source file preserved` |
| Poster download fails | Skip poster, continue scan |
| DB write fails | Show error, suggest freeing disk space |

**Acceptance Criteria:**

- GIVEN a cross-device move WHEN the copy fails mid-transfer THEN the source file is NOT deleted and the partial destination is cleaned up

---

## 4. Offline Mode / Graceful Degradation

When the network is unavailable, commands should degrade gracefully:

| Command | Online Behavior | Offline Behavior |
|---------|----------------|-------------------|
| `movie scan` | Fetch TMDb metadata + poster | Scan files, insert with cleaned filename only, skip metadata |
| `movie search` | Search TMDb | `❌ Network required for TMDb search` |
| `movie info <title>` | DB lookup → TMDb fallback | DB lookup only, skip TMDb fallback |
| `movie suggest` | TMDb recommendations + trending | `❌ Network required for suggestions` |
| `movie ls` | List from DB | Works fully offline ✅ |
| `movie move` | Works locally | Works fully offline ✅ |
| `movie rename` | Works locally | Works fully offline ✅ |
| `movie undo` | Works locally | Works fully offline ✅ |
| `movie play` | Opens local file | Works fully offline ✅ |
| `movie stats` | Reads from DB | Works fully offline ✅ |
| `movie tag` | Reads/writes DB | Works fully offline ✅ |
| `movie export` | Reads from DB | Works fully offline ✅ |
| `movie config` | Reads/writes DB | Works fully offline ✅ |
| `movie update` | `git pull` + rebuild | `❌ Network required for update` |

**Detection pattern:**

```go
func isNetworkError(err error) bool {
    var netErr net.Error
    if errors.As(err, &netErr) {
        return true
    }
    var dnsErr *net.DNSError
    return errors.As(err, &dnsErr)
}
```

**Acceptance Criteria:**

- GIVEN no internet WHEN `movie scan ~/dir` runs THEN files are scanned and inserted with local data only (no metadata)
- GIVEN no internet WHEN `movie ls` runs THEN the library is displayed normally
- GIVEN no internet WHEN `movie search` runs THEN a clear "network required" message is shown

---

## 5. Error Message Standards

All error messages follow these rules:

| Rule | Example |
|------|---------|
| Start with emoji indicator | `❌` error, `⚠️` warning, `📭` empty |
| Include the fix when possible | `❌ No API key. Run: movie config set tmdb_api_key YOUR_KEY` |
| Never expose stack traces to users | Log to debug, show human message |
| Print to stderr for errors | `fmt.Fprintf(os.Stderr, ...)` |
| Print to stdout for warnings | `fmt.Printf("⚠️ ...")` |

---

## Cross-References

- [Error Management Overview](./00-overview.md)
- [Error Architecture](./02-error-architecture/)
- [Error Code Registry](./03-error-code-registry/)
- [Acceptance Criteria](./97-acceptance-criteria.md)
- [TMDb Client](../../tmdb/client.go) — HTTP timeout, request logic
- [DB Layer](../../db/db.go) — Open(), migrations, pragmas

---

*Runtime error handling spec — updated: 2026-04-10*
