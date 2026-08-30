//go:build !windows && !js

package mcp

import (
	"os/exec"
	"syscall"
)

// setPdeathsig arranges for a stdio MCP subprocess to receive SIGKILL from
// the kernel the moment its parent process dies (crash, SIGKILL, unhandled
// SIGTERM — anything). Pipe registers no shutdown hook for MCP clients today
// (no signal.Notify anywhere in cmd/pipe or pkg/object/pkg/mcp), so without
// this the child is simply orphaned (reparented to PID 1) on parent exit.
// Most MCP server packages happen to notice their stdin pipe close and exit
// on their own, but that is package-specific best-effort behavior, not a
// guarantee — reproduced live with @dangahagan/weather-mcp, which does not
// exit on its own and accumulates as an orphaned process on every restart.
// Pdeathsig fixes this at the kernel level regardless of the child's own
// code.
//
// Also sets Setpgid so this child (e.g. `npm exec ...`) becomes the leader
// of its own new process group, distinct from Pipe's. Pdeathsig is a
// per-process attribute that is NOT inherited across fork() — verified
// live: killing Pipe correctly killed the direct `npm exec` child via
// Pdeathsig, but the grandchildren it forked on its own (`sh -c ...` ->
// `node ...`) were unaffected and stayed orphaned. Setpgid alone does not
// kill anything by itself, but it lets Close() (see killProcessGroup below)
// target the WHOLE subtree with one signal on graceful shutdown, which
// Pdeathsig structurally cannot do.
func setPdeathsig(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL, Setpgid: true}
}

// killProcessGroup sends SIGKILL to the entire process group led by cmd's
// process (negative PID = process group, see kill(2)) — reaches
// grandchildren a plain cmd.Process.Kill() (which only signals the single
// direct child) cannot, as long as Setpgid was set at spawn time (see
// setPdeathsig above).
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
