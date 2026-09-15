// Command pipe-dap is a Debug Adapter Protocol (DAP) server for Pipe. It
// speaks the same Content-Length-framed JSON transport as cmd/pipe-lsp (see
// pkg/rpcframing) over stdio, and drives one pkg/vm.VM per debug session via
// pkg/vm's DebugSession breakpoint/step API.
//
// Stage 1 scope: a single top-level VM per launch, no spawn/concurrency
// debugging (spawned child VMs run un-paused -- see vm.DebugSession's doc
// comment), no conditional breakpoints or watch expressions.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/MachuraHarry/pipe/pkg/cache"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/rpcframing"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

// ---- wire envelope ----

type dapRequest struct {
	Seq       int             `json:"seq"`
	Type      string          `json:"type"`
	Command   string          `json:"command"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type dapResponse struct {
	Seq        int    `json:"seq"`
	Type       string `json:"type"`
	RequestSeq int    `json:"request_seq"`
	Success    bool   `json:"success"`
	Command    string `json:"command"`
	Message    string `json:"message,omitempty"`
	Body       any    `json:"body,omitempty"`
}

type dapEvent struct {
	Seq   int    `json:"seq"`
	Type  string `json:"type"`
	Event string `json:"event"`
	Body  any    `json:"body,omitempty"`
}

// frameWriter serializes writes to the client from any goroutine: the main
// request loop, the debuggee's own Run() goroutine (via DebugSession's
// onStop callback and PrintHook), and the goroutine that waits for Run() to
// finish -- all three send events independently.
type frameWriter struct {
	mu  sync.Mutex
	w   *bufio.Writer
	seq int64
}

func newFrameWriter(w io.Writer) *frameWriter {
	return &frameWriter{w: bufio.NewWriter(w)}
}

func (fw *frameWriter) nextSeq() int {
	return int(atomic.AddInt64(&fw.seq, 1))
}

func (fw *frameWriter) send(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fw.mu.Lock()
	defer fw.mu.Unlock()
	rpcframing.WriteMessage(fw.w, b)
	fw.w.Flush()
}

func (fw *frameWriter) respond(req dapRequest, success bool, message string, body any) {
	fw.send(dapResponse{
		Seq:        fw.nextSeq(),
		Type:       "response",
		RequestSeq: req.Seq,
		Success:    success,
		Command:    req.Command,
		Message:    message,
		Body:       body,
	})
}

func (fw *frameWriter) event(name string, body any) {
	fw.send(dapEvent{Seq: fw.nextSeq(), Type: "event", Event: name, Body: body})
}

// ---- session state ----

// session holds everything for one debug target. Stage 1 only ever debugs
// one program per pipe-dap process (matching cmd/pipe-lsp's one-process-
// per-editor-session convention), so this is process-global rather than
// keyed by a session id.
type session struct {
	fw *frameWriter

	mu          sync.Mutex
	programPath string
	machine     *vm.VM
	debug       *vm.DebugSession
	breakpoints []int
}

func newSession(fw *frameWriter) *session {
	return &session{fw: fw}
}

// ---- DAP argument shapes ----

type launchArgs struct {
	Program     string `json:"program"`
	StopOnEntry bool   `json:"stopOnEntry"`
}

type setBreakpointsArgs struct {
	Source struct {
		Path string `json:"path"`
	} `json:"source"`
	Breakpoints []struct {
		Line int `json:"line"`
	} `json:"breakpoints"`
}

type scopesArgs struct {
	FrameID int `json:"frameId"`
}

type variablesArgs struct {
	VariablesReference int `json:"variablesReference"`
}

// variablesReference encoding: Stage 1 has exactly one thread and one VM, so
// a simple bijection is enough. 0 is reserved (DAP's "no reference"); locals
// for frame i are encoded as 2*i+1 (odd), globals as a single fixed even
// non-zero value.
const globalsVarRef = 1_000_000

func localsVarRef(frameIdx int) int { return 2*frameIdx + 1 }

func frameIdxFromLocalsRef(ref int) (int, bool) {
	if ref <= 0 || ref%2 == 0 {
		return 0, false
	}
	return (ref - 1) / 2, true
}

// ---- request dispatch ----

func (s *session) handle(req dapRequest) {
	switch req.Command {
	case "initialize":
		s.fw.respond(req, true, "", map[string]any{
			"supportsConfigurationDoneRequest": true,
		})
		s.fw.event("initialized", nil)

	case "launch":
		s.handleLaunch(req)

	case "setBreakpoints":
		s.handleSetBreakpoints(req)

	case "configurationDone":
		s.fw.respond(req, true, "", nil)
		s.start()

	case "threads":
		s.fw.respond(req, true, "", map[string]any{
			"threads": []map[string]any{{"id": 1, "name": "main"}},
		})

	case "stackTrace":
		s.handleStackTrace(req)

	case "scopes":
		s.handleScopes(req)

	case "variables":
		s.handleVariables(req)

	// continue/next/stepIn/stepOut respond BEFORE unblocking the debuggee:
	// resuming can let it run to completion (or hit the next stop) almost
	// immediately, racing ahead to send its own stopped/exited/terminated
	// events before this response would otherwise go out.
	case "continue":
		s.fw.respond(req, true, "", map[string]any{"allThreadsContinued": true})
		s.withDebug(req, func(d *vm.DebugSession) { d.Continue() })

	case "next":
		s.fw.respond(req, true, "", nil)
		s.withDebug(req, func(d *vm.DebugSession) { d.StepOver() })

	case "stepIn":
		s.fw.respond(req, true, "", nil)
		s.withDebug(req, func(d *vm.DebugSession) { d.StepInto() })

	case "stepOut":
		s.fw.respond(req, true, "", nil)
		s.withDebug(req, func(d *vm.DebugSession) { d.StepOut() })

	case "disconnect", "terminate":
		// Respond before killing: Kill unblocks the debuggee's own
		// goroutine, which then races ahead to send its own exited/
		// terminated events, and a client expects this response first.
		s.fw.respond(req, true, "", nil)
		s.mu.Lock()
		d := s.debug
		s.mu.Unlock()
		if d != nil {
			d.Kill()
		}

	default:
		s.fw.respond(req, false, "unsupported command: "+req.Command, nil)
	}
}

func (s *session) withDebug(req dapRequest, fn func(d *vm.DebugSession)) {
	s.mu.Lock()
	d := s.debug
	s.mu.Unlock()
	if d == nil {
		return
	}
	fn(d)
}

func (s *session) handleLaunch(req dapRequest) {
	var args launchArgs
	if err := json.Unmarshal(req.Arguments, &args); err != nil || args.Program == "" {
		s.fw.respond(req, false, "launch requires a \"program\" path", nil)
		return
	}

	bc, _, err := cache.LoadOrCompile(args.Program)
	if err != nil {
		s.fw.respond(req, false, err.Error(), nil)
		return
	}

	machine := vm.New(bc)
	debug := vm.NewDebugSession(func(reason string, line int) {
		s.fw.event("stopped", map[string]any{
			"reason":            reason,
			"threadId":          1,
			"allThreadsStopped": true,
		})
		_ = line
	})
	if args.StopOnEntry {
		debug.StopOnEntry()
	}
	machine.AttachDebugSession(debug)

	s.mu.Lock()
	s.programPath = args.Program
	s.machine = machine
	s.debug = debug
	if len(s.breakpoints) > 0 {
		debug.SetBreakpoints(s.breakpoints)
	}
	s.mu.Unlock()

	s.fw.respond(req, true, "", nil)
}

func (s *session) handleSetBreakpoints(req dapRequest) {
	var args setBreakpointsArgs
	if err := json.Unmarshal(req.Arguments, &args); err != nil {
		s.fw.respond(req, false, "invalid setBreakpoints arguments", nil)
		return
	}
	lines := make([]int, 0, len(args.Breakpoints))
	verified := make([]map[string]any, 0, len(args.Breakpoints))
	for _, b := range args.Breakpoints {
		lines = append(lines, b.Line)
		verified = append(verified, map[string]any{"verified": true, "line": b.Line})
	}

	s.mu.Lock()
	s.breakpoints = lines
	d := s.debug
	s.mu.Unlock()
	if d != nil {
		d.SetBreakpoints(lines)
	}

	s.fw.respond(req, true, "", map[string]any{"breakpoints": verified})
}

// start begins running the compiled program on its own goroutine, once
// configurationDone has been received (so breakpoints set beforehand are
// already in place). Output is relayed via object.PrintHook, set once at
// process startup.
func (s *session) start() {
	s.mu.Lock()
	machine := s.machine
	path := s.programPath
	s.mu.Unlock()
	if machine == nil {
		return
	}

	go func() {
		err := machine.Run()
		exitCode := 0
		if err != nil {
			s.fw.event("output", map[string]any{
				"category": "stderr",
				"output":   fmt.Sprintf("%s: %v\n", path, err),
			})
			exitCode = 1
		}
		s.fw.event("exited", map[string]any{"exitCode": exitCode})
		s.fw.event("terminated", nil)
	}()
}

func (s *session) handleStackTrace(req dapRequest) {
	s.mu.Lock()
	machine := s.machine
	path := s.programPath
	s.mu.Unlock()
	if machine == nil {
		s.fw.respond(req, true, "", map[string]any{"stackFrames": []any{}, "totalFrames": 0})
		return
	}

	frames := machine.StackFrames()
	out := make([]map[string]any, 0, len(frames))
	for _, f := range frames {
		out = append(out, map[string]any{
			"id":     f.Index,
			"name":   f.Name,
			"line":   f.Line,
			"column": 1,
			"source": map[string]any{"path": path},
		})
	}
	s.fw.respond(req, true, "", map[string]any{"stackFrames": out, "totalFrames": len(out)})
}

func (s *session) handleScopes(req dapRequest) {
	var args scopesArgs
	_ = json.Unmarshal(req.Arguments, &args)
	s.fw.respond(req, true, "", map[string]any{
		"scopes": []map[string]any{
			{"name": "Locals", "variablesReference": localsVarRef(args.FrameID), "expensive": false},
			{"name": "Globals", "variablesReference": globalsVarRef, "expensive": false},
		},
	})
}

func (s *session) handleVariables(req dapRequest) {
	var args variablesArgs
	_ = json.Unmarshal(req.Arguments, &args)

	s.mu.Lock()
	machine := s.machine
	s.mu.Unlock()
	if machine == nil {
		s.fw.respond(req, true, "", map[string]any{"variables": []any{}})
		return
	}

	var vars []vm.Variable
	if args.VariablesReference == globalsVarRef {
		vars = machine.Globals()
	} else if fi, ok := frameIdxFromLocalsRef(args.VariablesReference); ok {
		vars = machine.Locals(fi)
	}

	out := make([]map[string]any, 0, len(vars))
	for _, v := range vars {
		out = append(out, map[string]any{"name": v.Name, "value": v.Value, "variablesReference": 0})
	}
	s.fw.respond(req, true, "", map[string]any{"variables": out})
}

// ---- transport loop ----

// serve reads Content-Length framed DAP requests from r and writes
// responses/events to w until r is exhausted. Split out from main so a test
// can drive it over an io.Pipe without a real stdio process.
func serve(r io.Reader, w io.Writer) {
	fw := newFrameWriter(w)
	s := newSession(fw)

	object.PrintHook = func(args ...object.Object) {
		var sb strings.Builder
		for i, a := range args {
			if i > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(a.Inspect())
		}
		sb.WriteByte('\n')
		fw.event("output", map[string]any{"category": "stdout", "output": sb.String()})
	}

	reader := bufio.NewReader(r)
	for {
		body, err := rpcframing.ReadMessage(reader)
		if err != nil {
			return
		}
		var req dapRequest
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}
		s.handle(req)
	}
}

func main() {
	serve(os.Stdin, os.Stdout)
}
