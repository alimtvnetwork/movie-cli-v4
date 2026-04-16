package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// findRepoPath locates the git repository root by checking (in order):
//  1. The directory containing the running binary
//  2. A sibling movie-cli-v4/ clone next to the binary
//  3. The current working directory
//  4. Bootstrap clone (fresh clone next to the binary)
func findRepoPath() (string, bool, error) {
	exe, exeErr := os.Executable()
	if exeErr == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir := filepath.Dir(exe)

		// 1. Binary's own directory
		if isValidRepo(exeDir) {
			return repoRoot(exeDir), false, nil
		}

		// 2. Sibling clone
		sibling := filepath.Join(exeDir, "movie-cli-v4")
		if isValidRepo(sibling) {
			return repoRoot(sibling), false, nil
		}
	}

	// 3. CWD
	cwd, cwdErr := os.Getwd()
	if cwdErr == nil && isValidRepo(cwd) {
		return repoRoot(cwd), false, nil
	}

	// 4. Bootstrap clone
	if exeErr == nil {
		exeDir := filepath.Dir(exe)
		cloneDir := filepath.Join(exeDir, "movie-cli-v4")
		fmt.Printf("📥 No local repo found. Cloning to: %s\n", cloneDir)
		if _, cloneErr := gitOutput(exeDir, "clone", "--depth", "1", repoURL); cloneErr != nil {
			return "", false, fmt.Errorf("cannot clone repository: %w", cloneErr)
		}
		return cloneDir, true, nil
	}

	return "", false, fmt.Errorf("cannot locate the movie-cli-v4 repository")
}

// isValidRepo checks if a directory is a valid movie-cli-v4 repo
// by verifying both .git and go.mod exist.
func isValidRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	goMod := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(gitDir); err != nil {
		return false
	}
	if _, err := os.Stat(goMod); err != nil {
		return false
	}
	return true
}

// repoRoot uses git to resolve the actual repo root from a subdirectory.
func repoRoot(dir string) string {
	p, err := gitOutput(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return dir
	}
	return p
}

// gitOutput runs a git command in the given directory and returns trimmed stdout.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", text)
	}
	return text, nil
}
