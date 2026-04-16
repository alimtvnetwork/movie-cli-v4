package updater

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/alimtvnetwork/movie-cli-v4/apperror"
)

// createHandoffCopy creates a temporary copy of the binary for the handoff worker.
func createHandoffCopy(selfPath string) (string, error) {
	name := handoffName()
	copyPath := filepath.Join(filepath.Dir(selfPath), name)

	if copyFile(selfPath, copyPath) == nil {
		makeExecutable(copyPath)
		return copyPath, nil
	}

	// Fallback to temp directory
	copyPath = filepath.Join(os.TempDir(), name)
	if err := copyFile(selfPath, copyPath); err != nil {
		return "", apperror.Wrap("cannot create handoff copy", err)
	}
	makeExecutable(copyPath)
	return copyPath, nil
}

// launchHandoff starts the handoff binary in foreground (blocking) so
// the terminal stays stable and the user sees all output.
func launchHandoff(copyPath, repoPath, targetBinary string) error {
	args := []string{
		"update-runner",
		"--repo-path", repoPath,
		"--target-binary", targetBinary,
	}

	fmt.Printf("🚀 Update handed off to %s\n", copyPath)

	cmd := exec.Command(copyPath, args...)
	if err := runAttached(cmd); err != nil {
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
