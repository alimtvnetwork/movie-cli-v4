// movie_scan_helpers.go — shared helpers for movie scan (dir resolution, output dirs, print)
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/alimtvnetwork/movie-cli-v4/version"
)

// resolveScanDir determines and validates the scan directory from args.
func resolveScanDir(args []string, quiet bool) (string, error) {
	scanDir, err := scanDirFromArgs(args, quiet)
	if err != nil {
		return "", err
	}

	scanDir, err = expandTilde(scanDir)
	if err != nil {
		return "", err
	}

	return validateDirPath(scanDir)
}

func scanDirFromArgs(args []string, quiet bool) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine current directory: %v", err)
	}
	if !quiet {
		fmt.Printf("📂 No folder specified — scanning current directory\n\n")
	}
	return dir, nil
}

func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %v", err)
	}
	return filepath.Join(home, path[1:]), nil
}

func validateDirPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("folder not found: %s", path)
	}
	return path, nil
}

// createOutputDirs creates the .movie-output directory structure.
func createOutputDirs(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("cannot create output directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "json", string(db.MediaTypeMovie)), 0755); err != nil {
		return fmt.Errorf("cannot create json/%s dir: %v", db.MediaTypeMovie, err)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "json", string(db.MediaTypeTV)), 0755); err != nil {
		return fmt.Errorf("cannot create json/%s dir: %v", db.MediaTypeTV, err)
	}
	return nil
}

// printScanHeader prints the scan mode banner (gitmap-style box).
func printScanHeader(scanDir, outputDir string) {
	ver := version.Short()
	// Pad version to center it in the box (38 chars inner width)
	label := fmt.Sprintf("🎬  Movie CLI %s", ver)
	padTotal := 38 - len(label) + 2 // +2 for emoji width
	if padTotal < 0 {
		padTotal = 0
	}
	padLeft := padTotal / 2
	padRight := padTotal - padLeft
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Printf("  ║%s%s%s║\n", strings.Repeat(" ", padLeft), label, strings.Repeat(" ", padRight))
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  📂 Scanning: %s\n", scanDir)
	if scanDryRun {
		fmt.Println("  🧪 Mode: dry run (no writes)")
	}
	if scanRecursive && scanDepth > 0 {
		fmt.Printf("  🔄 Mode: recursive (max depth: %d)\n", scanDepth)
	}
	if scanRecursive && scanDepth <= 0 {
		fmt.Println("  🔄 Mode: recursive (all subdirectories)")
	}
	if !scanDryRun {
		fmt.Printf("  📁 Output: %s\n", outputDir)
	}
	fmt.Println()
	fmt.Println("  ■ Scanned Items")
	fmt.Println("  ──────────────────────────────────────────")
}

// printScanFooter prints the summary after scanning completes (gitmap-style).
func printScanFooter(scanDir, outputDir string, scannedItems []db.Media,
	totalFiles, movieCount, tvCount, skipped, removed int) {
	fmt.Println()
	fmt.Println("  ■ Summary")
	fmt.Println("  ──────────────────────────────────────────")
	label := "📊 Scan Complete!"
	if scanDryRun {
		label = "📊 Dry Run Complete!"
	}
	fmt.Println("  " + label)
	printScanCounts(totalFiles, movieCount, tvCount, skipped, removed)

	if scanDryRun {
		fmt.Println("\n  💡 Run without --dry-run to actually scan and save.")
		return
	}

	printScanOutputFiles(outputDir, scanDir, scannedItems, totalFiles, movieCount, tvCount, skipped)
	fmt.Println()
}
