package vm

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

func parseAndCompile(t *testing.T, input string) *compiler.Bytecode {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	c := compiler.New()
	if err := c.Compile(program); err != nil {
		t.Fatalf("compile error: %s", err)
	}
	return c.Bytecode()
}

func runVM(t *testing.T, bc *compiler.Bytecode) string {
	t.Helper()
	vm := New(bc)
	if err := vm.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	return vm.LastPoppedStackElem().Inspect()
}

func TestLiteralInteger(t *testing.T) {
	bc := parseAndCompile(t, "42")
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42, got %s", result)
	}
}

func TestLiteralString(t *testing.T) {
	bc := parseAndCompile(t, `"hello"`)
	result := runVM(t, bc)
	if result != "hello" {
		t.Errorf("expected hello, got %s", result)
	}
}

func TestArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2", "3"},
		{"10 - 3", "7"},
		{"4 * 5", "20"},
		{"20 / 4", "5"},
		{"7 % 3", "1"},
		{"2 + 3 * 4", "14"},
		{"(2 + 3) * 4", "20"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bc := parseAndCompile(t, tt.input)
			result := runVM(t, bc)
			if result != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 == 1", "true"},
		{"1 != 2", "true"},
		{"1 < 2", "true"},
		{"2 > 1", "true"},
		{"2 <= 2", "true"},
		{"2 >= 2", "true"},
		{"1 == 2", "false"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bc := parseAndCompile(t, tt.input)
			result := runVM(t, bc)
			if result != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestStringConcat(t *testing.T) {
	bc := parseAndCompile(t, `"hello " ++ "world"`)
	result := runVM(t, bc)
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %s", result)
	}
}

func TestConstantFolding(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2 + 3 * 4", "14"},
		{"(2 + 3) * 4", "20"},
		{"10 - 2 - 3", "5"},
		{"1 + 2.5", "3.5"},
		{"7.5 - 7", "0.5"},
		{`"a" ++ "b" ++ "c"`, "abc"},
		{"1 < 2", "true"},
		{"2 >= 2", "true"},
		{`"abc" == "abc"`, "true"},
		{"!0", "false"},
		{`!""`, "false"},
		{"-(-5)", "5"},
		{"-(2 + 3)", "-5"},
		{"1 && 2", "2"},
		{"nil && 42", "nil"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bc := parseAndCompile(t, tt.input)
			result := runVM(t, bc)
			if result != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestVariable(t *testing.T) {
	input := "x: 42\nx"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42, got %s", result)
	}
}

func TestIfExpression(t *testing.T) {
	input := "if true\n    42\nelse\n    10"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42, got %s", result)
	}
}

func TestIfElseExpression(t *testing.T) {
	input := "if false\n    42\nelse\n    10"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "10" {
		t.Errorf("expected 10, got %s", result)
	}
}

func TestMatchExpression(t *testing.T) {
	input := "match 2\n    | 0 -> \"null\"\n    | 1 -> \"one\"\n    | _ -> \"other\""
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "other" {
		t.Errorf("expected other, got %s", result)
	}
}

func TestMatchGuard(t *testing.T) {
	input := "fn sign x\n    match x\n        | _ if x > 0 -> \"positive\"\n        | _ if x < 0 -> \"negative\"\n        | _ -> \"zero\"\n\nsign (-3)"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "negative" {
		t.Errorf("expected negative, got %s", result)
	}
}

func TestMatchGuardErrorFallsThrough(t *testing.T) {
	input := "match 1\n    | 1 if raise \"boom\" -> \"never\"\n    | _ -> \"fallback\""
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "fallback" {
		t.Errorf("expected fallback, got %s", result)
	}
}

func TestMatchGuardErrorThenMatchingCase(t *testing.T) {
	// The error guard must skip to the next case with the match value intact,
	// so a later literal case still matches.
	input := "match 1\n    | 1 if raise \"boom\" -> \"never\"\n    | 1 -> \"one\"\n    | _ -> \"fallback\""
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "one" {
		t.Errorf("expected one, got %s", result)
	}
}

func TestMatchGuardMultiPattern(t *testing.T) {
	input := "match 2\n    | 1 | 2 if true -> \"small\"\n    | _ -> \"big\""
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "small" {
		t.Errorf("expected small, got %s", result)
	}
}

func TestFunction(t *testing.T) {
	input := "fn double x\n    x * 2\n\ndouble 21"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42, got %s", result)
	}
}

func TestFunctionLocalOutlivesBuiltinCall(t *testing.T) {
	input := "fn one_var src\n    x: 42\n    len(src)\n    x\n\none_var \"OK\""
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42, got %s", result)
	}
}

func TestFunctionReturnPreservesCallerExpressionStack(t *testing.T) {
	input := "fn inner n\n    tmp: 2\n    tmp\n\nfn outer src\n    x: 40\n    x + inner(src)\n\nouter 1"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42, got %s", result)
	}
}

func TestRecursiveFunction(t *testing.T) {
	input := "fn fact n\n    match n\n        | 0 -> 1\n        | _ -> n * fact(n - 1)\n\nfact 5"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "120" {
		t.Errorf("expected 120, got %s", result)
	}
}

func TestUserFunctionMapLiteral(t *testing.T) {
	// A map literal inside a lambda executed via a callback builtin
	// (map/filter route through callUserFunction -> executeFrame, which
	// historically missed OpMap/OpStruct/OpSelect/OpHalt and failed with
	// "unknown opcode in user fn: 35").
	bc := parseAndCompile(t, "nums: [1, 2]\nr: map nums (fn x: {v: x * 10})\n(at r 1).v")
	if got := runVM(t, bc); got != "20" {
		t.Errorf("map literal in user fn: want 20, got %s", got)
	}
}

func TestVMDeepButLegalRecursion(t *testing.T) {
	input := "fn count n acc\n    if n <= 0\n        acc\n    else\n        count (n - 1) (acc + 1)\n\ncount 300 0"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "300" {
		t.Errorf("expected 300, got %s", result)
	}
}

func TestVMRecursionOverflowNoCrash(t *testing.T) {
	// Unbounded recursion must surface as a catchable E008 error object, not
	// a Go panic. The operand-space guard covers deep calls that would
	// exhaust the 2048-slot operand stack before the frame limit.
	for _, input := range []string{
		"fn f x\n    f x\n\nf 0",
		"fn count n acc\n    if n <= 0\n        acc\n    else\n        count (n - 1) (acc + 1)\n\ncount 100000 0",
	} {
		bc := parseAndCompile(t, input)
		v := New(bc)
		if err := v.Run(); err != nil {
			if e, ok := err.(*object.Error); ok && strings.Contains(e.Message, "E008") {
				continue
			}
			t.Errorf("%q: expected E008 error, got Run error %s", input, err)
			continue
		}
		top := v.LastPoppedStackElem()
		if _, isErr := top.(*object.Error); !isErr {
			t.Errorf("%q: expected E008 error object, got %s", input, top.Inspect())
		}
	}
}

func TestVMFrameGuardRejectsCall(t *testing.T) {
	// The frame-limit branch of the guard is not reachable through the
	// language (≥1-arg recursion exhausts the operand stack first), so drive
	// it directly: callFunction at frameIndex MaxCallDepth-1 must inject a
	// catchable E008 error object and restore a consistent stack/frame state.
	v := New(&compiler.Bytecode{})
	fn := &object.Closure{Fn: &object.CompiledFunction{Instructions: compiler.Instructions{byte(compiler.OpReturn)}}}
	v.stack[0] = fn
	v.stack[1] = &object.Integer{Value: 42}
	v.sp = 2
	v.frameIndex = object.MaxCallDepth - 1

	v.callFunction(1)

	if v.frameIndex != object.MaxCallDepth-1 {
		t.Errorf("frameIndex should be restored, got %d", v.frameIndex)
	}
	// basePtr = sp(2) - numArgs(1) = 1; error lands at basePtr-1, sp = basePtr
	if v.sp != 1 {
		t.Errorf("sp should be restored to basePtr (1), got %d", v.sp)
	}
	err, ok := v.stack[v.sp-1].(*object.Error)
	if !ok || !strings.Contains(err.Message, "E008") {
		t.Errorf("expected E008 error object at stack[sp-1], got %v", v.stack[v.sp-1])
	}
}

func TestVMTryCatchCatchesRecursionError(t *testing.T) {
	// Deep recursion (operand-guard path) must be catchable via try/catch.
	input := `fn f x
    f x

try
    f 0
catch e
    "caught: " ++ e`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if !strings.HasPrefix(result, "caught: ") || !strings.Contains(result, "E008") {
		t.Errorf("expected catch of E008, got %q", result)
	}
}

func TestVMTryCatchCatchesDeepArgRecursionError(t *testing.T) {
	// Multi-arg recursion exhausts the operand stack before the frame limit;
	// the operand-space guard must still surface a catchable E008.
	input := `fn count n acc
    if n <= 0
        acc
    else
        count (n - 1) (acc + 1)

try
    count 100000 0
catch e
    "caught: " ++ e`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if !strings.HasPrefix(result, "caught: ") || !strings.Contains(result, "E008") {
		t.Errorf("expected catch of E008, got %q", result)
	}
}

func TestVMContinuesAfterCaughtRecursionError(t *testing.T) {
	// After catching the recursion error, operand stack and frame state must
	// be consistent: subsequent statements evaluate correctly.
	input := `fn f x
    f x

try
    f 0
catch e
    "recovered"

n: 40
n + 2`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42 after caught recursion error, got %s", result)
	}
}

func TestVMTryAIValidCodeReturnsValue(t *testing.T) {
	// Regression: the AI-fix path compiled a skipFix jump that landed on the
	// catch-skip test, so a successful try_ai result was treated as the error
	// flag and the catch block ran. Healthy code must return the value.
	input := `x: try_ai
    6 * 7
catch e
    -1

x`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "42" {
		t.Errorf("expected 42 from healthy try_ai, got %s", result)
	}
}

func TestVMTryAIWithFixFallbackRunsCatch(t *testing.T) {
	// When the AI fix still returns an error, try_ai must run the catch block
	// instead of silently returning the error value.
	orig := object.TryAIEvalFn
	object.TryAIEvalFn = func(source string) object.Object {
		return &object.Error{Message: "still broken"}
	}
	defer func() { object.TryAIEvalFn = orig }()

	input := `x: try_ai
    "42" * 3
catch e
    "fixed: " ++ e

x`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "fixed: still broken" {
		t.Errorf("expected catch fallback from broken try_ai, got %s", result)
	}
}

func TestPipeline(t *testing.T) {
	input := "fn double x\n    x * 2\n\n42\n    > double"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "84" {
		t.Errorf("expected 84, got %s", result)
	}
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"true && true", "true"},
		{"true && false", "false"},
		{"false && true", "false"},
		{"true || false", "true"},
		{"false || true", "true"},
		{"false || false", "false"},
		{"1 && 2", "2"},
		{"nil && 42", "nil"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bc := parseAndCompile(t, tt.input)
			result := runVM(t, bc)
			if result != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestPrefixNot(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"!true", "false"},
		{"!false", "true"},
		{"!nil", "true"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bc := parseAndCompile(t, tt.input)
			result := runVM(t, bc)
			if result != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestMapLiteral(t *testing.T) {
	input := "m: {a: 1, b: 2}\nm"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result == "nil" || result == "" {
		t.Errorf("expected map output, got %s", result)
	}
	if !(strings.Contains(result, "a: 1") || strings.Contains(result, "b: 2")) {
		t.Errorf("expected map-like output, got %s", result)
	}
}

func TestVMZeroArityBuiltinWithArgs(t *testing.T) {
	bc := parseAndCompile(t, `ai_cost "reset"`)
	if got := runVM(t, bc); got != "cost metrics reset" {
		t.Errorf("ai_cost \"reset\": got %q", got)
	}
}

// TestVMNestedCallFromBuiltinResumesCaller guards against executeFrame
// unwinding the whole frame stack when a user function called from inside a
// builtin callback (e.g. sorted_by) returns via OpReturnValue. The caller
// (keyfn) and the closure must resume after inner's return, so keys are the
// strings "10!", "100!", "9!" instead of the raw integers 10, 100, 9.
func TestVMNestedCallFromBuiltinResumesCaller(t *testing.T) {
	input := `
fn inner v
    return v
fn keyfn v
    k: inner v
    k: (to_str k) ++ "!"
    k
l: [10, 9, 100]
sorted_by l (fn v: keyfn v)
`
	bc := parseAndCompile(t, input)
	if got := runVM(t, bc); got != "[10, 100, 9]" {
		t.Errorf("expected [10, 100, 9], got %s", got)
	}
}

// TestVMNestedReturnInWhileResumesCaller covers the sqlite ORDER BY case: a
// key function calling a helper that returns from inside a while loop, called
// through a closure from sorted_by. The intermediate frames must resume so the
// full (inverted) key is built; otherwise keys would be the raw integers and
// the rows would sort ascending as [[0, 80], [0, 90], [0, 95], [0, 100]].
func TestVMNestedReturnInWhileResumesCaller(t *testing.T) {
	input := `
fn eval_col row
    i: 0
    while i < (len row)
        if i == 1
            return (get row i)
        i: i + 1
    nil
fn order_key row
    key: ""
    key: key ++ (to_str (999999999999 - (eval_col row)))
    key
rows: [[0, 100], [0, 80], [0, 95], [0, 90]]
sorted_by rows (fn o: order_key o)
`
	bc := parseAndCompile(t, input)
	if got := runVM(t, bc); got != "[[0, 100], [0, 95], [0, 90], [0, 80]]" {
		t.Errorf("expected [[0, 100], [0, 95], [0, 90], [0, 80]], got %s", got)
	}
}

func TestVMWhileWithVarStatementIfBranchChunking(t *testing.T) {
	input := `fn esc text
    replace_all text "&" "&amp;"

fn send_chunked t
    n: len t
    start: 0
    out: []
    while start < n
        end: start + 3800
        if end > n
            end: n
        push out (len (substring t start end))
        start: end
    out

t: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
i: 0
acc: []
while i < 200
    push acc t
    i: i + 1
t: join acc ""
send_chunked t`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "[3800, 3800, 2800]" {
		t.Errorf("expected chunks [3800, 3800, 2800], got %q", result)
	}
}

// TestSpawnUserFunctionRunsParallel proves that >> with a user-defined
// function really executes in a background goroutine in the VM. Both spawned
// calls must reach a barrier builtin before either can finish; if >> fell back
// to a synchronous call (the pre-fix behaviour), the second spawn would never
// start and the test would time out.
func TestSpawnUserFunctionRunsParallel(t *testing.T) {
	var (
		mu       sync.Mutex
		arrived  int
		released bool
	)
	release := make(chan struct{})

	barrier := object.BuiltinInfo{
		Name: "test_barrier",
		Fn: func(args ...object.Object) object.Object {
			mu.Lock()
			arrived++
			if arrived >= 2 && !released {
				released = true
				close(release)
			}
			mu.Unlock()
			<-release
			return object.NILOBJ
		},
	}

	origLen := len(object.Builtins)
	object.Builtins = append(object.Builtins, barrier)
	defer func() { object.Builtins = object.Builtins[:origLen] }()

	input := "fn work x\n    test_barrier x\n    x * 2\n\na: 1\n    >> work\nb: 2\n    >> work\n\na + b"
	bc := parseAndCompile(t, input)

	done := make(chan string, 1)
	go func() {
		v := New(bc)
		if err := v.Run(); err != nil {
			done <- "ERROR: " + err.Error()
			return
		}
		done <- v.LastPoppedStackElem().Inspect()
	}()

	select {
	case <-release:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for both spawns to reach the barrier: >> did not run in parallel")
	}

	select {
	case res := <-done:
		if res != "6" {
			t.Errorf("expected 6, got %s", res)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for spawned results")
	}
}

func TestVMIfConsequenceVarStatementStackBalance(t *testing.T) {
	input := `fn f cond
    x: 0
    if cond
        x: 42
    x + 1

a: f true
b: f false
(to_str a) ++ "," ++ (to_str b)`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "43,1" {
		t.Errorf("expected 43,1, got %q", result)
	}
}

// TestConcurrentVMsMapClosure runs two VMs in parallel, each calling map with
// a user-defined closure. Before the per-closure executor refactor, both VMs
// shared a process-global callUserFn hook, which this test exposes as a data
// race under `go test -race`.
func TestConcurrentVMsMapClosure(t *testing.T) {
	input := "fn double x\n    x * 2\n\nmap ([1, 2, 3]) double"
	bc := parseAndCompile(t, input)

	workers := 8
	errs := make(chan string, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := New(bc)
			if err := v.Run(); err != nil {
				errs <- "vm error: " + err.Error()
				return
			}
			if got := v.LastPoppedStackElem().Inspect(); got != "[2, 4, 6]" {
				errs <- "expected [2, 4, 6], got " + got
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

// TestSpawnChannelSharedAcrossVMs proves a channel created in the parent VM is
// shared by reference with child VMs: a spawned function sends to it and the
// caller receives across the spawn boundary. Run under `go test -race` this
// exercises the concurrent channel object access.
func TestSpawnChannelSharedAcrossVMs(t *testing.T) {
	input := "fn put c\n    send c 42\n\nch: chan 1\nspawn put ch\nrecv ch"
	bc := parseAndCompile(t, input)

	v := New(bc)
	if err := v.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	if got := v.LastPoppedStackElem().Inspect(); got != "42" {
		t.Errorf("expected 42, got %q", got)
	}
}

func TestTestStatementPass(t *testing.T) {
	bc := parseAndCompile(t, "test \"ok\"\n    assert_eq (2 + 2) 4")
	v := New(bc)
	if err := v.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	if v.TestFailed {
		t.Error("TestFailed should be false for a passing test")
	}
}

func TestTestStatementFail(t *testing.T) {
	bc := parseAndCompile(t, "test \"bad\"\n    assert_eq (2 + 2) 5")
	v := New(bc)
	if err := v.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	if !v.TestFailed {
		t.Error("TestFailed should be set for a failing test")
	}
}

func TestTestStatementAbortsOnMiddleError(t *testing.T) {
	// The first assertion fails; the rest of the body is skipped by the probe,
	// matching the tree-walker's block short-circuit. The following test must
	// still run and pass.
	bc := parseAndCompile(t, "test \"a\"\n    assert_eq 1 2\n    print \"never\"\ntest \"b\"\n    assert_eq 2 2")
	v := New(bc)
	if err := v.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	if !v.TestFailed {
		t.Error("TestFailed should be set when a test body errors")
	}
}

func TestTestStatementDoesNotLeakFailure(t *testing.T) {
	// The second test passes; a single failing test must not fail it.
	bc := parseAndCompile(t, "test \"a\"\n    assert_eq 1 2\ntest \"b\"\n    assert_eq 3 3")
	v := New(bc)
	if err := v.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	if !v.TestFailed {
		t.Error("TestFailed should be set")
	}
	last := v.LastPoppedStackElem()
	if last.Type() == object.ERROR {
		t.Error("the passing test should leave a non-error top value")
	}
}

func TestTestSetupHookRunsBeforeTests(t *testing.T) {
	// Variables bound in a setup hook must be visible to the tests, matching
	// the tree-walker (shared environment).
	bc := parseAndCompile(t, "test setup\n    shared: 40\ntest \"uses setup\"\n    assert_eq shared 40")
	v := New(bc)
	if err := v.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	if v.TestFailed {
		t.Error("TestFailed should be false when setup and tests pass")
	}
}

func TestTestSetupHookFailureAborts(t *testing.T) {
	// A failing setup aborts the whole file: Run returns the setup error and
	// the following test never gets a chance to clear it.
	bc := parseAndCompile(t, "test setup\n    raise \"boom\"\ntest \"x\"\n    assert_eq 1 1")
	v := New(bc)
	err := v.Run()
	if err == nil {
		t.Fatal("expected Run to return the setup error")
	}
	if strings.Contains(err.Error(), "some tests failed") {
		t.Errorf("expected the setup error, got %s", err)
	}
}

func TestTestTeardownHookFailsFile(t *testing.T) {
	// A failing teardown fails the file even when all tests pass.
	bc := parseAndCompile(t, "test \"ok\"\n    assert_eq 1 1\ntest teardown\n    raise \"cleanup failed\"")
	v := New(bc)
	err := v.Run()
	if err == nil {
		t.Fatal("expected Run to return the teardown error")
	}
}
