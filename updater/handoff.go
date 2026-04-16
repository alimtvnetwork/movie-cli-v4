package updater

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// createHandoffCopy creates a temporary copy of the binary for the handoff worker.
func createHandoffCopy(selfPath string) string {
	name := handoffName()
	copyPath := filepath.Join(filepath.Dir(selfPath), name)

	if copyFile(selfPath, copyPath) == nil {
		makeExecutable(copyPath)
		return copyPath
	}

	// Fallback to temp directory
	copyPath = filepath.Join(os.TempDir(), name)
	if err := copyFile(selfPath, copyPath); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot create handoff copy: %v\n", err)
		os.Exit(1)
	}
	makeExecutable(copyPath)
	return copyPath
}

// launchHandoff runs the handoff binary with the update-runner command (foreground/blocking).
func launchHandoff(copyPath, repoPath string) error {
	args := []string{"update-runner", "--repo-path", repoPath}

	cmd := exec.Command(copyPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return apperror.Wrap("update worker failed", err)
	}
	return nil
}

// handoffName returns the temp binary name with PID suffix.
func handoffName() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("movie-update-%d.exe", os.Getpid())
	}
	return fmt.Sprintf("movie-update-%d", os.Getpid())
}

// makeExecutable sets +x permission on Unix systems.
func makeExecutable(path string) {
	if runtime.GOOS == "windows" {
		return
	}
	_ = os.Chmod(path, 0o755)
}

// copyFile copies src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
