//go:build linux

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
// Pdeathsig is a Linux-only prctl(PR_SET_PDEATHSIG) feature — there is no
// BSD/Darwin equivalent, hence this file's "linux" build tag; see
// procattr_bsd.go for the reduced (Setpgid-only) behavior everywhere else.
//
// Also sets Setpgid so this child (e.g. `npm exec ...`) becomes the leader
// of its own new process group, distinct from Pipe's. Pdeathsig is a
// per-process attribute that is NOT inherited across fork() — verified
// live: killing Pipe correctly killed the direct `npm exec` child via
// Pdeathsig, but the grandchildren it forked on its own (`sh -c ...` ->
// `node ...`) were unaffected and stayed orphaned. Setpgid alone does not
// kill anything by itself, but it lets Close() (see killProcessGroup in
// procattr_unix.go) target the WHOLE subtree with one signal on graceful
// shutdown, which Pdeathsig structurally cannot do.
func setPdeathsig(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL, Setpgid: true}
}
