package updater

import (
	"os"
	"os/exec"
)

// runAttached runs a command in the current console session.
func runAttached(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}