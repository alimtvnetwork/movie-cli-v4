// errorlog.go — error_logs table and DB writer for the error logging system.
package db

import "fmt"

// migrateErrorLogs creates the error_logs table if it doesn't exist.
func (d *DB) migrateErrorLogs() error {
	_, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS error_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp   TEXT NOT NULL,
			level       TEXT NOT NULL CHECK(level IN ('ERROR', 'WARN', 'INFO')),
			source      TEXT NOT NULL,
			function    TEXT DEFAULT '',
			command     TEXT DEFAULT '',
			work_dir    TEXT DEFAULT '',
			message     TEXT NOT NULL,
			stack_trace TEXT DEFAULT '',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_error_logs_level ON error_logs(level);
		CREATE INDEX IF NOT EXISTS idx_error_logs_ts    ON error_logs(timestamp);
	`)
	return err
}

// InsertErrorLog writes an error entry to the error_logs table.
func (d *DB) InsertErrorLog(timestamp, level, source, function, command, workDir, message, stackTrace string) error {
	_, err := d.Exec(`
		INSERT INTO error_logs (timestamp, level, source, function, command, work_dir, message, stack_trace)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		timestamp, level, source, function, command, workDir, message, stackTrace,
	)
	if err != nil {
		return fmt.Errorf("insert error log: %w", err)
	}
	return nil
}

// RecentErrorLogs returns the most recent N error log entries.
func (d *DB) RecentErrorLogs(limit int) ([]map[string]string, error) {
	rows, err := d.Query(`
		SELECT id, timestamp, level, source, function, command, work_dir, message, stack_trace
		FROM error_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query error logs: %w", err)
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var id int
		var ts, lvl, src, fn, cmd, wd, msg, st string
		if scanErr := rows.Scan(&id, &ts, &lvl, &src, &fn, &cmd, &wd, &msg, &st); scanErr != nil {
			return nil, fmt.Errorf("scan error log: %w", scanErr)
		}
		results = append(results, map[string]string{
			"id":          fmt.Sprintf("%d", id),
			"timestamp":   ts,
			"level":       lvl,
			"source":      src,
			"function":    fn,
			"command":     cmd,
			"work_dir":    wd,
			"message":     msg,
			"stack_trace": st,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return results, nil
}
