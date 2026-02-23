//go:build windows

package main

import "syscall"

// daemonProcAttr returns process attributes for detaching the daemon on Windows.
func daemonProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
