// types.go — shared option structs for functions with >3 parameters.
package cmd

import (
	"bufio"
	"net/http"
	"os"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
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
	Client  *tmdb.Client
	ScanDir string
	BatchID string
	UseJSON bool
	UseTable bool
	HasTMDb bool
}

// ScanOutputOpts groups output format flags used during scan processing.
type ScanOutputOpts struct {
	UseTable bool
	UseJSON  bool
}

// DryRunCounters groups counter pointers for dry-run scan output.
type DryRunCounters struct {
	TotalFiles *int
	MovieCount *int
	TVCount    *int
}

// WatchState groups mutable state for watch-mode polling cycles.
type WatchState struct {
	Client  *tmdb.Client
	HasTMDb bool
	Seen    map[string]bool
}

// SuggestTypeInput groups parameters for type-based suggestion generation.
type SuggestTypeInput struct {
	Database  *db.DB
	Client    *tmdb.Client
	MediaType string
	Count     int
}

// BatchMovePreview groups parameters for batch move preview generation.
type BatchMovePreview struct {
	Files     []os.FileInfo
	SourceDir string
	MoviesDir string
	TVDir     string
}

// TrackMoveInput groups parameters for recording a file move operation.
type TrackMoveInput struct {
	Database  *db.DB
	Result    cleaner.Result
	FileInfo  os.FileInfo
	SrcPath   string
	DestPath  string
	CleanName string
}

// FindMoveMediaInput groups parameters for finding or creating media during moves.
type FindMoveMediaInput struct {
	Database *db.DB
	Result   cleaner.Result
	FileInfo os.FileInfo
	SrcPath  string
	DestPath string
}

// WalkEntryInput groups parameters for processing a single walk entry during popout discovery.
type WalkEntryInput struct {
	RootDir  string
	Path     string
	Info     os.FileInfo
	MaxDepth int
	Items    *[]popoutItem
}

// FolderRemoveInput groups parameters for folder removal operations.
type FolderRemoveInput struct {
	Database *db.DB
	DirPath  string
	DirName  string
	BatchID  string
}

// MediaRequest groups database context for REST media handlers.
type MediaRequest struct {
	Database *db.DB
	ID       int64
}

// MediaPatchRequest groups parameters for media PATCH REST handlers.
type MediaPatchRequest struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	Database *db.DB
	ID       int64
}

// MediaUpdateField groups parameters for a single media field update.
type MediaUpdateField struct {
	Database *db.DB
	ID       int64
	Key      string
	Val      interface{}
}

// UniqueFilter groups parameters for deduplicating search results.
type UniqueFilter struct {
	ExistingIDs map[int]bool
	Count       int
}

// RecursiveFileContext groups parameters for handling a file during recursive directory walks.
type RecursiveFileContext struct {
	Entry   os.DirEntry
	Path    string
	ScanDir string
	Opts    RecursiveWalkOpts
	Files   *[]videoFile
}

// TrackScanResult groups the result of scanning a single file for action tracking.
type TrackScanResult struct {
	Media     *db.Media
	FullPath  string
	MediaID   int64
	InsertErr error
}

// DiscoverGenreInput groups parameters for genre-based discovery in suggestions.
type DiscoverGenreInput struct {
	Sorted    []genreCount
	MediaType string
	TypeName  string
}

// FillRecoInput groups parameters for recommendation-based suggestion filling.
type FillRecoInput struct {
	Database  *db.DB
	MediaType string
}
