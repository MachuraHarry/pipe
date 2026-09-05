//go:build !windows && !js

package mcp

import (
	"os"
	"os/exec"
	"strconv"
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

// startMCPWatchdog spawns a tiny companion process — a re-exec of this very
// binary in the hidden "--mcp-watchdog <pgid>" mode (see cmd/pipe/main.go) —
// that blocks reading from the returned pipe's write end until EOF. When
// THIS process dies, no matter how (crash, SIGKILL, os.Exit(1) from a
// runtime error, normal shutdown), the kernel closes the write end, the
// watchdog sees EOF and SIGKILLs the MCP server's ENTIRE process group
// (negative PID).
//
// This closes the gap Pdeathsig structurally cannot: Pdeathsig only kills
// the DIRECT child and is not inherited across fork(), so grandchildren
// (e.g. `npm exec` -> `sh -c` -> `node`, or `uv` -> `python`) were orphaned
// on every non-graceful exit and accumulated indefinitely. The graceful
// path (Client.Close) kills the group itself and only then closes the pipe;
// the watchdog then fires as a harmless no-op on an already-dead group.
//
// The watchdog deliberately has no Pdeathsig of its own: its trigger is the
// EOF, and the write end is held exclusively by this process (exec.Cmd
// passes fds to children only via Explicitly-listed stdin/stdout/stderr/
// ExtraFiles), so EOF is guaranteed on this process's death in any mode.
func startMCPWatchdog(cmd *exec.Cmd) (*os.File, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	wd := exec.Command("/proc/self/exe", "--mcp-watchdog", strconv.Itoa(cmd.Process.Pid))
	wd.Stdin = pr
	wd.Stdout = nil
	wd.Stderr = nil
	startErr := wd.Start()
	pr.Close() // the watchdog holds its own copy of the read end
	if startErr != nil {
		pw.Close()
		return nil, startErr
	}
	return pw, nil
}
