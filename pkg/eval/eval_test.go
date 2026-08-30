package eval

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

func parseAndEval(t *testing.T, input string) object.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	ctx := NewEvalContext("<test>")
	env := object.NewEnvironment()
	return ctx.Eval(program, env)
}

func expectValue(t *testing.T, input string, expected string) {
	t.Helper()
	result := parseAndEval(t, input)
	if result == nil {
		t.Fatalf("%q: got nil", input)
	}
	got := result.Inspect()
	if got != expected {
		t.Errorf("%q: expected %q, got %q", input, expected, got)
	}
}

func expectError(t *testing.T, input string) {
	t.Helper()
	result := parseAndEval(t, input)
	if result == nil {
		t.Fatalf("%q: got nil, expected error", input)
	}
	if result.Type() != object.ERROR {
		t.Errorf("%q: expected error, got %s (%q)", input, result.Type(), result.Inspect())
	}
}

func TestEvalIntegerLiterals(t *testing.T) {
	expectValue(t, "42", "42")
	expectValue(t, "0", "0")
	expectValue(t, "-1", "-1")
	expectValue(t, "999999", "999999")
}

func TestEvalFloatLiterals(t *testing.T) {
	expectValue(t, "3.14", "3.14")
	expectValue(t, "0.5", "0.5")
	expectValue(t, "-2.7", "-2.7")
}

func TestEvalStringLiterals(t *testing.T) {
	expectValue(t, `"hello"`, "hello")
	expectValue(t, `""`, "")
	expectValue(t, `"Hallo Welt"`, "Hallo Welt")
}

func TestEvalBooleanAndNil(t *testing.T) {
	expectValue(t, "true", "true")
	expectValue(t, "false", "false")
	expectValue(t, "nil", "nil")
}

func TestEvalArithmetic(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1 + 2", "3"},
		{"10 - 3", "7"},
		{"4 * 5", "20"},
		{"20 / 4", "5"},
		{"7 % 3", "1"},
		{"2 ** 3", "8"},
		{"2 ** 10", "1024"},
		{"2 + 3 * 4", "14"},
		{"(2 + 3) * 4", "20"},
		{"10 / 3", "3"},
		{"3.0 + 2.0", "5"},
		{"3.5 * 2", "7"},
	}
	for _, tt := range tests {
		expectValue(t, tt.input, tt.expected)
	}
}

func TestEvalComparison(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1 == 1", "true"},
		{"1 != 2", "true"},
		{"1 < 2", "true"},
		{"2 > 1", "true"},
		{"2 <= 2", "true"},
		{"2 >= 2", "true"},
		{"1 == 2", "false"},
		{"3 < 2", "false"},
		{`"abc" == "abc"`, "true"},
		{`"abc" != "xyz"`, "true"},
	}
	for _, tt := range tests {
		expectValue(t, tt.input, tt.expected)
	}
}

func TestEvalLogical(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"true && true", "true"},
		{"true && false", "false"},
		{"false && true", "false"},
		{"true || false", "true"},
		{"false || true", "true"},
		{"false || false", "false"},
	}
	for _, tt := range tests {
		expectValue(t, tt.input, tt.expected)
	}
}

func TestEvalStringConcat(t *testing.T) {
	expectValue(t, `"hello " ++ "world"`, "hello world")
	expectValue(t, `"a" ++ "b" ++ "c"`, "abc")
}

func TestEvalPrefixOperators(t *testing.T) {
	expectValue(t, "!true", "false")
	expectValue(t, "!false", "true")
	expectValue(t, "!nil", "true")
	expectValue(t, "-5", "-5")
	expectValue(t, "-(-5)", "5")
}

func TestEvalVariables(t *testing.T) {
	input := "x: 42\nx"
	expectValue(t, input, "42")
}

func TestEvalReassignment(t *testing.T) {
	input := "x: 42\nx: x + 8\nx"
	expectValue(t, input, "50")
}

func TestEvalCompoundAssignment(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"x: 10\nx += 5\nx", "15"},
		{"x: 10\nx -= 3\nx", "7"},
		{"x: 10\nx *= 2\nx", "20"},
		{"x: 10\nx /= 4\nx", "2"},
	}
	for _, tt := range tests {
		expectValue(t, tt.input, tt.expected)
	}
}

func TestEvalIfExpression(t *testing.T) {
	expectValue(t, "if true\n    42\nelse\n    10", "42")
	expectValue(t, "if false\n    42\nelse\n    10", "10")
	expectValue(t, "if 1\n    \"yes\"\nelse\n    \"no\"", "yes")
	expectValue(t, "if nil\n    \"yes\"\nelse\n    \"no\"", "no")
	expectValue(t, "if false\n    42\nelse\n    10", "10")
}

func TestEvalIfElseIf(t *testing.T) {
	input := `if 1 > 2
    "a"
else if 2 > 1
    "b"
else
    "c"`
	expectValue(t, input, "b")
}

func TestEvalMatchExpression(t *testing.T) {
	input := "match 2\n    | 0 -> \"null\"\n    | 1 -> \"one\"\n    | _ -> \"other\""
	expectValue(t, input, "other")

	input2 := "match 1\n    | 0 -> \"null\"\n    | 1 -> \"one\"\n    | _ -> \"other\""
	expectValue(t, input2, "one")
}

func TestMatchBindingPattern(t *testing.T) {
	// Binding pattern binds the matched value
	input := `match 42
    | x: 42 -> x * 2
    | _ -> 0`
	expectValue(t, input, "84")
}

func TestMatchListDestructure(t *testing.T) {
	// List destructuring extracts elements
	input := `match [10, 20, 30]
    | [a, b, c] -> a + b + c
    | _ -> 0`
	expectValue(t, input, "60")
}

func TestMatchListDestructureWildcard(t *testing.T) {
	// Wildcard elements are skipped
	input := `match [10, 20, 30]
    | [a, _, c] -> a + c
    | _ -> 0`
	expectValue(t, input, "40")
}

func TestMatchListDestructureRest(t *testing.T) {
	// Rest captures remaining elements with postfix `..`
	input := `match [1, 2, 3, 4, 5]
    | [first, rest..] -> first + len(rest)
    | _ -> 0`
	expectValue(t, input, "5")
}

func TestMatchMapDestructure(t *testing.T) {
	// Map destructuring extracts values by key
	input := `match {name: "Alice", age: 30, city: "NYC"}
    | {name: n, age: a} -> n
    | _ -> "unknown"`
	expectValue(t, input, "Alice")
}

func TestMatchMapDestructureArithmetic(t *testing.T) {
	input := `match {x: 5, y: 10}
    | {x: a, y: b} -> a + b
    | _ -> 0`
	expectValue(t, input, "15")
}

func TestEvalFunctions(t *testing.T) {
	input := "fn double x\n    x * 2\n\ndouble 21"
	expectValue(t, input, "42")
}

func TestEvalMultiArgFunction(t *testing.T) {
	input := "fn add a b\n    a + b\n\nadd 3 4"
	expectValue(t, input, "7")
}

func TestEvalRecursiveFunction(t *testing.T) {
	input := "fn fact n\n    match n\n        | 0 -> 1\n        | _ -> n * fact(n - 1)\n\nfact 5"
	expectValue(t, input, "120")
}

func TestEvalClosure(t *testing.T) {
	input := "fn make_adder x\n    fn adder y\n        x + y\n\nadd5: make_adder 5\nadd5 10"
	expectValue(t, input, "15")
}

func TestEvalPipeline(t *testing.T) {
	input := "fn double x\n    x * 2\n\n42\n    > double"
	expectValue(t, input, "84")
}

func TestEvalPipelineWithArgs(t *testing.T) {
	input := "fn add a b\n    a + b\n\n10\n    > add 5"
	expectValue(t, input, "15")
}

func TestEvalLists(t *testing.T) {
	expectValue(t, "[1, 2, 3]", "[1, 2, 3]")
	expectValue(t, "[]", "[]")
}

func TestEvalListIndex(t *testing.T) {
	input := "nums: [10, 20, 30]\nnums[1]"
	expectValue(t, input, "20")
}

func TestEvalListSlice(t *testing.T) {
	input := "nums: [10, 20, 30, 40]\nnums[1..3]"
	expectValue(t, input, "[20, 30]")
}

func TestEvalMaps(t *testing.T) {
	input := "{a: 1, b: 2}"
	result := parseAndEval(t, input)
	if result == nil {
		t.Fatal("got nil")
	}
	m, ok := result.(*object.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", result)
	}
	if len(m.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(m.Pairs))
	}
	aVal, _ := m.Get("a")
	bVal, _ := m.Get("b")
	if aVal == nil || aVal.Inspect() != "1" {
		t.Errorf("a: expected 1, got %v", aVal)
	}
	if bVal == nil || bVal.Inspect() != "2" {
		t.Errorf("b: expected 2, got %v", bVal)
	}
	if len(m.Pairs) != 2 || m.Pairs[0].Key != "a" || m.Pairs[1].Key != "b" {
		t.Errorf("map should preserve declaration order, got %v", m.Keys())
	}
}

func TestEvalMapAccess(t *testing.T) {
	input := "m: {name: \"Pipe\"}\nget m \"name\""
	expectValue(t, input, "Pipe")
}

func TestEvalWhileLoop(t *testing.T) {
	input := "x: 0\nwhile x < 3\n    x: x + 1\nx"
	expectValue(t, input, "3")
}

func TestEvalBreak(t *testing.T) {
	input := "x: 0\nwhile true\n    x: x + 1\n    if x >= 5\n        break\nx"
	expectValue(t, input, "5")
}

func TestEvalContinue(t *testing.T) {
	input := "x: 0\nwhile x < 5\n    x: x + 1\n    if x % 2 == 1\n        continue\nx"
	expectValue(t, input, "5")
}

func TestEvalForIn(t *testing.T) {
	input := "sum: 0\nfor n in (range 1 4)\n    sum: sum + n\nsum"
	expectValue(t, input, "6")
}

func TestEvalCStyleFor(t *testing.T) {
	input := "sum: 0\nfor i: 0; i < 5; i: i + 1\n    sum: sum + i\nsum"
	expectValue(t, input, "10")
}

func TestEvalCStyleForContinue(t *testing.T) {
	input := "sum: 0\nfor i: 0; i < 5; i: i + 1\n    if i == 3\n        continue\n    sum: sum + i\nsum"
	expectValue(t, input, "7")
}

func TestEvalCStyleForInfinite(t *testing.T) {
	input := "k: 0\nfor ; ; k: k + 1\n    if k >= 5\n        break\nk"
	expectValue(t, input, "5")
}

func TestEvalNotKeyword(t *testing.T) {
	expectValue(t, "not true", "false")
	expectValue(t, "not false", "true")
	expectValue(t, "not (1 > 2)", "true")
}

func TestEvalMatchMultiPattern(t *testing.T) {
	input := "match 2\n    | 1 | 2 | 3 -> \"small\"\n    | _ -> \"big\""
	expectValue(t, input, "small")
	input2 := "match 9\n    | 1 | 2 | 3 -> \"small\"\n    | _ -> \"big\""
	expectValue(t, input2, "big")
}

func TestEvalMatchGuard(t *testing.T) {
	input := "fn sign x\n    match x\n        | _ if x > 0 -> \"positive\"\n        | _ if x < 0 -> \"negative\"\n        | _ -> \"zero\"\n\nsign 5"
	expectValue(t, input, "positive")
	expectValue(t, "fn sign x\n    match x\n        | _ if x > 0 -> \"positive\"\n        | _ if x < 0 -> \"negative\"\n        | _ -> \"zero\"\n\nsign (-3)", "negative")
	expectValue(t, "fn sign x\n    match x\n        | _ if x > 0 -> \"positive\"\n        | _ if x < 0 -> \"negative\"\n        | _ -> \"zero\"\n\nsign 0", "zero")
}

func TestEvalMatchGuardSpecificPattern(t *testing.T) {
	expectValue(t, "match 2\n    | 1 if true -> \"one\"\n    | 2 if true -> \"two\"\n    | _ -> \"other\"", "two")
	expectValue(t, "match 2\n    | 1 if false -> \"one\"\n    | 2 -> \"two\"\n    | _ -> \"other\"", "two")
}

func TestEvalMatchGuardMultiPattern(t *testing.T) {
	expectValue(t, "match 2\n    | 1 | 2 if true -> \"small\"\n    | _ -> \"big\"", "small")
	expectValue(t, "match 9\n    | 1 | 2 if true -> \"small\"\n    | _ -> \"big\"", "big")
}

func TestEvalMatchGuardErrorFallsThrough(t *testing.T) {
	input := "match 1\n    | 1 if raise \"boom\" -> \"never\"\n    | _ -> \"fallback\""
	expectValue(t, input, "fallback")
}

func TestEvalMatchGuardErrorThenMatchingCase(t *testing.T) {
	input := "match 1\n    | 1 if raise \"boom\" -> \"never\"\n    | 1 -> \"one\"\n    | _ -> \"fallback\""
	expectValue(t, input, "one")
}

func TestEvalReturn(t *testing.T) {
	input := "fn early x\n    if x < 0\n        return 0\n    x * 2\n\nearly 5"
	expectValue(t, input, "10")

	input2 := "fn early x\n    if x < 0\n        return 0\n    x * 2\n\nearly (-5)"
	expectValue(t, input2, "0")
}

func TestEvalDividByZero(t *testing.T) {
	expectError(t, "1 / 0")
}

func TestEvalUndefinedVar(t *testing.T) {
	expectError(t, "no_such_var")
}

func TestEvalBuiltins(t *testing.T) {
	expectValue(t, `len "hello"`, "5")
	expectValue(t, `len ([1, 2, 3])`, "3")
	expectValue(t, `abs (-5)`, "5")
	expectValue(t, `upper "hello"`, "HELLO")
	expectValue(t, `lower "HELLO"`, "hello")
	expectValue(t, `trim "  hi  "`, "hi")
	expectValue(t, `contains "hello" "ell"`, "true")
}

func TestEvalRange(t *testing.T) {
	expectValue(t, "range 3", "[0, 1, 2]")
	expectValue(t, "range 2 5", "[2, 3, 4]")
	expectValue(t, "range 0 10 3", "[0, 3, 6, 9]")
}

func TestEvalAnonymousFunction(t *testing.T) {
	input := "double: fn x\n    x * 2\n\ndouble 7"
	expectValue(t, input, "14")
}

func TestEvalEnum(t *testing.T) {
	input := "enum Color: Red, Green, Blue\nRed"
	expectValue(t, input, "0")
}

func TestEvalTryCatch(t *testing.T) {
	input := `try
    1 / 0
catch e
    "caught"`
	expectValue(t, input, "caught")
}

func TestEvalDefer(t *testing.T) {
	input := "x: 0\n\ndefer print \"deferred\"\nx: 42"
	expectValue(t, input, "42")
}

func TestEvalPipeResultType(t *testing.T) {
	expectValue(t, "Ok 42", "Ok(42)")
}

func TestEvalRecursionDepthGuard(t *testing.T) {
	// Unbounded recursion must not overflow the Go stack: the evaluator
	// rejects calls deeper than MaxCallDepth with a catchable E008 error
	// instead of crashing the process.
	input := `fn count n acc
    if n <= 0
        acc
    else
        count (n - 1) (acc + 1)

count 5000 0`
	expectError(t, input)

	input = "fn f x\n    if x\n        f x\n    else\n        f x\n\nf true"
	expectError(t, input)

	// Deep but legal recursion still works.
	expectValue(t, `fn count n acc
    if n <= 0
        acc
    else
        count (n - 1) (acc + 1)

count 500 0`, "500")

	// The error is catchable via try/catch, so scripts can recover.
	input = `fn count n acc
    if n <= 0
        acc
    else
        count (n - 1) (acc + 1)

try
    count 5000 0
catch e
    "caught"`
	expectValue(t, input, "caught")
}

func TestEvalStackTraceCap(t *testing.T) {
	// Deep errors must not produce a multi-thousand-line trace: it is capped
	// at maxTraceFrames entries followed by a "  ... (N more)" suffix.
	input := `fn count n acc
    if n <= 0
        acc
    else
        count (n - 1) (acc + 1)

count 5000 0`
	result := parseAndEval(t, input)
	if result == nil || result.Type() != object.ERROR {
		t.Fatalf("expected error, got %v", result)
	}
	msg := result.Inspect()
	if !strings.Contains(msg, "E008") {
		t.Errorf("expected E008, got %q", msg)
	}
	if frames := strings.Count(msg, "  in fn(count)"); frames != maxTraceFrames {
		t.Errorf("expected %d trace frames, got %d", maxTraceFrames, frames)
	}
	if !regexp.MustCompile(`\n  \.\.\. \(\d+ more\)$`).MatchString(msg) {
		t.Errorf("expected trace cap suffix, got %q", msg)
	}
}

func TestEvalErrorDepthMatchesSharedLimit(t *testing.T) {
	// The error message must report the same shared limit that both engines
	// enforce (object.MaxCallDepth), so the two engines cannot drift apart.
	result := parseAndEval(t, "fn f x\n    if x\n        f x\n    else\n        f x\n\nf true")
	if result == nil || result.Type() != object.ERROR {
		t.Fatalf("expected error, got %v", result)
	}
	if !strings.Contains(result.Inspect(), fmt.Sprintf("E008: call stack depth exceeded (%d)", object.MaxCallDepth)) {
		t.Errorf("expected shared limit %d in message, got %q", object.MaxCallDepth, result.Inspect())
	}
}

func TestEvalTailCallOptimization(t *testing.T) {
	// A direct self-call as the last statement of the body runs in constant
	// stack space via the TCO loop, well beyond the recursion depth guard.
	input := `fn countdown n
    if n <= 0
        return n
    countdown (n - 1)

countdown 5000`
	expectValue(t, input, "0")
}

func TestEvalBuiltinsTypeCheck(t *testing.T) {
	expectValue(t, "is_num 42", "true")
	expectValue(t, `is_str "hello"`, "true")
	expectValue(t, `is_list ([1, 2])`, "true")
	expectValue(t, `is_map ({a: 1})`, "true")
	expectValue(t, "is_nil nil", "true")
	expectValue(t, "is_num true", "false")
}

func TestEvalParallelPipeline(t *testing.T) {
	input := "fn double x\n    x * 2\n\n10\n    >> double\n    > to_num"
	expectValue(t, input, "20")
}

func TestEvalParallelPipelineVar(t *testing.T) {
	input := "fn triple x\n    x * 3\n\nresult: 5\n    >> triple\n\nresult + 10"
	expectValue(t, input, "25")
}

func TestTryAIFixBlockedByProfile(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"(no_such_var + 1)"}}]}`))
	}))
	defer srv.Close()

	prevCfg := ai.ActiveConfig
	ai.ActiveConfig = ai.Config{Provider: "openai", Model: "gpt-4o-mini", APIHost: srv.URL, Timeout: time.Second}
	defer func() { ai.ActiveConfig = prevCfg }()
	ai.SetAPIKey("openai", "test-key")

	prev := object.ActiveProfile.Load()
	blocked := object.NewSandboxProfile("no_ai")
	blocked.AI = false
	object.ActiveProfile.Store(blocked)
	defer object.ActiveProfile.Store(prev)

	result := parseAndEval(t, "try_ai\n    no_such_var\ncatch e\n    \"caught\"")
	if result == nil || result.Inspect() != "caught" {
		t.Fatalf("expected catch result, got %v", result)
	}
	if got := hits.Load(); got != 0 {
		t.Fatalf("expected no AI requests with ai:false, got %d", got)
	}
}

func TestTryAIRing2InheritsLimits(t *testing.T) {
	prev := object.NewSandboxProfile("caller")
	prev.Budget = 0.01
	prev.MaxToolCalls = 3
	prev.Timeout = 7
	prev.AuditLog = true

	ring2 := newTryAIRing2Profile(prev)
	if ring2.Name != "try_ai_ring2" {
		t.Errorf("name = %q, want try_ai_ring2", ring2.Name)
	}
	if !ring2.AI {
		t.Error("ring2 must allow the fix expression to use AI")
	}
	if ring2.FSAccess != object.FSNone || ring2.Network || ring2.Exec {
		t.Error("ring2 must be sandboxed (FS none, no network, no exec)")
	}
	if ring2.Budget != 0.01 {
		t.Errorf("budget = %v, want 0.01", ring2.Budget)
	}
	if ring2.MaxToolCalls != 3 {
		t.Errorf("max tool calls = %d, want 3", ring2.MaxToolCalls)
	}
	if ring2.Timeout != 7 {
		t.Errorf("timeout = %d, want 7", ring2.Timeout)
	}
	if !ring2.AuditLog {
		t.Error("audit log not inherited")
	}
}

func TestProcessStartWait(t *testing.T) {
	input := `h: proc_start "echo -n hello"
result: proc_wait h
result.output`
	expectValue(t, input, "hello")
}

func TestProcessRunning(t *testing.T) {
	input := `h: proc_start "sleep 10"
r: proc_running h
proc_kill h
to_str(r)`
	expectValue(t, input, "true")
}

func TestProcessKill(t *testing.T) {
	input := `h: proc_start "sleep 10"
proc_kill h
result: proc_wait h
to_str(result.status != 0)`
	expectValue(t, input, "true")
}

// TestCallUserFunctionDispatchesBuiltinIdentifierValue is the round-9 audit
// regression test: a bare builtin identifier evaluated as a value (not
// called) — e.g. "read_file" passed straight to ai_tool, the pattern every
// redteam*.pipe script and several examples use — must be dispatchable by
// object.CallUserFunction, the same path executeTool uses to invoke a
// registered ai_tool. Before the CallableBuiltin fix (pkg/object/object.go),
// this returned "not callable: BUILTIN": CallUserFunction's type switch only
// recognized *object.BuiltinInfo (what the VM's OpGetBuiltin produces), not
// this package's own *eval.Builtin wrapper that evalIdentifier returns for a
// non-zero-arity builtin referenced as a value. The VM was never affected;
// only the tree-walker's own builtin-as-value path was.
func TestCallUserFunctionDispatchesBuiltinIdentifierValue(t *testing.T) {
	fnVal := parseAndEval(t, "upper")
	if fnVal == nil || fnVal.Type() == object.ERROR {
		t.Fatalf("evaluating 'upper' as a value: got %v", fnVal)
	}
	if _, ok := fnVal.(*Builtin); !ok {
		t.Fatalf("expected a bare builtin identifier to evaluate to *eval.Builtin, got %T", fnVal)
	}

	result := object.CallUserFunction(fnVal, &object.String{Value: "hi"})
	if result == nil || result.Type() == object.ERROR {
		t.Fatalf("CallUserFunction on a builtin identifier value: got %v", result)
	}
	s, ok := result.(*object.String)
	if !ok || s.Value != "HI" {
		t.Fatalf(`CallUserFunction(upper, "hi") = %v, want "HI"`, result)
	}
}
