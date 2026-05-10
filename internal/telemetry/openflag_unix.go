//go:build unix

package telemetry

import "syscall"

var openFlagNoFollow = syscall.O_NOFOLLOW
