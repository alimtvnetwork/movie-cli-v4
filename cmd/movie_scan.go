// movie_scan.go — movie scan [folder] — command definition and orchestrator
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/tmdb"
)

var scanRecursive bool
var scanDepth int
var scanDryRun bool
var scanFormat string
var scanRest bool
var scanRestPort int

var movieScanCmd = &cobra.Command{
	Use:   "scan [folder]",
	Short: "Scan a folder for movies and TV shows",
	Long: `Scans a folder for video files, cleans filenames, fetches metadata
from TMDb, downloads thumbnails, and stores everything in the database.

If no folder is specified, scans the current working directory.
Use --recursive (-r) to scan all subdirectories recursively.
Use --depth to limit how many levels deep the recursive scan goes.
Use --dry-run to preview what would be scanned without writing anything.

Examples:
  movie scan                      Scan current directory (top-level)
  movie scan ~/Movies             Scan specific folder
  movie scan -r                   Scan current directory recursively
  movie scan --dry-run            Preview files without writing to DB
  movie scan --format table       Show results as a formatted table
  movie scan --rest               Scan and start REST server + open browser`,
	Args: cobra.MaximumNArgs(1),
	Run:  runMovieScan,
}

func init() {
	movieScanCmd.Flags().BoolVarP(&scanRecursive, "recursive", "r", false,
		"scan all subdirectories recursively")
	movieScanCmd.Flags().IntVarP(&scanDepth, "depth", "d", 0,
		"max subdirectory depth for recursive scan (0 = unlimited)")
	movieScanCmd.Flags().BoolVar(&scanDryRun, "dry-run", false,
		"preview what would be scanned without writing to DB or .movie-output")
	movieScanCmd.Flags().StringVar(&scanFormat, "format", "default",
		"output format: default, table, or json")
	movieScanCmd.Flags().BoolVar(&scanRest, "rest", false,
		"start REST server and open HTML report in browser after scan")
	movieScanCmd.Flags().IntVar(&scanRestPort, "port", 8086,
		"port for REST server when using --rest")
	movieScanCmd.Flags().BoolVarP(&scanWatch, "watch", "w", false,
		"watch for new files after initial scan")
	movieScanCmd.Flags().IntVar(&scanWatchInterval, "interval", 10,
		"polling interval in seconds for --watch mode")
}

func runMovieScan(cmd *cobra.Command, args []string) {
	useJSON := scanFormat == "json"

	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Database error: %v\n", err)
		return
	}
	defer database.Close()

	scanDir, err := resolveScanDir(args, useJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return
	}

	creds := resolveScanTMDbCredentials(database)
	outputDir := filepath.Join(scanDir, ".movie-output")

	if !scanDryRun {
		if err := createOutputDirs(outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			return
		}
	}

	initScanLogger(database, outputDir)

	if !useJSON {
		printScanHeader(scanDir, outputDir)
	}

	var removed int
	var jsonItems []scanJSONItem

	videoFiles := collectVideoFiles(scanDir, scanRecursive, scanDepth)
	useTable := scanFormat == string(db.OutputFormatTable)
	useTMDb := creds.HasAuth()

	tmdbClient := tmdb.NewClientWithToken(creds.APIKey, creds.Token)
	ctx := &ScanContext{
		Database:  database,
		Client:    tmdbClient,
		HasTMDb:   useTMDb,
		OutputDir: outputDir,
		UseTable:  useTable || useJSON,
		BatchID:   generateBatchID(),
	}

	if useTable {
		printScanTableHeader()
	}

	if scanDryRun {
		runDryRunScan(videoFiles, useJSON, useTable,
			&jsonItems, &ctx.TotalFiles, &ctx.MovieCount, &ctx.TVCount)
	}
	if !scanDryRun {
		removed = runMainScanLoop(ctx, videoFiles, tmdbClient, scanDir, ctx.BatchID, useJSON, useTable, useTMDb, &jsonItems)
	}

	if useTable {
		printScanTableFooter()
	}

	registerScanHistory(database, scanDir, ctx)

	if useJSON {
		printScanJSON(scanDir, jsonItems, ScanStats{
			Total: ctx.TotalFiles, Movies: ctx.MovieCount, TV: ctx.TVCount, Skipped: ctx.Skipped,
		})
	}
	if !useJSON {
		printScanFooter(ScanStats{
			ScanDir: scanDir, OutputDir: outputDir, Items: ctx.ScannedItems,
			Total: ctx.TotalFiles, Movies: ctx.MovieCount, TV: ctx.TVCount,
			Skipped: ctx.Skipped, Removed: removed,
		})
	}

	startPostScanServices(cmd, scanDir, outputDir, database, creds, tmdbClient)
}

func initScanLogger(database *db.DB, outputDir string) {
	if initErr := errlog.Init(outputDir, "scan"); initErr != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not init error logger: %v\n", initErr)
		return
	}
	errlog.SetDBWriter(func(e errlog.Entry) {
		dbErr := database.InsertErrorLog(db.ErrorLogEntry{
			Timestamp: e.Timestamp, Level: string(e.Level), Source: e.Source,
			Function: e.Function, Command: e.Command, WorkDir: e.WorkDir,
			Message: e.Message, StackTrace: e.StackTrace,
		})
		if dbErr != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Could not write error to DB: %v\n", dbErr)
		}
	})
}

func registerScanHistory(database *db.DB, scanDir string, ctx *ScanContext) {
	if scanDryRun {
		return
	}
	folderId, folderErr := database.UpsertScanFolder(scanDir)
	if folderErr != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not register scan folder: %v\n", folderErr)
		return
	}
	if histErr := database.InsertScanHistory(db.ScanHistoryInput{
		ScanFolderID: int(folderId), TotalFiles: ctx.TotalFiles,
		Movies: ctx.MovieCount, TV: ctx.TVCount,
	}); histErr != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not log scan history: %v\n", histErr)
	}
}

func startPostScanServices(cmd *cobra.Command, scanDir, outputDir string, database *db.DB, creds tmdbCredentials, client *tmdb.Client) {
	if scanDryRun {
		return
	}
	if scanRest {
		startRestWithOptionalWatch(cmd, scanDir, outputDir, database, creds)
		return
	}
	if scanWatch {
		runWatchLoop(scanDir, outputDir, database, creds)
	}
}

func startRestWithOptionalWatch(cmd *cobra.Command, scanDir, outputDir string, database *db.DB, creds tmdbCredentials) {
	restPort = scanRestPort
	fmt.Printf("\n🚀 Starting REST server on http://localhost:%d ...\n", restPort)
	go openBrowser(fmt.Sprintf("http://localhost:%d", restPort))
	if scanWatch {
		go runWatchLoop(scanDir, outputDir, database, creds)
	}
	runMovieRest(cmd, []string{})
}

// runDryRunScan handles the dry-run scanning loop for all output formats.
func runDryRunScan(videoFiles []videoFile, useJSON, useTable bool,
	jsonItems *[]scanJSONItem, totalFiles, movieCount, tvCount *int) {
	if useJSON {
		items, mc, tc := buildDryRunJSONItems(videoFiles)
		*jsonItems = items
		*totalFiles = len(items)
		*movieCount = mc
		*tvCount = tc
		return
	}
	if useTable {
		rows, mc, tc := buildDryRunTableRows(videoFiles)
		for _, row := range rows {
			printScanTableRow(row)
		}
		*totalFiles = len(rows)
		*movieCount = mc
		*tvCount = tc
		return
	}
	runDryRunPlainOutput(videoFiles, totalFiles, movieCount, tvCount)
}

// runDryRunPlainOutput prints dry-run results in plain text format.
func runDryRunPlainOutput(videoFiles []videoFile, totalFiles, movieCount, tvCount *int) {
	for _, vf := range videoFiles {
		*totalFiles++
		result := cleaner.Clean(vf.Name)
		typeIcon := db.TypeIcon(result.Type)
		fmt.Printf("\n  %d. %s %s", *totalFiles, typeIcon, result.CleanTitle)
		if result.Year > 0 {
			fmt.Printf(" (%d)", result.Year)
		}
		fmt.Printf(" [%s]\n", result.Type)
		fmt.Printf("     └─ %s\n", vf.Name)
		incrementTypeCountPtr(result.Type, movieCount, tvCount)
	}
}

// incrementTypeCountPtr bumps movie or tv count pointer based on media type.
func incrementTypeCountPtr(mediaType string, movieCount, tvCount *int) {
	if mediaType == string(db.MediaTypeMovie) {
		*movieCount++
		return
	}
	*tvCount++
}
