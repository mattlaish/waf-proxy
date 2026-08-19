//go:build windows

package sigupdate

import (
	"os"
	"os/exec"
)

// ReExec starts the newly installed executable and exits the current process.
// Windows cannot replace a process image in place like Unix syscall.Exec, so
// this is the closest equivalent. The child inherits the current arguments,
// environment, and standard streams.
func ReExec() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
