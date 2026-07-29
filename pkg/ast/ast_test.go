package ast

import "testing"

func TestProgram(t *testing.T) {
	prog := &Program{Statements: nil}
	if prog.TokenLiteral() != "" {
		t.Errorf("empty program literal = %q, want empty", prog.TokenLiteral())
	}
	if prog.String() != "" {
		t.Errorf("empty program string = %q, want empty", prog.String())
	}

	prog.Statements = []Statement{
		&ExpressionStatement{Expression: &IntegerLiteral{Value: 42}},
	}
	if prog.TokenLiteral() != "42" {
		t.Errorf("program literal = %q, want 42", prog.TokenLiteral())
	}
}

func TestIntegerLiteral(t *testing.T) {
	n := &IntegerLiteral{Value: 42}
	if n.TokenLiteral() != "42" {
		t.Errorf("TokenLiteral = %q, want 42", n.TokenLiteral())
	}
	if n.String() != "42" {
		t.Errorf("String = %q, want 42", n.String())
	}
	n.expressionNode()
}

func TestFloatLiteral(t *testing.T) {
	n := &FloatLiteral{Value: 3.14}
	if n.TokenLiteral() != "3.14" {
		t.Errorf("TokenLiteral = %q, want 3.14", n.TokenLiteral())
	}
	if n.String() != "3.14" {
		t.Errorf("String = %q, want 3.14", n.String())
	}
	n.expressionNode()
}

func TestStringLiteral(t *testing.T) {
	n := &StringLiteral{Value: "hello"}
	if n.TokenLiteral() != "hello" {
		t.Errorf("TokenLiteral = %q, want hello", n.TokenLiteral())
	}
	if n.String() != `"hello"` {
		t.Errorf("String = %q, want \"hello\"", n.String())
	}
	n.expressionNode()
}

func TestBooleanLiteral(t *testing.T) {
	n := &BooleanLiteral{Value: true}
	if n.TokenLiteral() != "true" {
		t.Errorf("TokenLiteral = %q, want true", n.TokenLiteral())
	}
	if n.String() != "true" {
		t.Errorf("String = %q, want true", n.String())
	}
	n.expressionNode()

	n2 := &BooleanLiteral{Value: false}
	if n2.TokenLiteral() != "false" {
		t.Errorf("TokenLiteral = %q, want false", n2.TokenLiteral())
	}
}

func TestNilLiteral(t *testing.T) {
	n := &NilLiteral{}
	if n.TokenLiteral() != "nil" {
		t.Errorf("TokenLiteral = %q, want nil", n.TokenLiteral())
	}
	if n.String() != "nil" {
		t.Errorf("String = %q, want nil", n.String())
	}
	n.expressionNode()
}

func TestIdentifier(t *testing.T) {
	n := &Identifier{Value: "foobar"}
	if n.TokenLiteral() != "foobar" {
		t.Errorf("TokenLiteral = %q, want foobar", n.TokenLiteral())
	}
	if n.String() != "foobar" {
		t.Errorf("String = %q, want foobar", n.String())
	}
	n.expressionNode()
}

func TestExpressionStatement(t *testing.T) {
	n := &ExpressionStatement{Expression: &Identifier{Value: "x"}}
	if n.TokenLiteral() != "x" {
		t.Errorf("TokenLiteral = %q, want x", n.TokenLiteral())
	}
	if n.String() != "x" {
		t.Errorf("String = %q, want x", n.String())
	}
	n.statementNode()
}

func TestVarStatement(t *testing.T) {
	n := &VarStatement{
		Name:  &Identifier{Value: "x"},
		Value: &IntegerLiteral{Value: 10},
	}
	if n.TokenLiteral() != "x" {
		t.Errorf("TokenLiteral = %q, want x", n.TokenLiteral())
	}
	n.statementNode()
}

func TestBlockStatement(t *testing.T) {
	n := &BlockStatement{}
	if n.TokenLiteral() != "block" {
		t.Errorf("TokenLiteral = %q, want block", n.TokenLiteral())
	}
	n.statementNode()
}

func TestIfExpression(t *testing.T) {
	n := &IfExpression{
		Condition:   &Identifier{Value: "cond"},
		Consequence: &BlockStatement{},
	}
	if n.TokenLiteral() != "if" {
		t.Errorf("TokenLiteral = %q, want if", n.TokenLiteral())
	}
	n.expressionNode()

	// String without alternative
	s := n.String()
	if s != "if cond {\nblock}" {
		t.Errorf("String = %q, want if cond {\\nblock}", s)
	}

	// with alternative
	n.Alternative = &BlockStatement{}
	s = n.String()
	if s != "if cond {\nblock} else {\nblock}" {
		t.Errorf("String with else = %q, want if cond {\\nblock} else {\\nblock}", s)
	}
}

func TestMatchExpression(t *testing.T) {
	n := &MatchExpression{
		Value: &Identifier{Value: "x"},
		Cases: []MatchCase{
			{Pattern: &IntegerLiteral{Value: 1}, Body: &Identifier{Value: "one"}},
			{Pattern: &Identifier{Value: "_"}, Body: &IntegerLiteral{Value: 0}},
		},
	}
	if n.TokenLiteral() != "match" {
		t.Errorf("TokenLiteral = %q, want match", n.TokenLiteral())
	}
	n.expressionNode()
	s := n.String()
	if s != "match x\n  | 1 -> one\n  | _ -> 0" {
		t.Errorf("String = %q, want match x...", s)
	}
}

func TestFnStatement(t *testing.T) {
	n := &FnStatement{
		Name:       &Identifier{Value: "add"},
		Parameters: []*Identifier{{Value: "a"}, {Value: "b"}},
		Body:       &BlockStatement{},
	}
	if n.TokenLiteral() != "fn" {
		t.Errorf("TokenLiteral = %q, want fn", n.TokenLiteral())
	}
	n.statementNode()
}

func TestPrefixExpression(t *testing.T) {
	n := &PrefixExpression{
		Operator: "-",
		Right:    &IntegerLiteral{Value: 5},
	}
	if n.TokenLiteral() != "-" {
		t.Errorf("TokenLiteral = %q, want -", n.TokenLiteral())
	}
	if n.String() != "(-5)" {
		t.Errorf("String = %q, want (-5)", n.String())
	}
	n.expressionNode()
}

func TestInfixExpression(t *testing.T) {
	n := &InfixExpression{
		Left:     &IntegerLiteral{Value: 3},
		Operator: "+",
		Right:    &IntegerLiteral{Value: 4},
	}
	if n.TokenLiteral() != "+" {
		t.Errorf("TokenLiteral = %q, want +", n.TokenLiteral())
	}
	if n.String() != "(3 + 4)" {
		t.Errorf("String = %q, want (3 + 4)", n.String())
	}
	n.expressionNode()
}

func TestPipelineExpression(t *testing.T) {
	n := &PipelineExpression{
		Left:     &IntegerLiteral{Value: 10},
		Right:    &Identifier{Value: "double"},
		Parallel: false,
	}
	if n.TokenLiteral() != ">" {
		t.Errorf("TokenLiteral = %q, want >", n.TokenLiteral())
	}
	if n.String() != "(10 > double)" {
		t.Errorf("String = %q, want (10 > double)", n.String())
	}
	n.expressionNode()

	n.Parallel = true
	if n.TokenLiteral() != ">>" {
		t.Errorf("TokenLiteral = %q, want >>", n.TokenLiteral())
	}
	if n.String() != "(10 >> double)" {
		t.Errorf("String = %q, want (10 >> double)", n.String())
	}
}

func TestCallExpression(t *testing.T) {
	n := &CallExpression{
		Function: &Identifier{Value: "add"},
		Arguments: []Expression{
			&IntegerLiteral{Value: 1},
			&IntegerLiteral{Value: 2},
		},
	}
	if n.TokenLiteral() != "call" {
		t.Errorf("TokenLiteral = %q, want call", n.TokenLiteral())
	}
	if n.String() != "add(1 2)" {
		t.Errorf("String = %q, want add(1 2)", n.String())
	}
	n.expressionNode()
}

func TestListLiteral(t *testing.T) {
	n := &ListLiteral{
		Elements: []Expression{
			&IntegerLiteral{Value: 1},
			&IntegerLiteral{Value: 2},
			&IntegerLiteral{Value: 3},
		},
	}
	if n.TokenLiteral() != "[" {
		t.Errorf("TokenLiteral = %q, want [", n.TokenLiteral())
	}
	if n.String() != "[1, 2, 3]" {
		t.Errorf("String = %q, want [1, 2, 3]", n.String())
	}
	n.expressionNode()

	empty := &ListLiteral{}
	if empty.String() != "[]" {
		t.Errorf("empty list String = %q, want []", empty.String())
	}
}

func TestMapLiteral(t *testing.T) {
	n := &MapLiteral{
		Pairs: map[string]Expression{
			"a": &IntegerLiteral{Value: 1},
			"b": &IntegerLiteral{Value: 2},
		},
	}
	if n.TokenLiteral() != "{" {
		t.Errorf("TokenLiteral = %q, want {", n.TokenLiteral())
	}
	n.expressionNode()

	empty := &MapLiteral{Pairs: map[string]Expression{}}
	if empty.String() != "{}" {
		t.Errorf("empty map String = %q, want {}", empty.String())
	}
}

func TestDotExpression(t *testing.T) {
	n := &DotExpression{
		Left:  &Identifier{Value: "obj"},
		Field: "field",
	}
	if n.TokenLiteral() != "." {
		t.Errorf("TokenLiteral = %q, want .", n.TokenLiteral())
	}
	if n.String() != "(obj.field)" {
		t.Errorf("String = %q, want (obj.field)", n.String())
	}
	n.expressionNode()
}

func TestWhileExpression(t *testing.T) {
	n := &WhileExpression{
		Condition: &Identifier{Value: "cond"},
		Body:      &BlockStatement{},
	}
	if n.TokenLiteral() != "while" {
		t.Errorf("TokenLiteral = %q, want while", n.TokenLiteral())
	}
	if n.String() != "while cond { ... }" {
		t.Errorf("String = %q, want while cond { ... }", n.String())
	}
	n.expressionNode()
}

func TestForExpression(t *testing.T) {
	n := &ForExpression{
		Iterator: &Identifier{Value: "i"},
		Iterable: &Identifier{Value: "items"},
		IsForIn:  true,
	}
	if n.TokenLiteral() != "for" {
		t.Errorf("TokenLiteral = %q, want for", n.TokenLiteral())
	}
	n.expressionNode()

	n2 := &ForExpression{}
	if n2.String() != "for ... { ... }" {
		t.Errorf("String = %q, want for ... { ... }", n2.String())
	}
}

func TestBreakStatement(t *testing.T) {
	n := &BreakStatement{}
	if n.TokenLiteral() != "break" {
		t.Errorf("TokenLiteral = %q, want break", n.TokenLiteral())
	}
	n.statementNode()
}

func TestContinueStatement(t *testing.T) {
	n := &ContinueStatement{}
	if n.TokenLiteral() != "continue" {
		t.Errorf("TokenLiteral = %q, want continue", n.TokenLiteral())
	}
	n.statementNode()
}

func TestReturnStatement(t *testing.T) {
	n := &ReturnStatement{
		Value: &IntegerLiteral{Value: 42},
	}
	if n.TokenLiteral() != "return" {
		t.Errorf("TokenLiteral = %q, want return", n.TokenLiteral())
	}
	n.statementNode()
}

func TestImportStatement(t *testing.T) {
	n := &ImportStatement{Path: "math.pipe"}
	if n.TokenLiteral() != "import" {
		t.Errorf("TokenLiteral = %q, want import", n.TokenLiteral())
	}
	n.statementNode()
}

func TestDeferStatement(t *testing.T) {
	n := &DeferStatement{
		Expression: &Identifier{Value: "cleanup"},
	}
	if n.TokenLiteral() != "defer" {
		t.Errorf("TokenLiteral = %q, want defer", n.TokenLiteral())
	}
	n.statementNode()
}

func TestExportStatement(t *testing.T) {
	n := &ExportStatement{
		FnName: "add",
		Fn: &FnStatement{
			Name: &Identifier{Value: "add"},
		},
	}
	if n.TokenLiteral() != "export" {
		t.Errorf("TokenLiteral = %q, want export", n.TokenLiteral())
	}
	if n.ExportName() != "add" {
		t.Errorf("ExportName = %q, want add", n.ExportName())
	}
	n.statementNode()

	n2 := &ExportStatement{VarName: "x"}
	if n2.ExportName() != "x" {
		t.Errorf("ExportName = %q, want x", n2.ExportName())
	}

	n3 := &ExportStatement{EnumName: "Color"}
	if n3.ExportName() != "Color" {
		t.Errorf("ExportName = %q, want Color", n3.ExportName())
	}
}

func TestEnumStatement(t *testing.T) {
	n := &EnumStatement{
		Name:   "Color",
		Values: []string{"Red", "Green", "Blue"},
	}
	if n.TokenLiteral() != "enum" {
		t.Errorf("TokenLiteral = %q, want enum", n.TokenLiteral())
	}
	n.statementNode()
}

func TestFnLiteral(t *testing.T) {
	n := &FnLiteral{
		Parameters: []*Identifier{{Value: "x"}},
		Body:       &BlockStatement{},
	}
	if n.TokenLiteral() != "fn" {
		t.Errorf("TokenLiteral = %q, want fn", n.TokenLiteral())
	}
	if n.String() != "fn(...)" {
		t.Errorf("String = %q, want fn(...)", n.String())
	}
	n.expressionNode()
}

func TestSliceExpression(t *testing.T) {
	n := &SliceExpression{
		List:  &Identifier{Value: "items"},
		Start: &IntegerLiteral{Value: 1},
		End:   &IntegerLiteral{Value: 3},
	}
	if n.TokenLiteral() != "slice" {
		t.Errorf("TokenLiteral = %q, want slice", n.TokenLiteral())
	}
	if n.String() != "slice" {
		t.Errorf("String = %q, want slice", n.String())
	}
	n.expressionNode()
}

func TestTryExpression(t *testing.T) {
	n := &TryExpression{
		TryBlock:   &BlockStatement{},
		CatchParam: &Identifier{Value: "e"},
		CatchBlock: &BlockStatement{},
	}
	if n.TokenLiteral() != "try" {
		t.Errorf("TokenLiteral = %q, want try", n.TokenLiteral())
	}
	if n.String() != "try ... catch" {
		t.Errorf("String = %q, want try ... catch", n.String())
	}
	n.expressionNode()

	n2 := &TryExpression{AIFix: true}
	if n2.TokenLiteral() != "try_ai" {
		t.Errorf("TokenLiteral = %q, want try_ai", n2.TokenLiteral())
	}
}
