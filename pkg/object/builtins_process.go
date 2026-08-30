package object

import (
	"bytes"
	"os/exec"
	"sync"
)

// ProcessHandle tracks a running subprocess started by proc_start().
type ProcessHandle struct {
	Cmd    *exec.Cmd
	Output []byte
	Err    error
	Done   bool
	mu     sync.Mutex
	doneCh chan struct{}
}

// processRegistry maps integer handles to running processes.
var processRegistry = struct {
	sync.RWMutex
	handles map[int]*ProcessHandle
	nextID  int
}{
	handles: make(map[int]*ProcessHandle),
	nextID:  1,
}

func getProcess(id int) *ProcessHandle {
	processRegistry.RLock()
	defer processRegistry.RUnlock()
	return processRegistry.handles[id]
}

func removeProcess(id int) {
	processRegistry.Lock()
	defer processRegistry.Unlock()
	delete(processRegistry.handles, id)
}

func newErr(msg string) Object {
	return &Error{Message: msg}
}

// bProcStart starts a process asynchronously and returns an integer handle.
// Usage: handle: proc_start("ls -la")
func bProcStart(args ...Object) Object {
	if len(args) != 1 {
		return newErr("proc_start expects 1 argument (command string)")
	}
	cmd, ok := args[0].(*String)
	if !ok {
		return newErr("proc_start: argument must be a string")
	}

	// Security: same gating as exec — profile whitelist + legacy sandbox
	profile := ActiveProfile.Load()
	if profile.Name != "none" {
		if canErr := profile.CanExecCommand(cmd.Value); canErr != nil {
			return newErr(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowExec {
		return newErr(sandboxBlock("proc_start").Message)
	}

	c, cancel, cerr := buildExecCommand(profile, cmd.Value)
	if cerr != nil {
		return cerr
	}
	if cancel != nil {
		defer cancel()
	}

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Start(); err != nil {
		return newErr("proc_start: " + err.Error())
	}

	// Register process AFTER Start() succeeds
	processRegistry.Lock()
	id := processRegistry.nextID
	processRegistry.nextID++
	h := &ProcessHandle{Cmd: c, doneCh: make(chan struct{})}
	processRegistry.handles[id] = h
	processRegistry.Unlock()

	// Wait for exit in background
	go func() {
		err := c.Wait()
		h.mu.Lock()
		h.Output = append(stdout.Bytes(), stderr.Bytes()...)
		h.Err = err
		h.Done = true
		h.mu.Unlock()
		close(h.doneCh)
	}()

	return &Integer{Value: int64(id)}
}

// bProcWait blocks until the process finishes and returns {output, error, status}.
// Usage: result: proc_wait(handle)
func bProcWait(args ...Object) Object {
	if len(args) != 1 {
		return newErr("proc_wait expects 1 argument (process handle)")
	}
	id, ok := args[0].(*Integer)
	if !ok {
		return newErr("proc_wait: argument must be an integer handle from proc_start")
	}

	h := getProcess(int(id.Value))
	if h == nil {
		return newErr("proc_wait: invalid process handle")
	}

	// Wait for background goroutine to finish
	<-h.doneCh

	h.mu.Lock()
	defer h.mu.Unlock()

	status := 0
	if h.Err != nil {
		if exitErr, ok := h.Err.(*exec.ExitError); ok {
			status = exitErr.ExitCode()
		} else {
			status = -1
		}
	}

	output := ""
	if h.Output != nil {
		output = string(h.Output)
	}

	errStr := ""
	if h.Err != nil {
		errStr = h.Err.Error()
	}

	return &Map{Pairs: map[string]Object{
		"output": &String{Value: output},
		"error":  &String{Value: errStr},
		"status": &Integer{Value: int64(status)},
	}}
}

// bProcKill terminates a running process.
// Usage: proc_kill(handle)
func bProcKill(args ...Object) Object {
	if len(args) != 1 {
		return newErr("proc_kill expects 1 argument (process handle)")
	}
	id, ok := args[0].(*Integer)
	if !ok {
		return newErr("proc_kill: argument must be an integer handle from proc_start")
	}

	h := getProcess(int(id.Value))
	if h == nil {
		return newErr("proc_kill: invalid process handle")
	}

	if err := h.Cmd.Process.Kill(); err != nil {
		return newErr("proc_kill: " + err.Error())
	}

	return TRUE
}

// bProcRunning checks if a process is still running.
// Usage: proc_running(handle) → true/false
func bProcRunning(args ...Object) Object {
	if len(args) != 1 {
		return newErr("proc_running expects 1 argument (process handle)")
	}
	id, ok := args[0].(*Integer)
	if !ok {
		return newErr("proc_running: argument must be an integer handle from proc_start")
	}

	h := getProcess(int(id.Value))
	if h == nil {
		return FALSE
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	return NativeBoolToBoolean(!h.Done)
}
