// movie_undo.go — movie undo
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/db"
	"github.com/alimtvnetwork/movie-cli-v3/errlog"
)

var movieUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo the last move operation",
	Long:  `Reverts the most recent movie move operation.`,
	Run:   runMovieUndo,
}

func runMovieUndo(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	lastMove, moveErr := database.GetLastMove()
	if moveErr != nil {
		fmt.Println("📭 No move operations to undo.")
		return
	}

	fmt.Println("⏪ Last move operation:")
	fmt.Println()
	fmt.Printf("  📁 %s\n", lastMove.ToPath)
	fmt.Printf("  → %s\n", lastMove.FromPath)
	fmt.Println()

	// Confirmation prompt
	fmt.Print("Undo this? [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("❌ Undo canceled.")
		return
	}

	// Check source exists
	if _, statErr := os.Stat(lastMove.ToPath); statErr != nil {
		if os.IsNotExist(statErr) {
			errlog.Error("File not found at: %s — it may have been moved or deleted manually.", lastMove.ToPath)
		} else {
			errlog.Error("Cannot access file %s: %v", lastMove.ToPath, statErr)
		}
		return
	}

	// Move back
	if undoErr := MoveFile(lastMove.ToPath, lastMove.FromPath); undoErr != nil {
		errlog.Error("Undo failed: %v", undoErr)
		return
	}

	// Mark as undone in DB
	if markErr := database.MarkMoveUndone(lastMove.ID); markErr != nil {
		errlog.Warn("Could not mark move as undone: %v", markErr)
	}

	// Update media path back
	if pathErr := database.UpdateMediaPath(lastMove.MediaID, lastMove.FromPath); pathErr != nil {
		errlog.Warn("Could not update media path: %v", pathErr)
	}

	fmt.Println()
	fmt.Println("✅ Undo successful!")
	fmt.Printf("   %s\n", lastMove.ToPath)
	fmt.Printf("   → %s\n", lastMove.FromPath)
}
