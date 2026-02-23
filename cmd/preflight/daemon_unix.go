//go:build !windows

package main

import "syscall"

// daemonProcAttr returns process attributes for detaching the daemon on Unix.
func daemonProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
