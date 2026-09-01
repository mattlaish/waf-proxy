//go:build linux

package main

import "syscall"

// freebindControl sets IP_FREEBIND on the socket before bind, allowing the WAF
// to listen on an IP address that is not (currently) assigned to a local
// interface — e.g. a floating/VIP address, or one of several service IPs you
// manage outside the box. Without it the kernel rejects the bind with
// "cannot assign requested address".
//
// stdlib-only: syscall.IP_FREEBIND and syscall.SOL_IP are provided by the
// standard library on Linux; no external packages.
func freebindControl(network, address string, c syscall.RawConn) error {
	var setErr error
	if err := c.Control(func(fd uintptr) {
		setErr = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_FREEBIND, 1)
	}); err != nil {
		return err
	}
	return setErr
}
