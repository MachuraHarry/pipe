package vm

import (
	"testing"
	"time"
)

type stopEvent struct {
	reason string
	line   int
}

// runWithDebugSession compiles src, attaches a fresh DebugSession to a new
// VM, and runs it on a goroutine. stops receives every onStop call in
// order; done receives Run's error exactly once when the program finishes
// (nil on success).
func runWithDebugSession(t *testing.T, src string) (d *DebugSession, stops chan stopEvent, done chan error) {
	t.Helper()
	bc := parseAndCompile(t, src)
	v := New(bc)
	stops = make(chan stopEvent, 16)
	d = NewDebugSession(func(reason string, line int) {
		stops <- stopEvent{reason: reason, line: line}
	})
	v.AttachDebugSession(d)
	done = make(chan error, 1)
	go func() { done <- v.Run() }()
	return d, stops, done
}

func expectStop(t *testing.T, stops chan stopEvent, wantReason string, wantLine int) {
	t.Helper()
	select {
	case s := <-stops:
		if s.reason != wantReason || s.line != wantLine {
			t.Fatalf("stop event: got %+v, want {%s %d}", s, wantReason, wantLine)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a stop event")
	}
}

func expectDone(t *testing.T, done chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("program did not finish cleanly: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the program to finish")
	}
}

func expectNoStop(t *testing.T, stops chan stopEvent) {
	t.Helper()
	select {
	case s := <-stops:
		t.Fatalf("expected no more stops, got %+v", s)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDebugSessionBreakpointHitsAndResumes(t *testing.T) {
	src := "x: 1\ny: 2\nz: x + y\nz"
	d, stops, done := runWithDebugSession(t, src)
	d.SetBreakpoints([]int{3})

	expectStop(t, stops, "breakpoint", 3)
	d.Continue()
	expectDone(t, done)
}

func TestDebugSessionBreakpointOnMultiInstructionLineFiresOnce(t *testing.T) {
	// A single line with several sub-expressions must only stop once per
	// entry into that line, not once per instruction still on it.
	src := "x: 1 + 2 + 3 + 4\nx"
	d, stops, done := runWithDebugSession(t, src)
	d.SetBreakpoints([]int{1})

	expectStop(t, stops, "breakpoint", 1)
	d.Continue()
	expectNoStop(t, stops)
	expectDone(t, done)
}

func TestDebugSessionStepIntoStopsAtNextLineRegardlessOfDepth(t *testing.T) {
	src := "fn inc n\n    n + 1\n\nx: inc(1)\nx"
	d, stops, done := runWithDebugSession(t, src)
	d.SetBreakpoints([]int{4})

	expectStop(t, stops, "breakpoint", 4)
	d.StepInto()
	// Steps into inc's body (line 2), not straight to line 5.
	expectStop(t, stops, "step", 2)
	d.Continue()
	expectDone(t, done)
}

func TestDebugSessionStepOverSkipsCalledFunctionBody(t *testing.T) {
	src := "fn inc n\n    n + 1\n\nx: inc(1)\ny: 2\ny"
	d, stops, done := runWithDebugSession(t, src)
	d.SetBreakpoints([]int{4})

	expectStop(t, stops, "breakpoint", 4)
	d.StepOver()
	// Must land on line 5 (the next line in the SAME frame), never line 2
	// (inc's body).
	expectStop(t, stops, "step", 5)
	d.Continue()
	expectDone(t, done)
}

func TestDebugSessionStepOutReturnsToCaller(t *testing.T) {
	src := "fn inc n\n    m: n + 1\n    m\n\nx: inc(1)\nx"
	d, stops, done := runWithDebugSession(t, src)
	d.SetBreakpoints([]int{2})

	expectStop(t, stops, "breakpoint", 2)
	d.StepOut()
	// Back in the caller, at the next line after the call.
	expectStop(t, stops, "step", 6)
	d.Continue()
	expectDone(t, done)
}

func TestDebugSessionKillUnblocksAPausedProgram(t *testing.T) {
	src := "x: 1\nx"
	d, stops, done := runWithDebugSession(t, src)
	d.SetBreakpoints([]int{1})

	expectStop(t, stops, "breakpoint", 1)
	d.Kill()
	expectDone(t, done)
}

func TestDebugSessionKillIsIdempotent(t *testing.T) {
	d := NewDebugSession(nil)
	d.Kill()
	d.Kill() // must not panic (closing an already-closed channel)
}
