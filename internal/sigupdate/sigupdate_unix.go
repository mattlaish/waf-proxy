//go:build !windows

package sigupdate

import (
	"os"
	"syscall"
)

// ReExec replaces the current process image with a fresh exec of the running
// binary — used after an update replaces the binary on disk, so the new code
// takes effect. It does not return on success.
//
// On Linux, renaming a new binary over a running one is safe (the running
// process keeps the old inode); ReExec then launches the new file.
//
// NOTE: like any restart, this drops in-memory state (sessions, and for a vault
// product, the unsealed key). Warn the operator before calling it.
func ReExec() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}
