package vm

import (
	"strings"
	"testing"

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

func TestVMDeepButLegalRecursion(t *testing.T) {
	input := "fn count n acc\n    if n <= 0\n        acc\n    else\n        count (n - 1) (acc + 1)\n\ncount 300 0"
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "300" {
		t.Errorf("expected 300, got %s", result)
	}
}

func TestVMRecursionOverflowNoCrash(t *testing.T) {
	// Unbounded recursion must not crash the process. Depending on which
	// limit fires first, Run() either returns an error (operand-stack
	// recovery) or the script's result is a catchable E008 error object
	// (frame guard) — never a Go panic.
	for _, input := range []string{
		"fn f x\n    f x\n\nf 0",
		"fn count n acc\n    if n <= 0\n        acc\n    else\n        count (n - 1) (acc + 1)\n\ncount 100000 0",
	} {
		bc := parseAndCompile(t, input)
		v := New(bc)
		err := v.Run()
		if err == nil {
			top := v.LastPoppedStackElem()
			if _, isErr := top.(*object.Error); !isErr {
				t.Errorf("expected error or error object for unbounded recursion, got %s (%v)", top.Inspect(), err)
			}
		}
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

func TestVMIfConsequenceVarStatementStackBalance(t *testing.T) {
	input := `fn f cond
    x: 0
    if cond
        x: 42
    x + 1

a: f true
b: f false
a ++ "," ++ b`
	bc := parseAndCompile(t, input)
	result := runVM(t, bc)
	if result != "43,1" {
		t.Errorf("expected 43,1, got %q", result)
	}
}
