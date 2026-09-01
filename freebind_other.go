//go:build !linux

package main

import "syscall"

// freebindControl is a no-op on non-Linux platforms (IP_FREEBIND is Linux-only).
// Binding an unassigned IP will fail as usual off Linux, which is fine — the
// appliance target is Debian/Linux.
func freebindControl(network, address string, c syscall.RawConn) error { return nil }
