//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/eval"
	"github.com/MachuraHarry/pipe/pkg/gen"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

var outputBuf strings.Builder

var aiBuiltinNames = map[string]bool{
	"ai_provider": true, "ai_model": true, "ai_timeout": true, "ai_host": true, "ai_cache": true, "ai_set_key": true,
	"web_search": true, "wiki_search": true,
	"ai_chat": true, "ai_chat_json": true,
	"summarize": true, "translate": true, "classify": true, "extract": true, "generate": true, "generate_json": true, "ask": true,
	"ai_stream":   true,
	"ai_parallel": true, "ai_batch": true, "ai_rate_limit": true,
	"embed": true, "embed_batch": true, "cosine_sim": true, "dot_product": true, "nearest": true,
	"ai_tool": true, "ai_with_tools": true,
	"agent": true, "agent_ask": true, "agent_clear": true, "try_ai_log": true,
}

var sandboxNames = map[string]bool{
	"sandbox_profile": true, "set_sandbox": true, "with_sandbox": true,
}

var hofNames = map[string]bool{
	"map": true, "filter": true, "reduce": true, "each": true,
}

var ioBuiltins = map[string]bool{
	"read_file": true, "write_file": true, "append_file": true, "save": true,
	"read_lines": true, "http_get": true, "http_post": true, "print": true,
}

func init() {
	object.PrintHook = func(args ...object.Object) {
		for i, arg := range args {
			if i > 0 {
				outputBuf.WriteByte(' ')
			}
			outputBuf.WriteString(arg.Inspect())
		}
		outputBuf.WriteByte('\n')
	}
}

func pipeRun(this js.Value, args []js.Value) interface{} {
	code := args[0].String()
	outputBuf.Reset()

	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	if errs := p.Errors(); len(errs) > 0 {
		result := "Parse errors:\n"
		for _, e := range errs {
			result += "  " + e + "\n"
		}
		return result
	}

	env := object.NewEnvironment()
	ctx := eval.NewEvalContext("<playground>")
	result := ctx.Eval(program, env)
	if result != nil && result.Type() == object.ERROR {
		outputBuf.WriteString("Error: " + result.Inspect() + "\n")
	}
	return outputBuf.String()
}

func pipeGenerate(this js.Value, args []js.Value) interface{} {
	opts := gen.DefaultOptions()
	opts.Seed = time.Now().UnixNano()
	opts.MaxStmts = 10

	prog, src, err := gen.GenerateValid(opts)
	outputBuf.Reset()
	if err != nil {
		return "-- gen v2: " + err.Error()
	}
	_ = prog
	return "-- gen v2: runtime-valid\n" + src
}

func pipeSetKey(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return "ai_set_key expects 2 arguments (provider, key)"
	}
	provider := args[0].String()
	key := args[1].String()
	ai.SetAPIKey(provider, key)
	return "key set for " + provider
}

func pipeParse(this js.Value, args []js.Value) interface{} {
	code := args[0].String()
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	errs := p.Errors()

	var sb strings.Builder
	sb.WriteString(`{"pipelines":[`)

	first := true
	for _, stmt := range program.Statements {
		pipes := extractPipelineChains(stmt)
		for _, pipe := range pipes {
			stages := flattenPipeline(pipe)
			if len(stages) == 0 {
				continue
			}
			if !first {
				sb.WriteByte(',')
			}
			first = false
			sb.WriteString(`{"stages":[`)
			for i, s := range stages {
				if i > 0 {
					sb.WriteByte(',')
				}
				writeStageJSON(&sb, s)
			}
			sb.WriteString(`]}`)
		}
	}

	sb.WriteString(`],"errors":[`)
	for i, e := range errs {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(jsonQuote(e))
	}
	sb.WriteString(`]}`)

	return sb.String()
}

func extractPipelineChains(stmt ast.Statement) []*ast.PipelineExpression {
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		if pipe, ok := s.Expression.(*ast.PipelineExpression); ok {
			return []*ast.PipelineExpression{pipe}
		}
	case *ast.VarStatement:
		if pipe, ok := s.Value.(*ast.PipelineExpression); ok {
			return []*ast.PipelineExpression{pipe}
		}
	}
	return nil
}

type pipelineSegment struct {
	Fn       *ast.CallExpression
	Parallel bool
	Source   ast.Expression
	Line     int
}

func flattenPipeline(pipe *ast.PipelineExpression) []pipelineSegment {
	var rightStages []pipelineSegment
	var leftMost ast.Expression

	current := pipe
	for current != nil {
		parallel := current.Parallel
		fn := extractCall(current.Right)
		if fn != nil {
			rightStages = append(rightStages, pipelineSegment{
				Fn:       fn,
				Parallel: parallel,
				Line:     0,
			})
		} else {
			rightStages = append(rightStages, pipelineSegment{
				Fn:       nil,
				Parallel: parallel,
				Source:   current.Right,
				Line:     0,
			})
		}

		if next, ok := current.Left.(*ast.PipelineExpression); ok {
			current = next
		} else {
			leftMost = current.Left
			current = nil
		}
	}

	var stages []pipelineSegment

	if leftMost != nil {
		srcFn := extractCall(leftMost)
		stages = append(stages, pipelineSegment{
			Fn:     srcFn,
			Source: leftMost,
			Line:   0,
		})
	}

	for i := len(rightStages) - 1; i >= 0; i-- {
		stages = append(stages, rightStages[i])
	}

	return stages
}

func extractCall(expr ast.Expression) *ast.CallExpression {
	if call, ok := expr.(*ast.CallExpression); ok {
		return call
	}
	return nil
}

func classifyFunc(name string) string {
	if len(name) == 0 {
		return "value"
	}
	if aiBuiltinNames[name] {
		return "ai"
	}
	if sandboxNames[name] {
		return "sandbox"
	}
	if hofNames[name] {
		return "hof"
	}
	if ioBuiltins[name] {
		return "io"
	}
	first := name[0]
	if first >= 'a' && first <= 'z' {
		return "builtin"
	}
	return "function"
}

func writeStageJSON(sb *strings.Builder, s pipelineSegment) {
	id := ""
	label := ""
	typ := "value"
	var args []string

	if s.Fn != nil {
		if ident, ok := s.Fn.Function.(*ast.Identifier); ok {
			name := ident.Value
			id = name
			label = name
			typ = classifyFunc(name)
			for _, arg := range s.Fn.Arguments {
				args = append(args, arg.String())
			}
		} else {
			label = s.Fn.Function.String()
			id = label
			typ = "function"
		}
	} else if s.Source != nil {
		label = s.Source.String()
		id = label
		switch s.Source.(type) {
		case *ast.StringLiteral:
			typ = "string"
		case *ast.IntegerLiteral, *ast.FloatLiteral:
			typ = "number"
		case *ast.Identifier:
			name := s.Source.(*ast.Identifier).Value
			id = name
			label = name
			typ = "variable"
		case *ast.CallExpression:
			if ident, ok := s.Source.(*ast.CallExpression).Function.(*ast.Identifier); ok {
				name := ident.Value
				id = name
				label = name
				typ = classifyFunc(name)
			} else {
				typ = "expression"
			}
		default:
			typ = "expression"
		}
	}

	sb.WriteString(`{"id":`)
	sb.WriteString(jsonQuote(id))
	sb.WriteString(`,"label":`)
	sb.WriteString(jsonQuote(label))
	sb.WriteString(`,"type":`)
	sb.WriteString(jsonQuote(typ))
	sb.WriteString(`,"args":[`)
	for i, a := range args {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(jsonQuote(a))
	}
	sb.WriteString(`],"parallel":`)
	if s.Parallel {
		sb.WriteString("true")
	} else {
		sb.WriteString("false")
	}
	sb.WriteString(`,"line":`)
	sb.WriteString(itoa(s.Line))
	sb.WriteByte('}')
}

func jsonQuote(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+(n%10))) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func main() {
	object.SetSandbox(true)
	object.SetSandboxAllowAI(true)
	object.SetSandboxAllowNet(true)
	js.Global().Set("pipeRun", js.FuncOf(pipeRun))
	js.Global().Set("pipeGenerate", js.FuncOf(pipeGenerate))
	js.Global().Set("pipeSetKey", js.FuncOf(pipeSetKey))
	js.Global().Set("pipeParse", js.FuncOf(pipeParse))
	js.Global().Set("pipeVersion", js.ValueOf("v0.9.4.0"))
	select {}
}
