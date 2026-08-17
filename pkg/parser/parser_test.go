package parser

import (
	"fmt"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/lexer"
)

func TestIntegerLiteral(t *testing.T) {
	input := `42`
	program := parseProgram(t, input)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}

	testIntegerLiteral(t, stmt.Expression, 42)
}

func TestFloatLiteral(t *testing.T) {
	input := `3.14`
	program := parseProgram(t, input)

	float, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.FloatLiteral)
	if !ok {
		t.Fatalf("expected FloatLiteral, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if float.Value != 3.14 {
		t.Errorf("expected 3.14, got %g", float.Value)
	}
}

func TestStringLiteral(t *testing.T) {
	input := `"hello"`
	program := parseProgram(t, input)

	str, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if str.Value != "hello" {
		t.Errorf("expected 'hello', got %q", str.Value)
	}
}

func TestBooleanLiteral(t *testing.T) {
	tests := []struct {
		input string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		expr := program.Statements[0].(*ast.ExpressionStatement).Expression
		boolean, ok := expr.(*ast.BooleanLiteral)
		if !ok {
			t.Fatalf("%s: expected BooleanLiteral, got %T", tt.input, expr)
		}
		if boolean.Value != tt.value {
			t.Errorf("%s: expected %t, got %t", tt.input, tt.value, boolean.Value)
		}
	}
}

func TestNilLiteral(t *testing.T) {
	input := `nil`
	program := parseProgram(t, input)
	_, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.NilLiteral)
	if !ok {
		t.Fatalf("expected NilLiteral")
	}
}

func TestInfixExpression(t *testing.T) {
	tests := []struct {
		input    string
		left     string
		operator string
		right    string
	}{
		{"1 + 2", "1", "+", "2"},
		{"x * y", "x", "*", "y"},
		{"a == b", "a", "==", "b"},
		{"x < 10", "x", "<", "10"},
		{"\"a\" ++ \"b\"", "\"a\"", "++", "\"b\""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseProgram(t, tt.input)
			expr := program.Statements[0].(*ast.ExpressionStatement).Expression
			testInfixExpression(t, expr, tt.left, tt.operator, tt.right)
		})
	}
}

func TestVariableDef(t *testing.T) {
	input := "x: 42"
	program := parseProgram(t, input)

	stmt, ok := program.Statements[0].(*ast.VarStatement)
	if !ok {
		t.Fatalf("expected VarStatement, got %T", program.Statements[0])
	}
	if stmt.Name.Value != "x" {
		t.Errorf("expected name 'x', got %q", stmt.Name.Value)
	}
	testIntegerLiteral(t, stmt.Value, 42)
}

func TestVariableDefWithType(t *testing.T) {
	input := "x: int = 42"
	program := parseProgram(t, input)

	stmt, ok := program.Statements[0].(*ast.VarStatement)
	if !ok {
		t.Fatalf("expected VarStatement, got %T", program.Statements[0])
	}
	if stmt.Name.Value != "x" {
		t.Errorf("expected name 'x', got %q", stmt.Name.Value)
	}
	if stmt.TypeAnnotation == nil || stmt.TypeAnnotation.Name != "int" {
		t.Errorf("expected type 'int', got %v", stmt.TypeAnnotation)
	}
	testIntegerLiteral(t, stmt.Value, 42)
}

func TestFnDefWithTypes(t *testing.T) {
	input := "fn add(a: int, b: int) -> int\n    a + b"
	program := parseProgram(t, input)

	stmt, ok := program.Statements[0].(*ast.FnStatement)
	if !ok {
		t.Fatalf("expected FnStatement, got %T", program.Statements[0])
	}
	if stmt.Name.Value != "add" {
		t.Errorf("expected name 'add', got %q", stmt.Name.Value)
	}
	if len(stmt.Parameters) != 2 {
		t.Fatalf("expected 2 params, got %d", len(stmt.Parameters))
	}
	if stmt.Parameters[0].Value != "a" || stmt.Parameters[1].Value != "b" {
		t.Errorf("expected params [a, b], got [%s, %s]", stmt.Parameters[0].Value, stmt.Parameters[1].Value)
	}
	if stmt.ParamTypes[0] == nil || stmt.ParamTypes[0].Name != "int" {
		t.Errorf("expected first param type 'int', got %v", stmt.ParamTypes[0])
	}
	if stmt.ParamTypes[1] == nil || stmt.ParamTypes[1].Name != "int" {
		t.Errorf("expected second param type 'int', got %v", stmt.ParamTypes[1])
	}
	if stmt.ReturnType == nil || stmt.ReturnType.Name != "int" {
		t.Errorf("expected return type 'int', got %v", stmt.ReturnType)
	}
}

func TestInlineLambda(t *testing.T) {
	input := "fn x: x + 1"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	lit, ok := expr.Expression.(*ast.FnLiteral)
	if !ok {
		t.Fatalf("expected FnLiteral, got %T", expr.Expression)
	}
	if len(lit.Parameters) != 1 {
		t.Errorf("expected 1 param, got %d", len(lit.Parameters))
	}
	if lit.Parameters[0].Value != "x" {
		t.Errorf("expected param 'x', got %q", lit.Parameters[0].Value)
	}
	if len(lit.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(lit.Body.Statements))
	}
}

func TestInlineLambdaMultiParam(t *testing.T) {
	input := "fn a b: a + b"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	lit, ok := expr.Expression.(*ast.FnLiteral)
	if !ok {
		t.Fatalf("expected FnLiteral, got %T", expr.Expression)
	}
	if len(lit.Parameters) != 2 {
		t.Errorf("expected 2 params, got %d", len(lit.Parameters))
	}
}

func TestInlineLambdaAsArgument(t *testing.T) {
	input := "filter list (fn x: x > 0)"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	call, ok := expr.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", expr.Expression)
	}
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Arguments))
	}
	lit, ok := call.Arguments[1].(*ast.FnLiteral)
	if !ok {
		t.Fatalf("expected FnLiteral as second arg, got %T", call.Arguments[1])
	}
	if len(lit.Parameters) != 1 || lit.Parameters[0].Value != "x" {
		t.Errorf("expected param [x], got %v", paramsToSlice(lit.Parameters))
	}
}

func TestFunctionDef(t *testing.T) {
	input := "fn greet name\n    \"Hallo \" ++ name\n"
	program := parseProgram(t, input)

	stmt, ok := program.Statements[0].(*ast.FnStatement)
	if !ok {
		t.Fatalf("expected FnStatement, got %T", program.Statements[0])
	}
	if stmt.Name.Value != "greet" {
		t.Errorf("expected 'greet', got %q", stmt.Name.Value)
	}
	if len(stmt.Parameters) != 1 || stmt.Parameters[0].Value != "name" {
		t.Errorf("expected param [name], got %v", paramsToSlice(stmt.Parameters))
	}
	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(stmt.Body.Statements))
	}
}

func TestFunctionDefMultiParam(t *testing.T) {
	input := "fn add a b\n    a + b\n"
	program := parseProgram(t, input)

	stmt := program.Statements[0].(*ast.FnStatement)
	if len(stmt.Parameters) != 2 {
		t.Errorf("expected 2 params, got %d", len(stmt.Parameters))
	}
}

func TestIfExpression(t *testing.T) {
	input := "if x > 10\n    print \"big\"\nelse\n    print \"small\"\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("expected IfExpression, got %T", program.Statements[0])
	}
	if expr.Condition.String() != "(x > 10)" {
		t.Errorf("condition: got %s", expr.Condition.String())
	}
	if len(expr.Consequence.Statements) != 1 {
		t.Errorf("consequence: expected 1, got %d", len(expr.Consequence.Statements))
	}
	if expr.Alternative == nil {
		t.Fatal("expected alternative (else) block")
	}
}

func TestMatchExpression(t *testing.T) {
	input := "match x\n    | 0 -> \"null\"\n    | _ -> \"other\"\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", program.Statements[0])
	}
	if len(expr.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(expr.Cases))
	}
}

func TestMatchMultiPattern(t *testing.T) {
	input := "match x\n    | 1 | 2 | 3 -> \"small\"\n    | _ -> \"big\"\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", program.Statements[0])
	}
	if len(expr.Cases) != 4 {
		t.Fatalf("expected 4 cases after multi-pattern expansion, got %d", len(expr.Cases))
	}
	// Every case must share the same body: "small" for 1, 2, 3; "big" for _
	for i, c := range expr.Cases {
		want := `"big"`
		if i < 3 {
			want = `"small"`
		}
		if c.Body.String() != want {
			t.Errorf("case %d body = %s, want %s", i, c.Body.String(), want)
		}
	}
	// Last case pattern must be the wildcard
	if id, ok := expr.Cases[3].Pattern.(*ast.Identifier); !ok || id.Value != "_" {
		t.Errorf("expected wildcard as last pattern, got %T", expr.Cases[3].Pattern)
	}
}

func TestMatchGuard(t *testing.T) {
	input := "match x\n    | _ if x > 0 -> \"positive\"\n    | _ -> \"other\"\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if len(expr.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(expr.Cases))
	}
	if expr.Cases[0].Guard == nil {
		t.Fatal("expected guard on first case, got nil")
	}
	if expr.Cases[0].Guard.String() != "(x > 0)" {
		t.Errorf("guard = %s, want (x > 0)", expr.Cases[0].Guard.String())
	}
	if expr.Cases[1].Guard != nil {
		t.Error("expected no guard on wildcard case")
	}
}

func TestMatchGuardMultiPattern(t *testing.T) {
	input := "match x\n    | 1 | 2 if x > 1 -> \"small\"\n    | _ -> \"other\"\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if len(expr.Cases) != 3 {
		t.Fatalf("expected 3 cases after multi-pattern expansion, got %d", len(expr.Cases))
	}
	// Both expanded patterns share the same guard and body
	for i := 0; i < 2; i++ {
		c := expr.Cases[i]
		if c.Guard == nil {
			t.Fatalf("case %d: expected guard, got nil", i)
		}
		if c.Guard.String() != "(x > 1)" {
			t.Errorf("case %d guard = %s, want (x > 1)", i, c.Guard.String())
		}
		if c.Body.String() != `"small"` {
			t.Errorf("case %d body = %s, want \"small\"", i, c.Body.String())
		}
	}
	if expr.Cases[2].Guard != nil {
		t.Error("expected no guard on wildcard case")
	}
}

func TestPipelineExpression(t *testing.T) {
	input := "x\n    > f\n    > g\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.PipelineExpression)
	if !ok {
		t.Fatalf("expected PipelineExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if expr.String() != "((x > f) > g)" {
		t.Errorf("got %s", expr.String())
	}
}

func TestListLiteral(t *testing.T) {
	input := `[1, 2, 3]`
	program := parseProgram(t, input)

	list, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.ListLiteral)
	if !ok {
		t.Fatalf("expected ListLiteral, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if len(list.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(list.Elements))
	}
}

func TestMapLiteral(t *testing.T) {
	input := `{a: 1, b: 2}`
	program := parseProgram(t, input)

	m, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.MapLiteral)
	if !ok {
		t.Fatalf("expected MapLiteral, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if len(m.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(m.Pairs))
	}
}

func TestDotExpression(t *testing.T) {
	input := `user.name`
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.DotExpression)
	if !ok {
		t.Fatalf("expected DotExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if expr.Field != "name" {
		t.Errorf("expected field 'name', got %q", expr.Field)
	}
}

func TestPipelineVertical(t *testing.T) {
	input := "users\n    > filter\n    > map\n    > print\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.PipelineExpression)
	if !ok {
		t.Fatalf("expected PipelineExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if expr.String() != "(((users > filter) > map) > print)" {
		t.Errorf("got %s", expr.String())
	}
}

func TestPrefixExpression(t *testing.T) {
	input := `-x`
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.PrefixExpression)
	if !ok {
		t.Fatalf("expected PrefixExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if expr.Operator != "-" {
		t.Errorf("expected '-', got %q", expr.Operator)
	}
	ident, ok := expr.Right.(*ast.Identifier)
	if !ok || ident.Value != "x" {
		t.Errorf("expected right=ident('x'), got %v", expr.Right)
	}
}

func TestGroupedExpression(t *testing.T) {
	input := `(1 + 2)`
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if expr.Operator != "+" {
		t.Errorf("expected '+', got %q", expr.Operator)
	}
}

func TestEmptyInput(t *testing.T) {
	input := ""
	program := parseProgram(t, input)
	if len(program.Statements) != 0 {
		t.Errorf("expected 0 statements, got %d", len(program.Statements))
	}
}

// ---- Helpers ----

func parseProgram(t *testing.T, input string) *ast.Program {
	t.Helper()
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)
	return program
}

func checkParserErrors(t *testing.T, p *Parser) {
	t.Helper()
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}
	for _, msg := range errors {
		t.Errorf("parser error: %s", msg)
	}
	t.FailNow()
}

func testIntegerLiteral(t *testing.T, expr ast.Expression, value int64) {
	t.Helper()
	intLit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", expr)
	}
	if intLit.Value != value {
		t.Errorf("expected %d, got %d", value, intLit.Value)
	}
}

func testInfixExpression(t *testing.T, expr ast.Expression, left string, op string, right string) {
	t.Helper()
	infix, ok := expr.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", expr)
	}
	if infix.Left.String() != left {
		t.Errorf("left: expected %s, got %s", left, infix.Left.String())
	}
	if infix.Operator != op {
		t.Errorf("operator: expected %s, got %s", op, infix.Operator)
	}
	if infix.Right.String() != right {
		t.Errorf("right: expected %s, got %s", right, infix.Right.String())
	}
}

func TestParallelPipelineExpression(t *testing.T) {
	input := "x\n    >> f\n    >> g\n"
	program := parseProgram(t, input)

	expr, ok := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.PipelineExpression)
	if !ok {
		t.Fatalf("expected PipelineExpression, got %T", program.Statements[0].(*ast.ExpressionStatement).Expression)
	}
	if !expr.Parallel {
		t.Errorf("expected Parallel=true")
	}
	if expr.String() != "((x >> f) >> g)" {
		t.Errorf("got %s", expr.String())
	}
}

func paramsToSlice(params []*ast.Identifier) []string {
	var s []string
	for _, p := range params {
		s = append(s, fmt.Sprintf("{%s}", p.Value))
	}
	return s
}

func TestTestStatement(t *testing.T) {
	program := parseProgram(t, "test \"basic\"\n    assert_eq 1 1")
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	ts, ok := program.Statements[0].(*ast.TestStatement)
	if !ok {
		t.Fatalf("expected TestStatement, got %T", program.Statements[0])
	}
	if ts.Hook != "" {
		t.Errorf("regular test should not be a hook, got %q", ts.Hook)
	}
	if ts.Name == nil || ts.Name.Value != "basic" {
		t.Errorf("expected name 'basic', got %+v", ts.Name)
	}
}

func TestTestStatementHooks(t *testing.T) {
	program := parseProgram(t, "test setup\n    x: 1\ntest teardown\n    x: 2")
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	setup, ok := program.Statements[0].(*ast.TestStatement)
	if !ok {
		t.Fatalf("expected TestStatement, got %T", program.Statements[0])
	}
	if setup.Hook != "setup" {
		t.Errorf("expected hook 'setup', got %q", setup.Hook)
	}
	if setup.Name != nil {
		t.Errorf("hook should not have a name, got %+v", setup.Name)
	}

	teardown, ok := program.Statements[1].(*ast.TestStatement)
	if !ok {
		t.Fatalf("expected TestStatement, got %T", program.Statements[1])
	}
	if teardown.Hook != "teardown" {
		t.Errorf("expected hook 'teardown', got %q", teardown.Hook)
	}
}

func TestTestStatementHookRequiresBlock(t *testing.T) {
	// A hook name must be followed by a block; a bare `test setup` without a
	// block body must fail to parse.
	p := New(lexer.New("test setup"))
	p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Error("expected a parse error for a hook without a block")
	}
}

func TestSelectExpression(t *testing.T) {
	input := "select\n    | ch1 -> print \"got from ch1\"\n    | ch2 -> print \"got from ch2\"\n    | default -> print \"no data\""
	program := parseProgram(t, input)

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	se, ok := stmt.Expression.(*ast.SelectExpression)
	if !ok {
		t.Fatalf("expected SelectExpression, got %T", stmt.Expression)
	}
	if len(se.Cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(se.Cases))
	}
	if se.Cases[2].IsDefault != true {
		t.Error("expected third case to be default")
	}
}
