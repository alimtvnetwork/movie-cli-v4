// Package updater implements the copy-and-handoff self-update mechanism.
//
// Architecture (from spec/13-self-update-app-update/):
//
//	movie update → copies self → launches copy with "update-runner" → worker runs run.ps1 → deploys new binary
//
// This bypasses the Windows file-lock problem where a running binary cannot overwrite itself.
package updater

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// repoURL is the canonical GitHub URL used when no local repo exists.
const repoURL = "https://github.com/alimtvnetwork/movie-cli-v4.git"

// Run executes the update command: resolves repo, creates handoff copy, launches worker.
func Run() error {
	if _, err := exec.LookPath("git"); err != nil {
		return apperror.New("git is not installed or not in PATH")
	}

	repoPath, bootstrapped, err := findRepoPath()
	if err != nil {
		return err
	}

	if bootstrapped {
		commit, _ := gitOutput(repoPath, "rev-parse", "--short", "HEAD")
		fmt.Printf("\n✨ Bootstrapped local source repo in %s\n", repoPath)
		fmt.Printf("🔁 Commit: %s\n", commit)
		fmt.Println("\n💡 Run 'movie update' again to build and deploy")
		return nil
	}

	// Check for local changes
	dirty, err := gitOutput(repoPath, "status", "--porcelain")
	if err != nil {
		return apperror.Wrap("cannot check git status", err)
	}
	if strings.TrimSpace(dirty) != "" {
		return apperror.New("repository has local changes; commit or stash them before update")
	}

	selfPath, err := os.Executable()
	if err != nil {
		return apperror.Wrap("cannot determine executable path", err)
	}

	copyPath := createHandoffCopy(selfPath)
	fmt.Printf("🔄 Starting update from %s\n", repoPath)

	return launchHandoff(copyPath, repoPath)
}

// RunWorker is the hidden update-runner entry point called from the handoff copy.
func RunWorker(repoPath string) error {
	fmt.Println("🔧 Update worker started")
	fmt.Printf("📂 Repo: %s\n", repoPath)

	if runtime.GOOS == "windows" {
		return executeUpdateWindows(repoPath)
	}
	return executeUpdateUnix(repoPath)
}
