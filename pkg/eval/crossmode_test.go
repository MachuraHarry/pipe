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
	ctx := NewEvalContext("<cross>")
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
