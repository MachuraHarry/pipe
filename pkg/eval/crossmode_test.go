package eval

import (
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

func parseProgram(t *testing.T, input string) *ast.Program {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	return program
}

func evalResult(program *ast.Program) object.Object {
	// Empty source file: keeps error messages comparable with the VM, which
	// has no file context in these unit tests.
	ctx := NewEvalContext("")
	env := object.NewEnvironment()
	return ctx.Eval(program, env)
}

func vmResult(t *testing.T, program *ast.Program) object.Object {
	t.Helper()
	c := compiler.New()
	if err := c.Compile(program); err != nil {
		t.Fatalf("compile error: %s", err)
	}
	v := vm.New(c.Bytecode())
	if err := v.Run(); err != nil {
		// An uncaught error halts the VM and is returned from Run — the VM
		// equivalent of the tree-walker's error result. Surface it as the
		// error object so both engines are compared as values.
		if e, ok := err.(*object.Error); ok {
			return e
		}
		t.Fatalf("vm error: %s", err)
	}
	return v.LastPoppedStackElem()
}

func assertBothEqual(t *testing.T, input string, expected string) {
	t.Helper()
	program := parseProgram(t, input)

	evalResult := evalResult(program)
	if evalResult == nil {
		t.Fatalf("eval %q: got nil", input)
	}
	evalGot := evalResult.Inspect()

	vmResult := vmResult(t, program)
	vmGot := ""
	if vmResult != nil {
		vmGot = vmResult.Inspect()
	}

	if evalGot != expected {
		t.Errorf("eval %q: expected %q, got %q", input, expected, evalGot)
	}
	if vmGot != expected {
		t.Errorf("vm   %q: expected %q, got %q", input, expected, vmGot)
	}
}

func assertBothError(t *testing.T, input string) {
	t.Helper()
	program := parseProgram(t, input)

	evalR := evalResult(program)
	vmR := vmResult(t, program)

	if evalR == nil || evalR.Type() != object.ERROR {
		t.Errorf("eval %q: expected error, got %v", input, evalR)
	}
	if vmR == nil || vmR.Type() != object.ERROR {
		t.Errorf("vm   %q: expected error, got %v", input, vmR)
	}
}

func assertBothErrorContains(t *testing.T, input string, needle string) {
	t.Helper()
	program := parseProgram(t, input)

	evalR := evalResult(program)
	vmR := vmResult(t, program)

	if evalR == nil || evalR.Type() != object.ERROR {
		t.Errorf("eval %q: expected error, got %v", input, evalR)
		return
	}
	if vmR == nil || vmR.Type() != object.ERROR {
		t.Errorf("vm   %q: expected error, got %v", input, vmR)
		return
	}

	evalMsg := evalR.Inspect()
	vmMsg := vmR.Inspect()
	if !strings.Contains(evalMsg, needle) {
		t.Errorf("eval %q: error message %q does not contain %q", input, evalMsg, needle)
	}
	if !strings.Contains(vmMsg, needle) {
		t.Errorf("vm   %q: error message %q does not contain %q", input, vmMsg, needle)
	}
}

func TestCrossLiterals(t *testing.T) {
	assertBothEqual(t, "42", "42")
	assertBothEqual(t, "0", "0")
	assertBothEqual(t, "-1", "-1")
	assertBothEqual(t, "3.14", "3.14")
	assertBothEqual(t, `"hello"`, "hello")
	assertBothEqual(t, "true", "true")
	assertBothEqual(t, "false", "false")
	assertBothEqual(t, "nil", "nil")
}

func TestCrossArithmetic(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1 + 2", "3"},
		{"10 - 3", "7"},
		{"4 * 5", "20"},
		{"20 / 4", "5"},
		{"7 % 3", "1"},
		{"2 + 3 * 4", "14"},
		{"(2 + 3) * 4", "20"},
		{"3.0 + 2.0", "5"},
	}
	for _, tt := range tests {
		assertBothEqual(t, tt.input, tt.expected)
	}
}

func TestCrossComparison(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1 == 1", "true"},
		{"1 != 2", "true"},
		{"1 < 2", "true"},
		{"2 > 1", "true"},
		{"2 <= 2", "true"},
		{"2 >= 2", "true"},
		{"1 == 2", "false"},
		{`"abc" == "abc"`, "true"},
		{`"abc" != "xyz"`, "true"},
	}
	for _, tt := range tests {
		assertBothEqual(t, tt.input, tt.expected)
	}
}

func TestCrossLogical(t *testing.T) {
	tests := []struct{ input, expected string }{
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
		assertBothEqual(t, tt.input, tt.expected)
	}
}

func TestCrossStringConcat(t *testing.T) {
	assertBothEqual(t, `"hello " ++ "world"`, "hello world")
	assertBothEqual(t, `"a" ++ "b" ++ "c"`, "abc")
}

func TestCrossPrefix(t *testing.T) {
	assertBothEqual(t, "!true", "false")
	assertBothEqual(t, "!false", "true")
	assertBothEqual(t, "!nil", "true")
	assertBothEqual(t, "-5", "-5")
}

func TestCrossVariables(t *testing.T) {
	assertBothEqual(t, "x: 42\nx", "42")
	assertBothEqual(t, "x: 42\nx: x + 8\nx", "50")
}

func TestCrossIfExpression(t *testing.T) {
	assertBothEqual(t, "if true\n    42\nelse\n    10", "42")
	assertBothEqual(t, "if false\n    42\nelse\n    10", "10")
	assertBothEqual(t, "if 1\n    \"yes\"\nelse\n    \"no\"", "yes")
	assertBothEqual(t, "if nil\n    \"yes\"\nelse\n    \"no\"", "no")
}

func TestCrossMatchExpression(t *testing.T) {
	assertBothEqual(t, "match 2\n    | 0 -> \"null\"\n    | 1 -> \"one\"\n    | _ -> \"other\"", "other")
	assertBothEqual(t, "match 1\n    | 0 -> \"null\"\n    | 1 -> \"one\"\n    | _ -> \"other\"", "one")
}

func TestCrossFunctions(t *testing.T) {
	assertBothEqual(t, "fn double x\n    x * 2\n\ndouble 21", "42")
	assertBothEqual(t, "fn add a b\n    a + b\n\nadd 3 4", "7")
}

func TestCrossRecursiveFunction(t *testing.T) {
	assertBothEqual(t, "fn fact n\n    match n\n        | 0 -> 1\n        | _ -> n * fact(n - 1)\n\nfact 5", "120")
}

func TestCrossPipeline(t *testing.T) {
	assertBothEqual(t, "fn double x\n    x * 2\n\n42\n    > double", "84")
}

func TestCrossLists(t *testing.T) {
	assertBothEqual(t, "[1, 2, 3]", "[1, 2, 3]")
	assertBothEqual(t, "[]", "[]")
	assertBothEqual(t, "nums: [10, 20, 30]\nnums[1]", "20")
}

func TestCrossMaps(t *testing.T) {
	input := "m: {a: 1, b: 2}\nm"
	program := parseProgram(t, input)

	evalR := evalResult(program)
	vmR := vmResult(t, program)

	if evalR == nil {
		t.Fatal("eval: got nil")
	}
	if _, ok := evalR.(*object.Map); !ok {
		t.Errorf("eval: expected Map, got %T", evalR)
	}
	if vmR == nil {
		t.Fatal("vm: got nil")
	}
	if _, ok := vmR.(*object.Map); !ok {
		t.Errorf("vm: expected Map, got %T", vmR)
	}
}

func TestCrossWhileLoop(t *testing.T) {
	assertBothEqual(t, "x: 0\nwhile x < 3\n    x: x + 1\nx", "3")
}

func TestCrossBuiltins(t *testing.T) {
	assertBothEqual(t, `len "hello"`, "5")
	assertBothEqual(t, "abs (-5)", "5")
	assertBothEqual(t, `upper "hello"`, "HELLO")
	assertBothEqual(t, `lower "HELLO"`, "hello")
	assertBothEqual(t, `trim "  hi  "`, "hi")
	assertBothEqual(t, `contains "hello" "ell"`, "true")
	assertBothEqual(t, `ai_cost "reset"`, "cost metrics reset")
}

func TestCrossRange(t *testing.T) {
	assertBothEqual(t, "range 3", "[0, 1, 2]")
	assertBothEqual(t, "range 2 5", "[2, 3, 4]")
}

func TestCrossEnum(t *testing.T) {
	assertBothEqual(t, "enum Color: Red, Green, Blue\nRed", "0")
}

func TestCrossPrefixNot(t *testing.T) {
	assertBothEqual(t, "not true", "false")
	assertBothEqual(t, "not false", "true")
	assertBothEqual(t, "not (1 > 2)", "true")
}

func TestCrossReturn(t *testing.T) {
	assertBothEqual(t, "fn early x\n    if x < 0\n        return 0\n    x * 2\n\nearly 5", "10")
	assertBothEqual(t, "fn early x\n    if x < 0\n        return 0\n    x * 2\n\nearly (-5)", "0")
}

func TestCrossTypeChecks(t *testing.T) {
	assertBothEqual(t, "is_num 42", "true")
	assertBothEqual(t, `is_str "hello"`, "true")
	assertBothEqual(t, "is_nil nil", "true")
	assertBothEqual(t, "is_num true", "false")
}

func TestCrossParallelPipeline(t *testing.T) {
	assertBothEqual(t, "fn double x\n    x * 2\n\n10\n    >> double\n    > to_num", "20")
}

func TestCrossGoUserFunction(t *testing.T) {
	// `go` with a user function is fire-and-forget: it must return nil in
	// both the tree-walker and the VM instead of erroring.
	assertBothEqual(t, "fn double x\n    x * 2\n\ngo double 21", "nil")
}

func TestCrossSpawnAwait(t *testing.T) {
	// spawn returns a Future; await blocks until it resolves and returns the
	// value in both engines.
	assertBothEqual(t, "fn double x\n    x * 2\n\nf: spawn double 21\nawait f", "42")
}

func TestCrossAwaitResolvedValue(t *testing.T) {
	// await on a plain (non-Future) value is a no-op.
	assertBothEqual(t, "await 7", "7")
}

func TestCrossSpawnBuiltin(t *testing.T) {
	// spawn can launch a builtin in the background and await its result.
	assertBothEqual(t, "f: spawn upper \"hello\"\nawait f", "HELLO")
}

func TestCrossAwaitTimeout(t *testing.T) {
	// A Future that never resolves within the timeout must error.
	assertBothErrorContains(t, "fn slow x\n    sleep 500\n    x\n\nf: spawn slow 1\nawait f 20", "timed out")
}

func TestCrossChanBuffered(t *testing.T) {
	// A buffered channel sends and receives identically in both engines.
	assertBothEqual(t, "c: chan 2\nsend c 10\nsend c 20\nrecv c + recv c", "30")
}

func TestCrossChanLenCap(t *testing.T) {
	assertBothEqual(t, "c: chan 5\nsend c 1\nchan_len c", "1")
	assertBothEqual(t, "c: chan 5\nchan_cap c", "5")
}

func TestCrossChanTryRecvEmpty(t *testing.T) {
	assertBothEqual(t, "c: chan 0\ntry_recv c", "nil")
}

func TestCrossChanTrySend(t *testing.T) {
	assertBothEqual(t, "c: chan 1\nsend c 1\ntry_send c 2", "false")
}

func TestCrossChanClose(t *testing.T) {
	assertBothEqual(t, "c: chan 1\nsend c 7\nclose c\nrecv c", "7")
	assertBothEqual(t, "c: chan 0\nclose c\nrecv c", "nil")
}

func TestCrossMutexTryLock(t *testing.T) {
	assertBothEqual(t, "m: mutex\ntry_lock m", "true")
}

func TestCrossSemaphore(t *testing.T) {
	assertBothEqual(t, "s: semaphore 2\ntry_acquire s", "true")
}

func TestCrossChannelTypes(t *testing.T) {
	assertBothEqual(t, "type_of (chan 1)", "CHANNEL")
	assertBothEqual(t, "type_of mutex", "MUTEX")
	assertBothEqual(t, "type_of (semaphore 3)", "SEMAPHORE")
}

func TestCrossChannelAcrossSpawn(t *testing.T) {
	// A channel is shared by reference across the spawn boundary: a spawned
	// producer sends to it and the caller receives the value.
	input := "fn produce c\n    send c 42\n\nch: chan 0\nspawn produce ch\nrecv ch"
	assertBothEqual(t, input, "42")
}

func TestCrossAnonymousFunction(t *testing.T) {
	assertBothEqual(t, "double: fn x\n    x * 2\n\ndouble 7", "14")
}

func TestCrossResultType(t *testing.T) {
	assertBothEqual(t, "Ok 42", "Ok(42)")
}

func TestCrossMapAccess(t *testing.T) {
	assertBothEqual(t, "m: {name: \"Pipe\"}\nget m \"name\"", "Pipe")
}

func TestCrossErrorDivisionByZero(t *testing.T) {
	assertBothErrorContains(t, "1 / 0", "E003")
}

func TestCrossErrorNotAFunction(t *testing.T) {
	assertBothErrorContains(t, "42 1", "E004")
}

func TestCrossErrorCannotIndex(t *testing.T) {
	assertBothError(t, "42[0]")
}

func TestCrossErrorTypeMismatch(t *testing.T) {
	assertBothError(t, `"a" + 1`)
}

func TestCrossErrorUnsupportedOperator(t *testing.T) {
	assertBothError(t, `"a" * "b"`)
}

func TestCrossClosure(t *testing.T) {
	assertBothEqual(t, "fn make_adder x\n    fn adder y\n        x + y\n\nadd5: make_adder 5\nadd5 10", "15")
}

func TestCrossTryCatch(t *testing.T) {
	assertBothEqual(t, "try\n    1 / 0\ncatch e\n    \"caught\"", "caught")
}

func TestCrossTryCatchBindsErrorMessageString(t *testing.T) {
	// `catch e` binds the error message string, so `++` on it must work in
	// both execution modes (docs: "err is the error message string").
	assertBothEqual(t, "try\n    1 / 0\ncatch e\n    \"caught: \" ++ e", "caught: E003: division by zero")
	assertBothEqual(t, "try\n    1 / 0\ncatch e\n    \"len=\" ++ (to_str (len e))", "len=22")
}

func TestCrossTryCatchAbortsBlockAtFirstError(t *testing.T) {
	// The try block must abort at the first error (docs: "The try block aborts
	// immediately"); the catch param is still a message string.
	assertBothEqual(t, "try\n    x: 1 / 0\n    x: 2\ncatch e\n    \"caught: \" ++ e", "caught: E003: division by zero")
}

func TestCrossTryReturnsBlockValue(t *testing.T) {
	assertBothEqual(t, "try\n    5\ncatch e\n    \"no\"", "5")
	assertBothEqual(t, "try\n    x: 5\ncatch e\n    \"no\"", "5")
}

func TestCrossConcatStrictTypeCheck(t *testing.T) {
	// `++` requires strings (or bytes); mixed types are a type mismatch in
	// both execution modes.
	assertBothErrorContains(t, `"a" ++ 42`, "E002")
	assertBothErrorContains(t, `42 ++ "a"`, "E002")
	assertBothErrorContains(t, `"a" ++ 1.5`, "E002")
	assertBothErrorContains(t, `"a" ++ true`, "E002")
	assertBothErrorContains(t, `[1, 2] ++ "a"`, "E002")
}

func TestCrossPipelineWithArgs(t *testing.T) {
	assertBothEqual(t, "fn add a b\n    a + b\n\n10\n    > add 5", "15")
}

func TestCrossListSlice(t *testing.T) {
	assertBothEqual(t, "nums: [10, 20, 30, 40]\nnums[1..3]", "[20, 30]")
}

func TestCrossCompoundAssignment(t *testing.T) {
	assertBothEqual(t, "x: 10\nx += 5\nx", "15")
	assertBothEqual(t, "x: 10\nx -= 3\nx", "7")
}

func TestCrossContinueBreak(t *testing.T) {
	assertBothEqual(t, "x: 0\nwhile true\n    x: x + 1\n    if x >= 5\n        break\nx", "5")
}

func TestCrossForIn(t *testing.T) {
	assertBothEqual(t, "sum: 0\nfor n in (range 1 4)\n    sum: sum + n\nsum", "6")
}

func TestCrossMatchMultiPattern(t *testing.T) {
	assertBothEqual(t, "match 2\n    | 1 | 2 | 3 -> \"small\"\n    | _ -> \"big\"", "small")
	assertBothEqual(t, "match 9\n    | 1 | 2 | 3 -> \"small\"\n    | _ -> \"big\"", "big")
}

func TestCrossParallelPipelineVar(t *testing.T) {
	assertBothEqual(t, "fn triple x\n    x * 3\n\nresult: 5\n    >> triple\n\nresult + 10", "25")
}

func TestCrossStructDefineAndCreate(t *testing.T) {
	input := "struct Point\n    x\n    y\n\np: Point 10 20\np.x"
	assertBothEqual(t, input, "10")
}

func TestCrossStructMultipleFields(t *testing.T) {
	input := "struct Point\n    x\n    y\n\np: Point 1 2\np.y"
	assertBothEqual(t, input, "2")
}

func TestCrossStructWithDefaults(t *testing.T) {
	input := "struct Point\n    x: 0\n    y: 0\n\np: Point 5 3\np.x + p.y"
	assertBothEqual(t, input, "8")
}

func TestCrossTestHooks(t *testing.T) {
	// setup binds a variable the tests can read (shared environment), and the
	// hooks themselves are silent — the file result is the trailing value.
	assertBothEqual(t, "test setup\n    shared: 40\ntest \"uses setup\"\n    assert_eq shared 40\nshared", "40")
	assertBothEqual(t, "test setup\n    x: 1\ntest teardown\n    y: 2\n1 + 1", "2")
}

func TestCrossTestSetupHookError(t *testing.T) {
	// A failing setup aborts the file: the tests after it never run and the
	// program evaluates to the setup error in both engines.
	assertBothErrorContains(t, "test setup\n    raise \"boom\"\ntest \"never\"\n    assert_eq 1 2", "boom")
}

func TestCrossTestTeardownHookError(t *testing.T) {
	assertBothErrorContains(t, "test \"ok\"\n    assert_eq 1 1\ntest teardown\n    raise \"cleanup\"", "cleanup")
}

func TestCrossTestHookShortCircuit(t *testing.T) {
	// The first hook statement errors; the rest of the hook body is skipped
	// in both engines (block short-circuit).
	assertBothErrorContains(t, "test setup\n    raise \"first\"\n    print \"never\"", "first")
}

func TestCrossStructInline(t *testing.T) {
	input := "struct Point: x, y\n\np: Point 3 4\np.x"
	assertBothEqual(t, input, "3")
}

func TestCrossStructDotOnNonStruct(t *testing.T) {
	assertBothError(t, "struct Point: x\np: Point 1\n\"hello\".x")
}

func TestCrossStructUndefinedField(t *testing.T) {
	program := parseProgram(t, "struct Point: x\n\np: Point 1\np.y")
	evalR := evalResult(program)
	if evalR == nil || evalR.Type() != object.ERROR {
		t.Errorf("eval: expected error for unknown field, got %v", evalR)
	}
}

func TestCrossSecureRandom(t *testing.T) {
	program := parseProgram(t, "secure_random 16")
	evalR := evalResult(program)
	vmR := vmResult(t, program)

	if evalR == nil || evalR.Type() != object.STRING {
		t.Errorf("eval secure_random: expected string, got %v", evalR)
	}
	if vmR == nil || vmR.Type() != object.STRING {
		t.Errorf("vm secure_random: expected string, got %v", vmR)
	}
	evalHex := evalR.Inspect()
	if len(evalHex) != 32 {
		t.Errorf("eval secure_random 16: expected 32 hex chars, got %d (%q)", len(evalHex), evalHex)
	}
}

func TestCrossSecureRandomInt(t *testing.T) {
	program := parseProgram(t, "secure_random_int")
	evalR := evalResult(program)
	if evalR == nil || evalR.Type() != object.INTEGER {
		t.Errorf("eval secure_random_int: expected integer, got %v", evalR)
	}
}

func TestCrossSecureRandomRange(t *testing.T) {
	program := parseProgram(t, "secure_random_range 1 100")
	evalR := evalResult(program)
	if evalR == nil {
		t.Fatal("eval secure_random_range: got nil")
	}
	n, ok := object.ToInt(evalR)
	if !ok || n < 1 || n >= 100 {
		t.Errorf("eval secure_random_range 1 100: expected int in [1, 100), got %v", evalR)
	}
}

func TestCrossSecureRandomErrors(t *testing.T) {
	assertBothErrorContains(t, "secure_random 0", "between 1 and 1024")
	assertBothErrorContains(t, "secure_random 2000", "between 1 and 1024")
	assertBothErrorContains(t, "secure_random_range 5 3", "min must be less than max")
}

func TestCrossSecureRandomBytes(t *testing.T) {
	program := parseProgram(t, "secure_random_bytes 4")
	evalR := evalResult(program)
	vmR := vmResult(t, program)
	if evalR == nil || evalR.Type() != object.BYTES {
		t.Errorf("eval: expected bytes, got %v", evalR)
	}
	if vmR == nil || vmR.Type() != object.BYTES {
		t.Errorf("vm: expected bytes, got %v", vmR)
	}
	if evalR != nil && len(evalR.(*object.Bytes).Value) != 4 {
		t.Errorf("eval: expected 4 bytes, got %d", len(evalR.(*object.Bytes).Value))
	}
}

func TestCrossEncryptDecrypt(t *testing.T) {
	input := "key: secure_random 32\nplain: \"Hello World!\"\nenc: encrypt key plain\ndec: decrypt key enc\ndec"
	assertBothEqual(t, input, "Hello World!")
}

func TestCrossEncryptBytes(t *testing.T) {
	input := "key: secure_random 16\ndata: to_bytes \"Secret\"\nenc: encrypt key data\ndecrypt key enc"
	program := parseProgram(t, input)
	evalR := evalResult(program)
	vmR := vmResult(t, program)
	if evalR == nil || evalR.Inspect() != "Secret" {
		t.Errorf("eval: expected 'Secret', got %v", evalR)
	}
	if vmR == nil || vmR.Inspect() != "Secret" {
		t.Errorf("vm: expected 'Secret', got %v", vmR)
	}
}

func TestCrossDecryptWrongKey(t *testing.T) {
	assertBothErrorContains(t, "key: secure_random 32\nenc: encrypt key \"test\"\nwrong: secure_random 32\ndecrypt wrong enc", "authentication failed")
}

func TestCrossHmacSha256(t *testing.T) {
	input := "hmac_sha256 \"secret\" \"hello\""
	program := parseProgram(t, input)
	evalR := evalResult(program)
	vmR := vmResult(t, program)
	if evalR == nil || evalR.Type() != object.STRING {
		t.Errorf("eval: expected string, got %v", evalR)
	}
	if vmR == nil || vmR.Type() != object.STRING {
		t.Errorf("vm: expected string, got %v", vmR)
	}
	if evalR != nil && len(evalR.Inspect()) != 64 {
		t.Errorf("eval: expected 64 hex chars, got %d", len(evalR.Inspect()))
	}
}

func TestCrossHmacSha512(t *testing.T) {
	input := "hmac_sha512 \"key\" \"data\""
	program := parseProgram(t, input)
	evalR := evalResult(program)
	if evalR == nil || evalR.Type() != object.STRING {
		t.Errorf("eval: expected string, got %v", evalR)
	}
	if evalR != nil && len(evalR.Inspect()) != 128 {
		t.Errorf("eval: expected 128 hex chars, got %d", len(evalR.Inspect()))
	}
}

func TestCrossHmacDeterministic(t *testing.T) {
	assertBothEqual(t, "a: hmac_sha256 \"key\" \"hello\"\nb: hmac_sha256 \"key\" \"hello\"\nif a == b\n    1\nelse\n    0", "1")
}
