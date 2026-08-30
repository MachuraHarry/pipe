//go:build windows || js

package mcp

import "os/exec"

// setPdeathsig is a no-op on platforms without Pdeathsig (Windows, WASM/js).
// See procattr_unix.go for the Unix implementation and rationale.
func setPdeathsig(cmd *exec.Cmd) {
}

// killProcessGroup falls back to killing just the direct child on platforms
// without process-group signaling parity with Unix's negative-PID kill(2).
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}
