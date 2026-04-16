// movie_play.go — movie play <id>
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var moviePlayCmd = &cobra.Command{
	Use:   "play [id]",
	Short: "Play a movie or TV show with the default player",
	Long:  `Opens a media file with the system's default video player.`,
	Args:  cobra.ExactArgs(1),
	Run:   runMoviePlay,
}

func runMoviePlay(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error(msgDatabaseError, err)
		return
	}
	defer database.Close()

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		errlog.Error("Invalid ID.")
		return
	}

	m, err := database.GetMediaByID(id)
	if err != nil {
		errlog.Error("Media not found: %v", err)
		return
	}

	if !validateFilePath(m.CurrentFilePath) {
		return
	}

	fmt.Printf("▶️  Playing: %s", m.CleanTitle)
	if m.Year > 0 {
		fmt.Printf(" (%d)", m.Year)
	}
	fmt.Println()

	launchPlayer(m.CurrentFilePath)
}

func validateFilePath(filePath string) bool {
	_, statErr := os.Stat(filePath)
	if statErr == nil {
		return true
	}
	if os.IsNotExist(statErr) {
		errlog.Error("File not found: %s", filePath)
		return false
	}
	errlog.Error("Cannot access file %s: %v", filePath, statErr)
	return false
}

func launchPlayer(filePath string) {
	openCmd := buildOpenCommand(filePath)
	if openCmd == nil {
		errlog.Error("Unsupported OS: %s", runtime.GOOS)
		return
	}
	if err := openCmd.Start(); err != nil {
		errlog.Error("Cannot open player: %v", err)
	}
}

func buildOpenCommand(filePath string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", filePath)
	case "linux":
		return exec.Command("xdg-open", filePath)
	case "windows":
		return exec.Command("cmd", "/c", "start", "", filePath)
	default:
		return nil
	}
}
