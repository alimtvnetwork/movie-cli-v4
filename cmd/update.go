// update.go — implements the `movie update` command.
// Uses the copy-and-handoff pattern from gitmap-v2 to bypass Windows file locks.
// See spec/13-self-update-app-update/ for full architecture documentation.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/updater"
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
		exitOnUpdateError("Update failed", updater.Run())
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

		targetBinary, _ := cmd.Flags().GetString("target-binary")
		if targetBinary == "" {
			fmt.Fprintln(os.Stderr, "❌ --target-binary is required for update-runner")
			os.Exit(1)
		}

		exitOnUpdateError("Update worker failed", updater.RunWorker(repoPath, targetBinary))
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
			return
		}
		fmt.Println("✔ No update artifacts found")
	},
}

func init() {
	updateRunnerCmd.Flags().String("repo-path", "", "Path to the source repository")
	updateRunnerCmd.Flags().String("target-binary", "", "Original executable path to redeploy")
}

func exitOnUpdateError(label string, err error) {
	if err == nil {
		return
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}

	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", label, err)
	os.Exit(1)
}
