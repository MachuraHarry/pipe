//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/harry/pipe/pkg/ai"
	"github.com/harry/pipe/pkg/eval"
	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/object"
	"github.com/harry/pipe/pkg/parser"
)

var outputBuf strings.Builder
var apiKey string
var apiProvider string

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

	ai.SetTimeout(15)
}

func pipeSetKey(this js.Value, args []js.Value) interface{} {
	provider := args[0].String()
	key := args[1].String()
	apiProvider = provider
	apiKey = key
	ai.SetProvider(provider)
	ai.ActiveConfig.APIKey = key
	return nil
}

func pipeRun(this js.Value, args []js.Value) interface{} {
	code := args[0].String()
	outputBuf.Reset()

	// Debug: show key status
	outputBuf.WriteString("[WASM] Provider: " + ai.ActiveConfig.Provider)
	outputBuf.WriteString(", Key: " + fmt.Sprintf("%d chars", len(ai.ActiveConfig.APIKey)) + "\n\n")

	if apiProvider != "" {
		code = "ai_provider \"" + apiProvider + "\"\n" + code
	}

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

func main() {
	js.Global().Set("pipeRun", js.FuncOf(pipeRun))
	js.Global().Set("pipeSetKey", js.FuncOf(pipeSetKey))
	js.Global().Set("pipeVersion", js.ValueOf("v0.5.0"))
	select {}
}
