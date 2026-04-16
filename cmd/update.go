// update.go — implements the `movie update` command.
// Uses the copy-and-handoff pattern from gitmap-v2 to bypass Windows file locks.
// See spec/13-self-update-app-update/ for full architecture documentation.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v3/updater"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update movie-cli to the latest version",
	Long: `Updates movie-cli by pulling latest source, rebuilding, and deploying.

The update process:
  1. Finds the source repository (binary dir, CWD, or sibling clone)
  2. Creates a handoff copy of the binary (bypasses Windows file locks)
  3. Pulls latest code from GitHub
  4. Rebuilds via run.ps1 (go mod tidy → go build → deploy)
  5. Compares version before/after and shows changelog

If no local repo is found, it clones a fresh copy next to the binary.
Run 'movie update' again after bootstrap to build.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := updater.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Update failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var updateRunnerCmd = &cobra.Command{
	Use:    "update-runner",
	Hidden: true,
	Short:  "Internal worker for update handoff",
	Run: func(cmd *cobra.Command, args []string) {
		repoPath, _ := cmd.Flags().GetString("repo-path")
		if repoPath == "" {
			fmt.Fprintln(os.Stderr, "❌ --repo-path is required for update-runner")
			os.Exit(1)
		}
		if err := updater.RunWorker(repoPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Update worker failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var updateCleanupCmd = &cobra.Command{
	Use:   "update-cleanup",
	Short: "Remove leftover temp files from previous updates",
	Long: `Removes temporary artifacts created during the update process:
  - Handoff binary copies (movie-update-*.exe)
  - Backup binaries (*.bak)`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🧹 Cleaning update artifacts...")
		cleaned, err := updater.Cleanup()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Cleanup failed: %v\n", err)
			os.Exit(1)
		}
		if cleaned > 0 {
			fmt.Printf("✔ Cleaned %d artifact(s)\n", cleaned)
		} else {
			fmt.Println("✔ No update artifacts found")
		}
	},
}

func init() {
	updateRunnerCmd.Flags().String("repo-path", "", "Path to the source repository")
}
