// Package updater implements the copy-and-handoff self-update mechanism.
//
// Architecture (from spec/13-self-update-app-update/):
//
//	movie update → copies self → launches copy with "update-runner" → worker runs run.ps1 → deploys new binary
//
// This bypasses the Windows file-lock problem where a running binary cannot overwrite itself.
package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/apperror"
)

// repoURL is the canonical GitHub URL used when no local repo exists.
const repoURL = "https://github.com/alimtvnetwork/movie-cli-v3.git"

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
		return printBootstrapInfo(repoPath)
	}

	if err := prepareRepoBranch(repoPath); err != nil {
		return err
	}

	selfPath, err := resolveSelfPath()
	if err != nil {
		return err
	}

	copyPath := createHandoffCopy(selfPath)
	fmt.Printf("🔄 Starting update from %s\n", repoPath)

	return launchHandoff(copyPath, repoPath, selfPath)
}

func printBootstrapInfo(repoPath string) error {
	commit, _ := gitOutput(repoPath, "rev-parse", "--short", "HEAD")
	fmt.Printf("\n✨ Bootstrapped local source repo in %s\n", repoPath)
	fmt.Printf("🔁 Commit: %s\n", commit)
	fmt.Println("\n💡 Run 'movie update' again to build and deploy")
	return nil
}

func prepareRepoBranch(repoPath string) error {
	gm, gmErr := readGitMapLatest(repoPath)
	branch := ""
	if gmErr == nil && gm.Branch != "" {
		branch = gm.Branch
		fmt.Printf("📋 Gitmap: %s (branch: %s)\n", gm.Version, branch)
	}

	dirty, err := gitOutput(repoPath, "status", "--porcelain")
	if err != nil {
		return apperror.Wrap("cannot check git status", err)
	}
	if strings.TrimSpace(dirty) != "" {
		return apperror.New("repository has local changes; commit or stash them before update")
	}

	if branch != "" {
		currentBranch, _ := gitOutput(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
		if currentBranch != branch {
			fmt.Printf("🔀 Switching from %s to %s\n", currentBranch, branch)
			if _, checkoutErr := gitOutput(repoPath, "checkout", branch); checkoutErr != nil {
				return apperror.Wrap("cannot checkout branch from gitmap", checkoutErr)
			}
		}
	}
	return nil
}

func resolveSelfPath() (string, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return "", apperror.Wrap("cannot determine executable path", err)
	}
	resolved, resolveErr := filepath.EvalSymlinks(selfPath)
	if resolveErr == nil {
		return resolved, nil
	}
	return selfPath, nil
}

// RunWorker is the hidden update-runner entry point called from the handoff copy.
func RunWorker(repoPath, targetBinary string) error {
	fmt.Println("🔧 Update worker started")
	fmt.Printf("📂 Repo: %s\n", repoPath)
	fmt.Printf("🎯 Target: %s\n", targetBinary)

	if runtime.GOOS == "windows" {
		return executeUpdateWindows(repoPath, targetBinary)
	}
	return executeUpdateUnix(repoPath, targetBinary)
}
