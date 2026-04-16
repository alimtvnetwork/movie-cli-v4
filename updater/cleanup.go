package updater

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Cleanup removes leftover temp binaries and backup files from previous updates.
func Cleanup() (int, error) {
	cleaned := 0

	selfPath, err := os.Executable()
	if err != nil {
		return 0, apperror.Wrap("cannot determine executable path", err)
	}
	selfPath, _ = filepath.EvalSymlinks(selfPath)

	// Clean handoff copies from binary directory
	cleaned += cleanGlob(filepath.Join(filepath.Dir(selfPath), "movie-update-*"), selfPath)

	// Clean handoff copies from temp directory
	cleaned += cleanGlob(filepath.Join(os.TempDir(), "movie-update-*"), selfPath)

	// Clean .bak backup files from binary directory
	cleaned += cleanGlob(filepath.Join(filepath.Dir(selfPath), "*.bak"), selfPath)

	return cleaned, nil
}

// cleanGlob removes files matching a glob pattern, skipping the currently running binary.
func cleanGlob(pattern, selfPath string) int {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0
	}

	cleaned := 0
	for _, match := range matches {
		abs, _ := filepath.Abs(match)
		if strings.EqualFold(abs, selfPath) {
			continue
		}
		// Skip directories
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		if err := os.Remove(match); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ Could not remove %s: %v\n", filepath.Base(match), err)
			continue
		}
		fmt.Printf("  Removed: %s\n", filepath.Base(match))
		cleaned++
	}
	return cleaned
}
