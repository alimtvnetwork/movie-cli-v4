// scan_stats.go — ScanStats struct shared by scan report functions.
package cmd

import "github.com/alimtvnetwork/movie-cli-v4/db"

// ScanStats holds aggregate counts for scan output functions.
type ScanStats struct {
	ScanDir   string
	OutputDir string
	Items     []db.Media
	Total     int
	Movies    int
	TV        int
	Skipped   int
	Removed   int
}
