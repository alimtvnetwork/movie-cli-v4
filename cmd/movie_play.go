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
		errlog.Error("Database error: %v", err)
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

	filePath := m.CurrentFilePath
	if _, statErr := os.Stat(filePath); statErr != nil {
		if os.IsNotExist(statErr) {
			errlog.Error("File not found: %s", filePath)
			return
		}
		errlog.Error("Cannot access file %s: %v", filePath, statErr)
		return
	}

	fmt.Printf("▶️  Playing: %s", m.CleanTitle)
	if m.Year > 0 {
		fmt.Printf(" (%d)", m.Year)
	}
	fmt.Println()

	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		openCmd = exec.Command("open", filePath)
	case "linux":
		openCmd = exec.Command("xdg-open", filePath)
	case "windows":
		openCmd = exec.Command("cmd", "/c", "start", "", filePath)
	default:
		errlog.Error("Unsupported OS: %s", runtime.GOOS)
		return
	}

	if err := openCmd.Start(); err != nil {
		errlog.Error("Cannot open player: %v", err)
		return
	}
}
