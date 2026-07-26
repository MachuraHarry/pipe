package parser

import (
	"fmt"
	"testing"

	"github.com/harry/pulse/pkg/ast"
	"github.com/harry/pulse/pkg/lexer"
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
	input := "if x > 10\n    print \"groß\"\nelse\n    print \"klein\"\n"
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

func paramsToSlice(params []*ast.Identifier) []string {
	var s []string
	for _, p := range params {
		s = append(s, fmt.Sprintf("{%s}", p.Value))
	}
	return s
}
