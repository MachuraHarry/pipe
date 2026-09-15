package vm

import "sync"

// DebugSession implements pause/resume/step for one VM's instruction loop.
// A single VM's execution is single-threaded from the debugger's point of
// view: check() is only ever called from that VM's own Run()/executeFrame()
// goroutine (via decodeAndCheck), so the paused goroutine is that VM's own
// -- it blocks itself on resumeCh, not some other thread.
//
// Deliberately NOT attempting to coordinate across multiple concurrently
// running VMs (spawn creates a fully independent *VM per call, its own
// goroutine, no shared execution state -- see spawnClosure/newSpawnVM).
// Properly pausing/stepping *all* concurrently spawned code is a materially
// harder problem (a live registry of every VM, a real multi-thread DAP
// "threads" response, races when several are paused simultaneously
// inspecting shared-mutable globals) than this first version needs. A
// DebugSession is only ever attached to the single top-level VM a debug
// launch creates; spawned child VMs simply run unmodified/un-debuggable.
type DebugSession struct {
	mu          sync.Mutex
	breakpoints map[int]bool
	mode        string // "run" | "step-into" | "step-over" | "step-out"
	// stepFrameIndex is the frameIndex a step-over/step-out request compares
	// against (copied from lastStoppedFrameIndex when the step is issued);
	// see check()'s step-over/step-out cases for how it's used.
	stepFrameIndex int
	// lastStoppedFrameIndex is the frameIndex check() most recently stopped
	// at, so StepOver/StepOut (called later, while still paused) know which
	// frame the step is relative to without the DAP layer having to track
	// and pass it back in itself.
	lastStoppedFrameIndex int
	// lineAtFrame remembers, per frame index, the line last seen at that
	// exact frame depth -- so a breakpoint/step check only re-evaluates on
	// a genuine move to a new source line, not on every instruction still
	// within one. This is deliberately per-frame rather than a single
	// global "last line": without it, calling a function and returning
	// back to the same call-site line (e.g. the instruction that stores
	// the call's result) looks like "a new line" once naively compared
	// against whatever line was last seen deeper in the callee, and
	// spuriously re-fires a breakpoint or step that should only fire once
	// per genuine statement boundary. Entries for frames deeper than the
	// current one are dropped on every check() call, since a return makes
	// them stale (a later call reusing that depth must be seen as fresh).
	lineAtFrame map[int]int
	resumeCh    chan struct{}
	killCh      chan struct{}
	killOnce    sync.Once
	// onStop is called (without the session's lock held) whenever check()
	// is about to block -- e.g. to let a DAP server send a `stopped` event.
	// reason is "breakpoint" or "step".
	onStop func(reason string, line int)
}

// NewDebugSession creates a DebugSession in free-running mode (no
// breakpoints, never stops until SetBreakpoints/StepInto/etc. is called).
func NewDebugSession(onStop func(reason string, line int)) *DebugSession {
	return &DebugSession{
		breakpoints: make(map[int]bool),
		mode:        "run",
		lineAtFrame: make(map[int]int),
		resumeCh:    make(chan struct{}),
		killCh:      make(chan struct{}),
		onStop:      onStop,
	}
}

// SetBreakpoints replaces the full set of active breakpoint lines.
func (d *DebugSession) SetBreakpoints(lines []int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.breakpoints = make(map[int]bool, len(lines))
	for _, l := range lines {
		d.breakpoints[l] = true
	}
}

// Continue resumes free-running execution.
func (d *DebugSession) Continue() {
	d.mu.Lock()
	d.mode = "run"
	d.mu.Unlock()
	d.resumeCh <- struct{}{}
}

// StepInto resumes execution and stops at the next source line reached,
// regardless of call depth.
func (d *DebugSession) StepInto() {
	d.mu.Lock()
	d.mode = "step-into"
	d.mu.Unlock()
	d.resumeCh <- struct{}{}
}

// StepOver resumes execution and stops at the next source line reached at
// the same or a shallower frame depth than where execution last stopped --
// a call made from that line runs to completion without stopping inside it.
func (d *DebugSession) StepOver() {
	d.mu.Lock()
	d.mode = "step-over"
	d.stepFrameIndex = d.lastStoppedFrameIndex
	d.mu.Unlock()
	d.resumeCh <- struct{}{}
}

// StepOut resumes execution and stops at the next source line reached
// after returning from the frame execution last stopped in.
func (d *DebugSession) StepOut() {
	d.mu.Lock()
	d.mode = "step-out"
	d.stepFrameIndex = d.lastStoppedFrameIndex
	d.mu.Unlock()
	d.resumeCh <- struct{}{}
}

// StopOnEntry arms the session to stop at the very first instruction it
// checks, with reason "entry" -- for a DAP launch's stopOnEntry option. Must
// be called before the target VM's Run goroutine starts (it only sets a
// mode flag; it does not itself block).
func (d *DebugSession) StopOnEntry() {
	d.mu.Lock()
	d.mode = "entry"
	d.mu.Unlock()
}

// Kill unblocks any goroutine currently parked in check(), letting a
// disconnected/terminated debug session's target process actually exit
// instead of hanging forever on a resume that will never come. Safe to
// call multiple times or when nothing is paused.
func (d *DebugSession) Kill() {
	d.killOnce.Do(func() { close(d.killCh) })
}

// check is called once per instruction, from decodeAndCheck, immediately
// after vm.curLine/frame.ip are updated. It only evaluates breakpoint/step
// conditions on a genuine line transition (see lineAtFrame), so a breakpoint
// on a multi-instruction line triggers once per entry, not once per
// instruction. On a hit, it calls onStop and then blocks until Continue/
// StepInto/StepOver/StepOut sends on resumeCh, or Kill closes killCh.
func (d *DebugSession) check(vm *VM, frame *Frame) {
	d.mu.Lock()
	line := vm.curLine
	fi := vm.frameIndex

	for k := range d.lineAtFrame {
		if k > fi {
			delete(d.lineAtFrame, k)
		}
	}
	if seen, ok := d.lineAtFrame[fi]; ok && seen == line {
		d.mu.Unlock()
		return
	}
	d.lineAtFrame[fi] = line

	stop := false
	reason := "breakpoint"
	switch d.mode {
	case "entry":
		stop = true
		reason = "entry"
	case "step-into":
		stop = true
		reason = "step"
	case "step-over":
		if vm.frameIndex <= d.stepFrameIndex {
			stop = true
			reason = "step"
		}
	case "step-out":
		if vm.frameIndex < d.stepFrameIndex {
			stop = true
			reason = "step"
		}
	}
	if !stop && d.breakpoints[line] {
		stop = true
		reason = "breakpoint"
	}
	if !stop {
		d.mu.Unlock()
		return
	}
	d.mode = "run"
	d.lastStoppedFrameIndex = vm.frameIndex
	onStop := d.onStop
	d.mu.Unlock()

	if onStop != nil {
		onStop(reason, line)
	}

	select {
	case <-d.resumeCh:
	case <-d.killCh:
	}
}
