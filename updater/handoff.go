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

// launchHandoff starts the handoff binary and returns immediately so the
// current process can exit and release its file lock before rebuild/deploy.
func launchHandoff(copyPath, repoPath, targetBinary string) error {
	args := []string{
		"update-runner",
		"--repo-path", repoPath,
		"--target-binary", targetBinary,
	}

	cmd := exec.Command(copyPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return apperror.Wrap("cannot start update worker", err)
	}
	_ = cmd.Process.Release()
	fmt.Printf("🚀 Update handed off to %s\n", copyPath)
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
