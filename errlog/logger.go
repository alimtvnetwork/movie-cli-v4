// Package errlog provides centralized error logging to both file and database.
// All errors are written to .movie-output/logs/error.txt and to the error_logs
// table in the database. Each entry includes a timestamp, severity, source
// location, message, and optional stack trace.
package errlog

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents the severity of a log entry.
type Level string

const (
	LevelError Level = "ERROR"
	LevelWarn  Level = "WARN"
	LevelInfo  Level = "INFO"
)

// Entry holds a single error log record.
type Entry struct {
	Timestamp  string
	Level      Level
	Source     string // file:line
	Function   string
	Message    string
	StackTrace string
	Command    string // which CLI command was running
	WorkDir    string // CWD when the error occurred
}

// Logger writes errors to a log file and optionally to the database.
type Logger struct {
	mu       sync.Mutex
	filePath string
	file     *os.File
	dbLog    func(Entry) // optional DB writer, set after DB opens
	command  string
	workDir  string
}

// global singleton
var (
	global   *Logger
	globalMu sync.Mutex
)

// Init creates the global logger. Call once at startup before any logging.
// outputDir is the .movie-output directory (or data dir). command is the
// CLI subcommand being run (e.g. "scan", "rest").
func Init(outputDir, command string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	logDir := filepath.Join(outputDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return apperror.Wrap("cannot create log dir %s", logDir, err)
	}

	logPath := filepath.Join(logDir, "error.txt")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return apperror.Wrap("cannot open log file %s", logPath, err)
	}

	wd, _ := os.Getwd()

	global = &Logger{
		filePath: logPath,
		file:     f,
		command:  command,
		workDir:  wd,
	}
	return nil
}

// SetDBWriter sets the optional database writer function.
// This is called after the DB is opened so errlog doesn't import db.
func SetDBWriter(fn func(Entry)) {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global != nil {
		global.dbLog = fn
	}
}

// Close flushes and closes the log file.
func Close() {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global != nil && global.file != nil {
		global.file.Close()
		global.file = nil
	}
}

// FilePath returns the path to the error log file, or empty if not initialized.
func FilePath() string {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global != nil {
		return global.filePath
	}
	return ""
}

// Error logs an error-level message with stack trace.
func Error(msg string, args ...interface{}) {
	log(LevelError, fmt.Sprintf(msg, args...), true)
}

// Warn logs a warning-level message without stack trace.
func Warn(msg string, args ...interface{}) {
	log(LevelWarn, fmt.Sprintf(msg, args...), false)
}

// Info logs an info-level message without stack trace.
func Info(msg string, args ...interface{}) {
	log(LevelInfo, fmt.Sprintf(msg, args...), false)
}

// ErrorWithSource logs an error with explicit source info (for wrapping).
func ErrorWithSource(source, msg string) {
	entry := Entry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     LevelError,
		Source:    source,
		Message:   msg,
	}
	writeEntry(entry)
}

func log(level Level, msg string, includeStack bool) {
	// Capture caller info (skip 2: log -> Error/Warn/Info -> caller)
	source, fn := callerInfo(3)

	entry := Entry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Source:    source,
		Function:  fn,
		Message:   msg,
	}

	if includeStack {
		entry.StackTrace = captureStack(3)
	}

	globalMu.Lock()
	if global != nil {
		entry.Command = global.command
		entry.WorkDir = global.workDir
	}
	globalMu.Unlock()

	writeEntry(entry)
}

func writeEntry(entry Entry) {
	// Always print to stderr
	if entry.Level == LevelError {
		fmt.Fprintf(os.Stderr, "❌ [%s] %s: %s\n", entry.Level, entry.Source, entry.Message)
	} else if entry.Level == LevelWarn {
		fmt.Fprintf(os.Stderr, "⚠️  [%s] %s: %s\n", entry.Level, entry.Source, entry.Message)
	}

	globalMu.Lock()
	logger := global
	globalMu.Unlock()

	if logger == nil {
		return
	}

	// Write to file
	logger.mu.Lock()
	if logger.file != nil {
		var sb strings.Builder
		sb.WriteString("────────────────────────────────────────\n")
		sb.WriteString(fmt.Sprintf("Time:      %s\n", entry.Timestamp))
		sb.WriteString(fmt.Sprintf("Level:     %s\n", entry.Level))
		sb.WriteString(fmt.Sprintf("Source:    %s\n", entry.Source))
		if entry.Function != "" {
			sb.WriteString(fmt.Sprintf("Function:  %s\n", entry.Function))
		}
		if entry.Command != "" {
			sb.WriteString(fmt.Sprintf("Command:   %s\n", entry.Command))
		}
		if entry.WorkDir != "" {
			sb.WriteString(fmt.Sprintf("WorkDir:   %s\n", entry.WorkDir))
		}
		sb.WriteString(fmt.Sprintf("Message:   %s\n", entry.Message))
		if entry.StackTrace != "" {
			sb.WriteString(fmt.Sprintf("Stack:\n%s\n", entry.StackTrace))
		}
		sb.WriteString("\n")
		logger.file.WriteString(sb.String())
	}
	logger.mu.Unlock()

	// Write to DB
	if logger.dbLog != nil {
		logger.dbLog(entry)
	}
}

func callerInfo(skip int) (source, fn string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown"
	}
	source = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	fnObj := runtime.FuncForPC(pc)
	if fnObj != nil {
		parts := strings.Split(fnObj.Name(), "/")
		fn = parts[len(parts)-1]
	}
	return
}

func captureStack(skip int) string {
	var sb strings.Builder
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fnObj := runtime.FuncForPC(pc)
		name := "unknown"
		if fnObj != nil {
			name = fnObj.Name()
		}
		sb.WriteString(fmt.Sprintf("  %s\n    %s:%d\n", name, filepath.Base(file), line))
		if i-skip > 15 {
			sb.WriteString("  ... (truncated)\n")
			break
		}
	}
	return sb.String()
}
