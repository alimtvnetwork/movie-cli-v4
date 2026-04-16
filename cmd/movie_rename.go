// movie_rename.go — movie rename
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/cleaner"
	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var movieRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename files to clean names",
	Long: `Automatically renames messy filenames to clean format.
Example: Scream.2022.1080p.WEBRip.x264-RARBG.mkv → Scream (2022).mkv`,
	Run: runMovieRename,
}

// renameItem groups data for a single rename operation.
type renameItem struct {
	oldPath string
	newPath string
	oldName string
	newName string
	media   db.Media
}

func runMovieRename(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error(msgDatabaseError, err)
		return
	}
	defer database.Close()

	items := findRenameCandidates(database)
	if items == nil {
		return
	}

	printRenamePreview(items)

	if !confirmRenameAction() {
		return
	}

	executeRenames(database, items)
}

func findRenameCandidates(database *db.DB) []renameItem {
	media, listErr := database.ListMedia(0, 10000)
	if listErr != nil {
		errlog.Error("Failed to read media: %v", listErr)
		return nil
	}
	if len(media) == 0 {
		fmt.Println("📭 No media found.")
		return nil
	}

	var items []renameItem
	for i := range media {
		if item, ok := buildRenameItem(&media[i]); ok {
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		fmt.Println("✅ All files already have clean names!")
		return nil
	}
	return items
}

func buildRenameItem(m *db.Media) (renameItem, bool) {
	if m.CurrentFilePath == "" {
		return renameItem{}, false
	}
	dir := filepath.Dir(m.CurrentFilePath)
	oldName := filepath.Base(m.CurrentFilePath)
	newName := cleaner.ToCleanFileName(m.CleanTitle, m.Year, m.FileExtension)
	if oldName == newName {
		return renameItem{}, false
	}
	return renameItem{
		media: *m, oldPath: m.CurrentFilePath,
		newPath: filepath.Join(dir, newName),
		oldName: oldName, newName: newName,
	}, true
}

func printRenamePreview(items []renameItem) {
	fmt.Printf("📝 Found %d files to rename:\n\n", len(items))
	for i := range items {
		fmt.Printf("  %d. %s\n", i+1, items[i].oldName)
		fmt.Printf("     → %s\n\n", items[i].newName)
	}
}

func confirmRenameAction() bool {
	fmt.Print("Rename all? [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("❌ Canceled.")
		return false
	}
	return true
}

func executeRenames(database *db.DB, items []renameItem) {
	success := 0
	for i := range items {
		if executeSingleRename(database, &items[i]) {
			success++
		}
	}
	fmt.Printf("\n✅ Renamed %d/%d files.\n", success, len(items))
}

func executeSingleRename(database *db.DB, item *renameItem) bool {
	if moveErr := MoveFile(item.oldPath, item.newPath); moveErr != nil {
		errlog.Error("Failed: %s → %v", item.oldName, moveErr)
		return false
	}
	if updateErr := database.UpdateMediaPath(item.media.ID, item.newPath); updateErr != nil {
		errlog.Warn("DB update path error: %v", updateErr)
	}
	if histErr := database.InsertMoveHistory(db.MoveInput{
		MediaID: item.media.ID, FileActionID: int(db.FileActionRename),
		FromPath: item.oldPath, ToPath: item.newPath,
		OrigName: item.oldName, NewName: item.newName,
	}); histErr != nil {
		errlog.Warn("DB history error: %v", histErr)
	}
	fmt.Printf("  ✅ %s → %s\n", item.oldName, item.newName)
	return true
}
