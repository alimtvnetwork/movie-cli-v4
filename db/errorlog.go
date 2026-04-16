// errorlog.go — ErrorLog table helpers.
package db

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"fmt"
)

// ErrorLogEntry holds all fields for an error log row.
type ErrorLogEntry struct {
	Timestamp  string
	Level      string
	Source     string
	Function   string
	Command    string
	WorkDir    string
	Message    string
	StackTrace string
}

// InsertErrorLog writes an error entry to the ErrorLog table.
func (d *DB) InsertErrorLog(entry ErrorLogEntry) error {
	_, err := d.Exec(`
		INSERT INTO ErrorLog (Timestamp, Level, Source, Function, Command, WorkDir, Message, StackTrace)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp, entry.Level, entry.Source, entry.Function,
		entry.Command, entry.WorkDir, entry.Message, entry.StackTrace,
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
