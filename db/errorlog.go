// errorlog.go — ErrorLog table helpers.
package db

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"fmt"
)

// InsertErrorLog writes an error entry to the ErrorLog table.
func (d *DB) InsertErrorLog(timestamp, level, source, function, command, workDir, message, stackTrace string) error {
	_, err := d.Exec(`
		INSERT INTO ErrorLog (Timestamp, Level, Source, Function, Command, WorkDir, Message, StackTrace)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		timestamp, level, source, function, command, workDir, message, stackTrace,
	)
	if err != nil {
		return apperror.Wrap("insert error log", err)
	}
	return nil
}

// RecentErrorLogs returns the most recent N error log entries.
func (d *DB) RecentErrorLogs(limit int) ([]map[string]string, error) {
	rows, err := d.Query(`
		SELECT ErrorLogId, Timestamp, Level, Source, Function, Command, WorkDir, Message, StackTrace
		FROM ErrorLog ORDER BY ErrorLogId DESC LIMIT ?`, limit)
	if err != nil {
		return nil, apperror.Wrap("query error logs", err)
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var id int
		var ts, lvl, src, fn, cmd, wd, msg, st string
		if scanErr := rows.Scan(&id, &ts, &lvl, &src, &fn, &cmd, &wd, &msg, &st); scanErr != nil {
			return nil, apperror.Wrap("scan error log", scanErr)
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
		return nil, apperror.Wrap("rows iteration", err)
	}
	return results, nil
}
