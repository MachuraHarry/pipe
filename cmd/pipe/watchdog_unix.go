//go:build !windows && !js

package main

import (
	"io"
	"os"
	"strconv"
	"syscall"
)

// runMCPWatchdog implements the hidden "--mcp-watchdog" mode: consume stdin
// until EOF (i.e. until the parent pipe process died — the kernel closes the
// write end on process teardown regardless of the exit mode), then kill the
// MCP server's whole process group. See pkg/mcp/procattr_unix.go for the
// full rationale (grandchildren survive Pdeathsig).
//
// syscall.Kill has no Windows/js equivalent, hence this file's build tag —
// see watchdog_other.go for the stub used there. This mode is only ever
// spawned by pkg/mcp/procattr_unix.go's startMCPWatchdog, itself built only
// for !windows && !js, so the stub is never actually reached in practice.
func runMCPWatchdog(pgidArg string) {
	_, _ = io.Copy(io.Discard, os.Stdin)
	if pgid, err := strconv.Atoi(pgidArg); err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
	os.Exit(0)
}
