// movie_popout_cleanup.go — folder cleanup phase for popout command.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

// offerFolderCleanup lists source subfolders and offers removal options.
func offerFolderCleanup(cc CleanupContext, rootDir string, items []popoutItem) {
	folders := collectPopoutFolders(rootDir, items)
	if len(folders) == 0 {
		return
	}

	printFolderSummary(folders)
	promptFolderAction(cc, folders)
}

func collectPopoutFolders(rootDir string, items []popoutItem) []popoutFolderInfo {
	subDirs := make(map[string]bool)
	for _, item := range items {
		subDirs[item.subDir] = true
	}

	var folders []popoutFolderInfo
	for dir := range subDirs {
		dirPath := filepath.Join(rootDir, dir)
		info, statErr := os.Stat(dirPath)
		if statErr != nil || !info.IsDir() {
			continue
		}
		folders = append(folders, scanFolderContents(dir, dirPath))
	}
	return folders
}

func scanFolderContents(name, dirPath string) popoutFolderInfo {
	var files []string
	var totalSize int64
	_ = filepath.Walk(dirPath, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dirPath, p)
		files = append(files, fmt.Sprintf("%s (%s)", rel, humanSize(fi.Size())))
		totalSize += fi.Size()
		return nil
	})
	return popoutFolderInfo{name: name, path: dirPath, files: files, totalSize: totalSize}
}

func printFolderSummary(folders []popoutFolderInfo) {
	fmt.Println("  📁 Source folders after popout:")
	fmt.Println()
	for i, f := range folders {
		if len(f.files) == 0 {
			fmt.Printf("  %d. %s/\n     └── (empty)\n", i+1, f.name)
			continue
		}
		fmt.Printf("  %d. %s/\n     └── %d files remaining (%s)\n",
			i+1, f.name, len(f.files), humanSize(f.totalSize))
	}
}

func promptFolderAction(cc CleanupContext, folders []popoutFolderInfo) {
	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println("    [a] Remove all listed folders")
	fmt.Println("    [s] Select folders to remove one by one")
	fmt.Println("    [n] Keep all folders")
	fmt.Println("    [l] List files in each folder before deciding")
	fmt.Print("\n  Choose [a/s/n/l]: ")

	if !cc.Scanner.Scan() {
		return
	}
	choice := strings.ToLower(strings.TrimSpace(cc.Scanner.Text()))

	switch choice {
	case "a":
		for _, f := range folders {
			removeFolder(FolderRemoveInput{Database: cc.Database, DirPath: f.path, DirName: f.name, BatchID: cc.BatchID})
		}
	case "s":
		selectiveFolderRemoval(cc, folders)
	case "l":
		listThenDecide(cc, folders)
	case "n":
		fmt.Println("  📁 All folders kept.")
	default:
		fmt.Println("  📁 No folders removed.")
	}
}

func selectiveFolderRemoval(cc CleanupContext, folders []popoutFolderInfo) {
	for _, f := range folders {
		status := "empty"
		if len(f.files) > 0 {
			status = fmt.Sprintf("%d files (%s)", len(f.files), humanSize(f.totalSize))
		}
		fmt.Printf("\n  %s/ — %s\n", f.name, status)
		if len(f.files) > 0 {
			fmt.Println("    Files:")
			for _, file := range f.files {
				fmt.Printf("      - %s\n", file)
			}
		}
		fmt.Print("    Remove? [y/N]: ")
		if !cc.Scanner.Scan() {
			return
		}
		answer := strings.ToLower(strings.TrimSpace(cc.Scanner.Text()))
		if answer == "y" || answer == "yes" {
			removeFolder(FolderRemoveInput{Database: cc.Database, DirPath: f.path, DirName: f.name, BatchID: cc.BatchID})
			continue
		}
		fmt.Println("    Kept.")
	}
}

func listThenDecide(cc CleanupContext, folders []popoutFolderInfo) {
	for _, f := range folders {
		fmt.Printf("\n  📁 %s/\n", f.name)
		if len(f.files) == 0 {
			fmt.Println("    (empty)")
		}
		if len(f.files) > 0 {
			for _, file := range f.files {
				fmt.Printf("    - %s\n", file)
			}
		}
		fmt.Print("    Remove? [y/N]: ")
		if !cc.Scanner.Scan() {
			return
		}
		answer := strings.ToLower(strings.TrimSpace(cc.Scanner.Text()))
		if answer == "y" || answer == "yes" {
			removeFolder(FolderRemoveInput{Database: cc.Database, DirPath: f.path, DirName: f.name, BatchID: cc.BatchID})
			continue
		}
		fmt.Println("    Kept.")
	}
}

func removeFolder(input FolderRemoveInput) {
	if err := os.RemoveAll(input.DirPath); err != nil {
		errlog.Error("Failed to remove %s: %v", input.DirPath, err)
		return
	}
	fmt.Printf("    🗑  Removed: %s/\n", input.DirName)
	detail := fmt.Sprintf("Removed folder: %s/", input.DirName)
	snapshot := fmt.Sprintf(`{"folder_path":"%s"}`, input.DirPath)
	input.Database.InsertActionSimple(db.ActionSimpleInput{
		FileAction: db.FileActionDelete, Snapshot: snapshot,
		Detail: detail, BatchID: input.BatchID,
	})
}
