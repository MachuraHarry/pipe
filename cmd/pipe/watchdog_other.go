//go:build windows || js

package main

import "os"

// runMCPWatchdog is a no-op stub on platforms where the watchdog is never
// actually spawned (see pkg/mcp/procattr_unix.go's startMCPWatchdog, built
// only for !windows && !js) — kept here only so the hidden "--mcp-watchdog"
// CLI flag still compiles and exits cleanly if it's ever invoked directly.
func runMCPWatchdog(pgidArg string) {
	os.Exit(0)
}
