//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"time"

	"github.com/MachuraHarry/pipe/pkg/eval"
	"github.com/MachuraHarry/pipe/pkg/gen"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

var outputBuf strings.Builder

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

func main() {
	object.SetSandbox(true)
	js.Global().Set("pipeRun", js.FuncOf(pipeRun))
	js.Global().Set("pipeGenerate", js.FuncOf(pipeGenerate))
	js.Global().Set("pipeVersion", js.ValueOf("v0.7.0"))
	select {}
}
