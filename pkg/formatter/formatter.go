package formatter

import (
	"fmt"
	"os"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

func Format(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result := FormatSource(string(data))
	if result == string(data) {
		return nil
	}
	return os.WriteFile(path, []byte(result), 0644)
}

func FormatCheck(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	result := FormatSource(string(data))
	return result == string(data), nil
}

func FormatSource(src string) string {
	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return fallbackFormat(src)
	}
	return formatProgram(program)
}

func fallbackFormat(src string) string {
	lines := strings.Split(src, "\n")
	var out strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if trimmed == "" {
			out.WriteByte('\n')
			continue
		}
		idx := countLeadingSpaces(trimmed)
		content := strings.TrimLeft(trimmed, " \t")
		norm := (idx / 4) * 4
		for i := 0; i < norm; i++ {
			out.WriteByte(' ')
		}
		out.WriteString(strings.TrimSpace(content))
		out.WriteByte('\n')
	}
	result := out.String()
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

func FormatProgram(prog *ast.Program) string {
	return formatProgram(prog)
}

func formatProgram(program *ast.Program) string {
	var out strings.Builder
	var lastWasDef bool

	for i, stmt := range program.Statements {
		if i > 0 && lastWasDef {
			out.WriteByte('\n')
		}
		formatStatement(&out, stmt, 0)
		lastWasDef = isDefinition(stmt)

		if i < len(program.Statements)-1 && lastWasDef {
			out.WriteByte('\n')
		}
	}
	if out.Len() == 0 || out.String()[out.Len()-1] != '\n' {
		out.WriteByte('\n')
	}
	return out.String()
}

func isDefinition(stmt ast.Statement) bool {
	switch stmt.(type) {
	case *ast.FnStatement, *ast.ExportStatement, *ast.EnumStatement:
		return true
	}
	return false
}

func formatStatement(out *strings.Builder, stmt ast.Statement, depth int) {
	indent := strings.Repeat("    ", depth)

	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		if _, isPipeline := s.Expression.(*ast.PipelineExpression); !isPipeline {
			out.WriteString(indent)
		}
		formatExpr(out, s.Expression, depth, 0)
		if !strings.HasSuffix(out.String(), "\n") {
			out.WriteByte('\n')
		}

	case *ast.VarStatement:
		out.WriteString(indent)
		out.WriteString(s.Name.Value)
		out.WriteString(": ")
		if _, isPipeline := s.Value.(*ast.PipelineExpression); isPipeline {
			formatPipelineTop(out, s.Value, 0)
		} else {
			formatExpr(out, s.Value, depth, 0)
		}
		out.WriteByte('\n')

	case *ast.FnStatement:
		out.WriteString(indent)
		out.WriteString("fn ")
		out.WriteString(s.Name.Value)
		for _, p := range s.Parameters {
			out.WriteByte(' ')
			out.WriteString(p.Value)
		}
		out.WriteByte('\n')
		formatBlock(out, s.Body, depth+1)
		if s.Body == nil || len(s.Body.Statements) == 0 {
			out.WriteByte('\n')
		}

	case *ast.ExportStatement:
		if s.Fn != nil {
			out.WriteString(indent)
			out.WriteString("export fn ")
			out.WriteString(s.FnName)
			out.WriteByte('\n')
			formatBlock(out, s.Fn.Body, depth+1)
		}
		if s.Var != nil {
			out.WriteString(indent)
			out.WriteString("export ")
			out.WriteString(s.VarName)
			out.WriteString(": ")
			formatExpr(out, s.Var.Value, depth, 0)
			out.WriteByte('\n')
		}
		if s.Enum != nil {
			out.WriteString(indent)
			out.WriteString("export enum ")
			out.WriteString(s.EnumName)
			out.WriteString(": ")
			for i, v := range s.Enum.Values {
				if i > 0 {
					out.WriteString(", ")
				}
				out.WriteString(v)
			}
			out.WriteByte('\n')
		}

	case *ast.EnumStatement:
		out.WriteString(indent)
		out.WriteString("enum ")
		out.WriteString(s.Name)
		out.WriteString(": ")
		for i, v := range s.Values {
			if i > 0 {
				out.WriteString(", ")
			}
			out.WriteString(v)
		}
		out.WriteByte('\n')

	case *ast.ImportStatement:
		out.WriteString(indent)
		out.WriteString("import ")
		out.WriteString(parser.QuoteString(s.Path))
		out.WriteByte('\n')

	case *ast.ReturnStatement:
		out.WriteString(indent)
		out.WriteString("return ")
		formatExpr(out, s.Value, 0, 0)
		out.WriteByte('\n')

	case *ast.DeferStatement:
		out.WriteString(indent)
		out.WriteString("defer ")
		formatExpr(out, s.Expression, 0, 0)
		out.WriteByte('\n')

	case *ast.BreakStatement:
		out.WriteString(indent)
		out.WriteString("break\n")

	case *ast.ContinueStatement:
		out.WriteString(indent)
		out.WriteString("continue\n")

	case *ast.TestStatement:
		out.WriteString(indent)
		out.WriteString("test ")
		if s.Hook != "" {
			out.WriteString(s.Hook)
		} else {
			out.WriteString(parser.QuoteString(s.Name.Value))
		}
		out.WriteByte('\n')
		formatBlock(out, s.Body, depth+1)
	}
}

func formatBlock(out *strings.Builder, block *ast.BlockStatement, depth int) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		formatStatement(out, stmt, depth)
	}
}

func formatExpr(out *strings.Builder, expr ast.Expression, depth int, prec int) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		out.WriteString(fmt.Sprintf("%d", e.Value))
	case *ast.FloatLiteral:
		out.WriteString(fmt.Sprintf("%g", e.Value))
	case *ast.StringLiteral:
		out.WriteString(parser.QuoteString(e.Value))
	case *ast.BooleanLiteral:
		if e.Value {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case *ast.NilLiteral:
		out.WriteString("nil")
	case *ast.Identifier:
		out.WriteString(e.Value)

	case *ast.PrefixExpression:
		out.WriteString(e.Operator)
		formatExpr(out, e.Right, depth, 30)

	case *ast.InfixExpression:
		formatExpr(out, e.Left, depth, precOf(e.Operator))
		out.WriteByte(' ')
		out.WriteString(e.Operator)
		out.WriteByte(' ')
		formatExpr(out, e.Right, depth, precOf(e.Operator))

	case *ast.CallExpression:
		formatExpr(out, e.Function, depth, 0)
		for _, arg := range e.Arguments {
			out.WriteByte(' ')
			formatCallArg(out, arg, depth)
		}

	case *ast.PipelineExpression:
		formatPipelineTop(out, expr, depth)

	case *ast.IfExpression:
		out.WriteString("if ")
		formatExpr(out, e.Condition, depth, 0)
		out.WriteByte('\n')
		formatBlock(out, e.Consequence, depth+1)
		if e.Alternative != nil {
			indent := strings.Repeat("    ", depth)
			out.WriteString(indent)
			out.WriteString("else\n")
			formatBlock(out, e.Alternative, depth+1)
		}

	case *ast.WhileExpression:
		out.WriteString("while ")
		formatExpr(out, e.Condition, depth, 0)
		out.WriteByte('\n')
		formatBlock(out, e.Body, depth+1)

	case *ast.ForExpression:
		if e.IsForIn {
			out.WriteString("for ")
			out.WriteString(e.Iterator.Value)
			out.WriteString(" in ")
			formatExpr(out, e.Iterable, depth, 0)
			out.WriteByte('\n')
			formatBlock(out, e.Body, depth+1)
		} else if e.Init != nil || e.Condition != nil || e.Update != nil {
			out.WriteString("for ")
			if init, ok := e.Init.(*ast.VarStatement); ok {
				out.WriteString(init.Name.Value)
				out.WriteString(": ")
				formatExpr(out, init.Value, depth, 0)
			}
			out.WriteString("; ")
			if e.Condition != nil {
				formatExpr(out, e.Condition, depth, 0)
			}
			out.WriteString("; ")
			if upd, ok := e.Update.(*ast.VarStatement); ok {
				out.WriteString(upd.Name.Value)
				out.WriteString(": ")
				formatExpr(out, upd.Value, depth, 0)
			} else if updExpr, ok := e.Update.(*ast.ExpressionStatement); ok {
				formatExpr(out, updExpr.Expression, depth, 0)
			}
			out.WriteByte('\n')
			formatBlock(out, e.Body, depth+1)
		}

	case *ast.MatchExpression:
		out.WriteString("match ")
		formatExpr(out, e.Value, depth, 0)
		out.WriteByte('\n')
		for _, c := range e.Cases {
			indent := strings.Repeat("    ", depth+1)
			out.WriteString(indent)
			out.WriteString("| ")
			formatExpr(out, c.Pattern, depth, 0)
			if c.Guard != nil {
				out.WriteString(" if ")
				formatExpr(out, c.Guard, depth, 0)
			}
			out.WriteString(" -> ")
			formatExpr(out, c.Body, depth, 0)
			out.WriteByte('\n')
		}

	case *ast.FnLiteral:
		out.WriteString("fn ")
		for i, p := range e.Parameters {
			if i > 0 {
				out.WriteByte(' ')
			}
			out.WriteString(p.Value)
		}
		// Inline form: fn x: expression (single-expression body)
		if len(e.Body.Statements) == 1 {
			if exprStmt, ok := e.Body.Statements[0].(*ast.ExpressionStatement); ok {
				out.WriteString(": ")
				formatExpr(out, exprStmt.Expression, depth, 0)
				break
			}
		}
		out.WriteByte('\n')
		formatBlock(out, e.Body, depth+1)

	case *ast.ListLiteral:
		out.WriteByte('[')
		for i, elem := range e.Elements {
			if i > 0 {
				out.WriteString(", ")
			}
			formatExpr(out, elem, depth, 0)
		}
		out.WriteByte(']')

	case *ast.MapLiteral:
		out.WriteByte('{')
		i := 0
		for k, v := range e.Pairs {
			if i > 0 {
				out.WriteString(", ")
			}
			out.WriteString(k)
			out.WriteString(": ")
			formatExpr(out, v, depth, 0)
			i++
		}
		out.WriteByte('}')

	case *ast.DotExpression:
		formatExpr(out, e.Left, depth, precOf("."))
		out.WriteByte('.')
		out.WriteString(e.Field)

	case *ast.SliceExpression:
		formatExpr(out, e.List, depth, 0)
		out.WriteByte('[')
		if e.Start != nil {
			formatExpr(out, e.Start, depth, 0)
		}
		out.WriteString("..")
		if e.End != nil {
			formatExpr(out, e.End, depth, 0)
		}
		out.WriteByte(']')

	case *ast.TryExpression:
		if e.AIFix {
			out.WriteString("try_ai\n")
		} else {
			out.WriteString("try\n")
		}
		formatBlock(out, e.TryBlock, depth+1)
		if e.CatchBlock != nil {
			indent := strings.Repeat("    ", depth)
			out.WriteString(indent)
			out.WriteString("catch")
			if e.CatchParam != nil {
				out.WriteByte(' ')
				out.WriteString(e.CatchParam.Value)
			}
			out.WriteByte('\n')
			formatBlock(out, e.CatchBlock, depth+1)
		}
	}
}

func formatCallArg(out *strings.Builder, expr ast.Expression, depth int) {
	switch expr.(type) {
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral,
		*ast.BooleanLiteral, *ast.NilLiteral, *ast.Identifier:
		formatExpr(out, expr, depth, 0)
	default:
		// Pipe call arguments are parsed as value tokens; wrap complex
		// expressions in parentheses so they reparse to the same AST.
		out.WriteByte('(')
		formatExpr(out, expr, depth, 0)
		out.WriteByte(')')
	}
}

// formatPipelineTop renders a pipeline that begins a logical line at the given
// depth: the base value on its own line followed by one indented stage per
// pipeline hop. Each line ends with a newline.
func formatPipelineTop(out *strings.Builder, expr ast.Expression, depth int) {
	pe, ok := expr.(*ast.PipelineExpression)
	if !ok {
		formatExpr(out, expr, depth, 0)
		return
	}
	indent := strings.Repeat("    ", depth)

	leftmost, stages := pipelineStages(pe)
	out.WriteString(indent)
	formatExpr(out, leftmost, depth, 0)
	out.WriteByte('\n')

	for _, stage := range stages {
		out.WriteString(indent + "    ")
		if stage.Parallel {
			out.WriteString(">> ")
		} else {
			out.WriteString("> ")
		}
		if call, ok := stage.Right.(*ast.CallExpression); ok {
			formatExpr(out, call.Function, depth, 0)
			for _, arg := range call.Arguments {
				out.WriteByte(' ')
				formatCallArg(out, arg, depth)
			}
		} else {
			formatExpr(out, stage.Right, depth, 0)
		}
		out.WriteByte('\n')
	}
}

// pipelineStages splits a left-nested pipeline into the base value and the
// ordered list of stages from first to last.
func pipelineStages(pe *ast.PipelineExpression) (ast.Expression, []*ast.PipelineExpression) {
	var stages []*ast.PipelineExpression
	leftmost := pe.Left
	for cur := pe; cur != nil; {
		stages = append(stages, cur)
		inner, ok := cur.Left.(*ast.PipelineExpression)
		if !ok {
			leftmost = cur.Left
			break
		}
		cur = inner
	}
	for i, j := 0, len(stages)-1; i < j; i, j = i+1, j-1 {
		stages[i], stages[j] = stages[j], stages[i]
	}
	return leftmost, stages
}

func precOf(op string) int {
	switch op {
	case "||":
		return 1
	case "&&":
		return 2
	case ">":
		return 3
	case "==", "!=":
		return 4
	case "<", "<=", ">=":
		return 5
	case "+", "-":
		return 6
	case "*", "/", "%":
		return 7
	case "**":
		return 8
	case "++":
		return 4
	case ".":
		return 10
	default:
		return 0
	}
}

func countLeadingSpaces(s string) int {
	count := 0
	for _, ch := range s {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}
