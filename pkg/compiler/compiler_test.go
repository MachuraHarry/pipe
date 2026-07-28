package compiler

import (
	"testing"

	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/parser"
)

func parseAndCompile(t *testing.T, input string) *Bytecode {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	c := New()
	if err := c.Compile(program); err != nil {
		t.Fatalf("compile error: %s", err)
	}
	return c.Bytecode()
}

func hasOp(t *testing.T, bc *Bytecode, op Opcode) bool {
	t.Helper()
	ins := bc.Instructions
	for i := 0; i < len(ins); {
		o := Opcode(ins[i])
		if o == op {
			return true
		}
		switch o {
		case OpConstant, OpGetGlobal, OpSetGlobal, OpGetLocal, OpSetLocal,
			OpGetBuiltin, OpDot, OpClosure:
			i += 3
		case OpCall, OpList, OpMap, OpJump, OpJumpNotTruthy, OpJumpBackward:
			i += 3
		default:
			i++
		}
	}
	return false
}

func countOps(t *testing.T, bc *Bytecode, op Opcode) int {
	t.Helper()
	count := 0
	ins := bc.Instructions
	for i := 0; i < len(ins); {
		o := Opcode(ins[i])
		if o == op {
			count++
		}
		switch o {
		case OpConstant, OpGetGlobal, OpSetGlobal, OpGetLocal, OpSetLocal,
			OpGetBuiltin, OpDot, OpClosure:
			i += 3
		case OpCall, OpList, OpMap, OpJump, OpJumpNotTruthy, OpJumpBackward:
			i += 3
		default:
			i++
		}
	}
	return count
}

func TestCompileInteger(t *testing.T) {
	bc := parseAndCompile(t, "42")
	if !hasOp(t, bc, OpConstant) {
		t.Error("expected OpConstant for integer")
	}
}

func TestCompileFloat(t *testing.T) {
	bc := parseAndCompile(t, "3.14")
	if !hasOp(t, bc, OpConstant) {
		t.Error("expected OpConstant for float")
	}
}

func TestCompileString(t *testing.T) {
	bc := parseAndCompile(t, `"hello"`)
	if !hasOp(t, bc, OpConstant) {
		t.Error("expected OpConstant for string")
	}
}

func TestCompileBoolean(t *testing.T) {
	bc := parseAndCompile(t, "true")
	if !hasOp(t, bc, OpTrue) {
		t.Error("expected OpTrue")
	}
	bc2 := parseAndCompile(t, "false")
	if !hasOp(t, bc2, OpFalse) {
		t.Error("expected OpFalse")
	}
}

func TestCompileNil(t *testing.T) {
	bc := parseAndCompile(t, "nil")
	if !hasOp(t, bc, OpNil) {
		t.Error("expected OpNil")
	}
}

func TestCompileArithmetic(t *testing.T) {
	bc := parseAndCompile(t, "1 + 2")
	if !hasOp(t, bc, OpAdd) {
		t.Error("expected OpAdd")
	}
	bc2 := parseAndCompile(t, "5 - 3")
	if !hasOp(t, bc2, OpSub) {
		t.Error("expected OpSub")
	}
	bc3 := parseAndCompile(t, "4 * 5")
	if !hasOp(t, bc3, OpMul) {
		t.Error("expected OpMul")
	}
	bc4 := parseAndCompile(t, "20 / 5")
	if !hasOp(t, bc4, OpDiv) {
		t.Error("expected OpDiv")
	}
}

func TestCompileComparison(t *testing.T) {
	bc := parseAndCompile(t, "1 == 2")
	if !hasOp(t, bc, OpEqual) {
		t.Error("expected OpEqual")
	}
	bc2 := parseAndCompile(t, "1 != 2")
	if !hasOp(t, bc2, OpNotEqual) {
		t.Error("expected OpNotEqual")
	}
	bc3 := parseAndCompile(t, "1 < 2")
	if !hasOp(t, bc3, OpLess) {
		t.Error("expected OpLess")
	}
	bc4 := parseAndCompile(t, "2 > 1")
	if !hasOp(t, bc4, OpGreater) {
		t.Error("expected OpGreater")
	}
}

func TestCompileConcat(t *testing.T) {
	bc := parseAndCompile(t, `"a" ++ "b"`)
	if !hasOp(t, bc, OpConcat) {
		t.Error("expected OpConcat")
	}
}

func TestCompileVariable(t *testing.T) {
	bc := parseAndCompile(t, "x: 42")
	if !hasOp(t, bc, OpSetGlobal) {
		t.Error("expected OpSetGlobal for variable definition")
	}
}

func TestCompileVariableRead(t *testing.T) {
	input := "x: 42\nx"
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpGetGlobal) {
		t.Error("expected OpGetGlobal for variable read")
	}
}

func TestCompileIfExpression(t *testing.T) {
	input := "if true\n    42\nelse\n    10"
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpJumpNotTruthy) {
		t.Error("expected OpJumpNotTruthy for if")
	}
	if !hasOp(t, bc, OpJump) {
		t.Error("expected OpJump for else branch skip")
	}
}

func TestCompileMatchExpression(t *testing.T) {
	input := "match 1\n    | 0 -> \"null\"\n    | 1 -> \"eins\"\n    | _ -> \"sonst\""
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpDup) {
		t.Error("expected OpDup for match value")
	}
}

func TestCompileFunction(t *testing.T) {
	input := "fn add a b\n    a + b\n\nadd 3 4"
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpClosure) {
		t.Error("expected OpClosure for function definition")
	}
	// Verify constants contain the compiled function
	if len(bc.Constants) == 0 {
		t.Error("expected constants for compiled function")
	}
}

func TestCompileFunctionCall(t *testing.T) {
	input := `print "hi"`
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpCall) {
		t.Error("expected OpCall for builtin call")
	}
}

func TestCompileList(t *testing.T) {
	input := "[1, 2, 3]"
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpList) {
		t.Error("expected OpList for list literal")
	}
}

func TestCompileMap(t *testing.T) {
	input := "{a: 1, b: 2}"
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpMap) {
		t.Error("expected OpMap for map literal")
	}
}

func TestCompilePipeline(t *testing.T) {
	input := "fn double x\n    x * 2\n\n42\n    > double"
	bc := parseAndCompile(t, input)
	// Pipeline should have OpCall or related ops
	if !hasOp(t, bc, OpClosure) {
		t.Error("expected OpClosure for function")
	}
	if len(bc.Instructions) == 0 {
		t.Error("expected instructions for pipeline")
	}
}

func TestCompileEmptyProgram(t *testing.T) {
	input := ""
	bc := parseAndCompile(t, input)
	if len(bc.Instructions) != 0 {
		t.Errorf("expected 0 instructions for empty program, got %d", len(bc.Instructions))
	}
}

func TestCompileConstants(t *testing.T) {
	input := `42 "hello" true`
	bc := parseAndCompile(t, input)
	if len(bc.Constants) < 2 {
		t.Errorf("expected at least 2 constants, got %d", len(bc.Constants))
	}
}

func TestCompileOpcodeNames(t *testing.T) {
	for _, op := range []Opcode{
		OpConstant, OpTrue, OpFalse, OpNil, OpPop,
		OpAdd, OpSub, OpMul, OpDiv, OpMod,
		OpEqual, OpNotEqual, OpGreater, OpLess,
		OpConcat, OpMinus,
		OpJump, OpJumpNotTruthy, OpJumpBackward,
		OpGetGlobal, OpSetGlobal,
		OpGetLocal, OpSetLocal,
		OpCall, OpClosure, OpList, OpMap, OpDot,
		OpReturn, OpReturnValue, OpHalt,
	} {
		if op.String() == "Opcode(unknown)" || op.String() == "" {
			t.Errorf("opcode %d has no name", op)
		}
	}
}

func TestCompilePrefixMinus(t *testing.T) {
	bc := parseAndCompile(t, "-5")
	if !hasOp(t, bc, OpMinus) {
		t.Error("expected OpMinus for prefix -")
	}
}

func TestCompileWhileLoop(t *testing.T) {
	input := "while true\n    x: x + 1"
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpJumpBackward) {
		t.Error("expected OpJumpBackward for while loop")
	}
}

func TestCompileMultiLine(t *testing.T) {
	input := "x: 10\ny: x + 5\nprint y"
	bc := parseAndCompile(t, input)
	if len(bc.Constants) < 2 {
		t.Errorf("expected at least 2 constants, got %d", len(bc.Constants))
	}
}

func TestCompileBuiltinCall(t *testing.T) {
	input := `print "hello"`
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpGetBuiltin) {
		t.Error("expected OpGetBuiltin for builtin function call")
	}
}

func TestCompileDotExpression(t *testing.T) {
	input := "m: {name: \"test\"}\nm.name"
	bc := parseAndCompile(t, input)
	if !hasOp(t, bc, OpDot) {
		t.Error("expected OpDot for dot expression")
	}
}

func TestSymbolTable(t *testing.T) {
	st := NewSymbolTable()
	s1 := st.Define("x")
	if s1.Scope != GlobalScope || s1.Index != 0 {
		t.Errorf("first define: scope=%v index=%d", s1.Scope, s1.Index)
	}
	s2 := st.Define("y")
	if s2.Index != 1 {
		t.Errorf("second define index: expected 1, got %d", s2.Index)
	}
	sym, ok := st.Resolve("x")
	if !ok || sym.Index != 0 {
		t.Errorf("resolve x: ok=%v, idx=%d", ok, sym.Index)
	}
	_, notFound := st.Resolve("z")
	if notFound {
		t.Error("should not find undefined symbol")
	}
}

func TestEnclosedSymbolTable(t *testing.T) {
	outer := NewSymbolTable()
	outer.Define("x")
	inner := NewEnclosedSymbolTable(outer)
	inner.Define("y")

	sym, ok := inner.Resolve("x")
	if !ok || sym.Scope != GlobalScope {
		t.Errorf("should resolve x from outer scope: ok=%v scope=%v", ok, sym.Scope)
	}
	sym2, ok := inner.Resolve("y")
	if !ok || sym2.Scope != LocalScope {
		t.Errorf("should resolve y from inner scope: ok=%v scope=%v", ok, sym2.Scope)
	}
}

func TestMakeInstructions(t *testing.T) {
	ins := Make(OpConstant, 42)
	if len(ins) != 3 {
		t.Errorf("OpConstant with operand: expected 3 bytes, got %d", len(ins))
	}
	if Opcode(ins[0]) != OpConstant {
		t.Errorf("first byte should be OpConstant, got %d", ins[0])
	}
	if ReadUint16(ins, 1) != 42 {
		t.Errorf("operand: expected 42, got %d", ReadUint16(ins, 1))
	}
}

func TestInstructionsString(t *testing.T) {
	ins := Instructions(Make(OpConstant, 1))
	s := ins.String()
	if s == "" {
		t.Error("Instructions.String() should not be empty")
	}
}
