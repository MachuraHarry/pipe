//go:build !windows && !js && !linux

package mcp

import (
	"os/exec"
	"syscall"
)

// setPdeathsig on Darwin/BSD: there is no Pdeathsig equivalent here (it's a
// Linux-only prctl(PR_SET_PDEATHSIG) feature — see procattr_linux.go), so a
// crashed Pipe process cannot make the kernel auto-kill an MCP subprocess on
// this platform. Setpgid is still set so killProcessGroup (procattr_unix.go)
// can reach the whole subtree on a *graceful* shutdown, same as on Linux —
// only the crash-without-cleanup case loses coverage here.
func setPdeathsig(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
