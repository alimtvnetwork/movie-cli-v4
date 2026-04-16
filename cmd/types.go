// types.go — shared option structs for functions with >3 parameters.
package cmd

import (
	"bufio"
	"os"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

// MoveContext groups parameters for batch and interactive move flows.
type MoveContext struct {
	Database  *db.DB
	Scanner   *bufio.Scanner
	SourceDir string
	Files     []os.FileInfo
	Home      string
}

// CleanupContext groups parameters for popout folder cleanup operations.
type CleanupContext struct {
	Scanner  *bufio.Scanner
	Database *db.DB
	BatchID  string
}

// ScanServiceConfig groups parameters for post-scan services (REST, watch).
type ScanServiceConfig struct {
	ScanDir   string
	OutputDir string
	Database  *db.DB
	Creds     tmdbCredentials
}

// SuggestCollector groups parameters for suggestion collection helpers.
type SuggestCollector struct {
	Client      *tmdb.Client
	ExistingIDs map[int]bool
	Count       int
}

// StatsCounts groups the three media count values for stats rendering.
type StatsCounts struct {
	Movies int
	TV     int
	Total  int
}

// LsPage groups pagination parameters for list display.
type LsPage struct {
	Offset   int
	PageSize int
	Total    int
}

// RecursiveWalkOpts groups depth-control parameters for recursive directory walks.
type RecursiveWalkOpts struct {
	BaseParts int
	MaxDepth  int
}

// ThumbnailInput groups parameters for poster/thumbnail download functions.
type ThumbnailInput struct {
	Client     *tmdb.Client
	Database   *db.DB
	Media      *db.Media
	PosterPath string
	OutputDir  string
}

// HistoryLogInput groups parameters for saving move history to JSON log.
type HistoryLogInput struct {
	BasePath string
	Title    string
	Year     int
	FromPath string
	ToPath   string
}

// ScanLoopConfig groups parameters for the main scan processing loop.
type ScanLoopConfig struct {
	Client      *tmdb.Client
	ScanDir     string
	BatchID     string
	UseJSON     bool
	UseTable    bool
	HasTMDb     bool
}

// ScanOutputOpts groups output format flags used during scan processing.
type ScanOutputOpts struct {
	UseTable bool
	UseJSON  bool
}
