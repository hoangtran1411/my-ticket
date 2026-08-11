---
name: sqlite-migrations
description: SQLite performance tuning, Write-Ahead Logging (WAL) configuration, and schema migration strategies.
---

# SQLite Optimization & Migration Skill

This skill provides configuration standards for high-concurrency embedded SQLite in Go.

## High Performance SQLite Rules

1. **Pragma Optimization**:
   ```sql
   PRAGMA journal_mode = WAL;
   PRAGMA synchronous = NORMAL;
   PRAGMA foreign_keys = ON;
   PRAGMA busy_timeout = 5000;
   ```

2. **Connection Pool Management**:
   - `SetMaxOpenConns(10)`: Allows parallel readers in WAL mode.
   - `SetMaxIdleConns(5)`: Retains warm connections.
   - `SetConnMaxLifetime(1 * time.Hour)`: Prevents stale file handles.

3. **CGO-Free Engine**:
   - Use `modernc.org/sqlite` for cross-platform pure Go builds without external C compilers.
