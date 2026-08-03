package gen

import (
	"fmt"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/eval"
	"github.com/MachuraHarry/pipe/pkg/formatter"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

const maxValidateTries = 500

func GenerateCompilable(opts GenOptions) (*ast.Program, string, error) {
	for try := 0; try < maxValidateTries; try++ {
		seed := opts.Seed + int64(try*1000)
		prog := tryGenerate(seed, opts)
		src := formatter.FormatProgram(prog)

		if !checkParse(src) {
			continue
		}

		comp := compiler.New()
		if err := comp.Compile(prog); err != nil {
			continue
		}
		return prog, src, nil
	}
	return nil, "", fmt.Errorf("failed to generate compilable program after %d tries", maxValidateTries)
}

func GenerateValid(opts GenOptions) (*ast.Program, string, error) {
	var lastSrc string
	for try := 0; try < maxValidateTries; try++ {
		seed := opts.Seed + int64(try*1000)
		prog := tryGenerate(seed, opts)
		src := formatter.FormatProgram(prog)

		if !checkFull(prog, src) {
			lastSrc = src
			continue
		}
		return prog, src, nil
	}
	return nil, "", fmt.Errorf("failed to generate valid program after %d tries, last:\n%s", maxValidateTries, lastSrc)
}

func tryGenerate(seed int64, opts GenOptions) *ast.Program {
	opts.Seed = seed
	return Generate(opts)
}

func checkParse(src string) bool {
	l := lexer.New(src)
	p := parser.New(l)
	_ = p.ParseProgram()
	return len(p.Errors()) == 0
}

func checkCompile(prog *ast.Program) bool {
	comp := compiler.New()
	return comp.Compile(prog) == nil
}
func checkFull(prog *ast.Program, src string) bool {
	if !checkParse(src) {
		return false
	}
	comp := compiler.New()
	if err := comp.Compile(prog); err != nil {
		return false
	}
	bc := comp.Bytecode()
	machine := vm.New(bc)

	oldHook := object.PrintHook
	object.PrintHook = func(args ...object.Object) {}
	defer func() { object.PrintHook = oldHook }()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		done <- machine.Run()
	}()
	select {
	case err := <-done:
		if err != nil {
			return false
		}
	case <-time.After(2 * time.Second):
		return false
	}

	res := machine.LastPoppedStackElem()
	if res != nil && res.Type() == object.ERROR {
		return false
	}

	env := object.NewEnvironment()
	evalCtx := eval.NewEvalContext("<gen>")
	evalRes := evalCtx.Eval(prog, env)
	if evalRes != nil && evalRes.Type() == object.ERROR {
		return false
	}

	return true
}
