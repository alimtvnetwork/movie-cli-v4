// movie_scan_collect.go — video file discovery for movie scan
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
)

// videoFile holds a discovered video file's display name and full path.
type videoFile struct {
	Name     string // display name used for cleaning (dir name or filename)
	FullPath string // absolute path to the actual video file
}

// collectVideoFiles finds video files in the given directory.
func collectVideoFiles(scanDir string, recursive bool, maxDepth int) []videoFile {
	if recursive {
		return collectRecursive(scanDir, maxDepth)
	}
	return collectTopLevel(scanDir)
}

func collectRecursive(scanDir string, maxDepth int) []videoFile {
	var files []videoFile
	scanDir = filepath.Clean(scanDir)
	baseParts := len(splitPath(scanDir))

	_ = filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠️  Cannot access %s: %v\n", path, err)
			return nil
		}
		if d.IsDir() {
			return handleRecursiveDir(d, path, baseParts, maxDepth)
		}
		return handleRecursiveFile(d, path, scanDir, baseParts, maxDepth, &files)
	})
	return files
}

func handleRecursiveDir(d os.DirEntry, path string, opts RecursiveWalkOpts) error {
	base := d.Name()
	if base == ".movie-output" || (strings.HasPrefix(base, ".") && base != ".") {
		return filepath.SkipDir
	}
	if opts.MaxDepth > 0 {
		dirParts := len(splitPath(filepath.Clean(path)))
		if dirParts-opts.BaseParts > opts.MaxDepth {
			return filepath.SkipDir
		}
	}
	return nil
}

func handleRecursiveFile(d os.DirEntry, path, scanDir string, opts RecursiveWalkOpts, files *[]videoFile) error {
	if opts.MaxDepth > 0 {
		fileParts := len(splitPath(filepath.Clean(filepath.Dir(path))))
		if fileParts-opts.BaseParts > opts.MaxDepth {
			return nil
		}
	}
	if !cleaner.IsVideoFile(d.Name()) {
		return nil
	}
	parentDir := filepath.Dir(path)
	name := d.Name()
	if parentDir != scanDir {
		name = filepath.Base(parentDir)
	}
	*files = append(*files, videoFile{Name: name, FullPath: path})
	return nil
}

func collectTopLevel(scanDir string) []videoFile {
	var files []videoFile
	entries, readErr := os.ReadDir(scanDir)
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot read folder: %v\n", readErr)
		return nil
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(scanDir, name)

		if entry.IsDir() {
			if vf, ok := findVideoInSubdir(name, fullPath); ok {
				files = append(files, vf)
			}
			continue
		}
		if cleaner.IsVideoFile(name) {
			files = append(files, videoFile{Name: name, FullPath: fullPath})
		}
	}
	return files
}

func findVideoInSubdir(dirName, dirPath string) (videoFile, bool) {
	subEntries, subErr := os.ReadDir(dirPath)
	if subErr != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  Cannot read subdirectory %s: %v\n", dirName, subErr)
		return videoFile{}, false
	}
	for _, sub := range subEntries {
		if !sub.IsDir() && cleaner.IsVideoFile(sub.Name()) {
			return videoFile{
				Name:     dirName,
				FullPath: filepath.Join(dirPath, sub.Name()),
			}, true
		}
	}
	return videoFile{}, false
}

// splitPath splits a filepath into its components.
func splitPath(p string) []string {
	var parts []string
	for p != "" && p != "." && p != "/" && p != string(filepath.Separator) {
		dir, file := filepath.Split(p)
		if file != "" {
			parts = append(parts, file)
		}
		p = filepath.Clean(dir)
		if p == dir {
			break
		}
	}
	return parts
}
