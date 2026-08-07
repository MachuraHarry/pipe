package compiler_test

import (
	"testing"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/parser"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

func FuzzCompiler(f *testing.F) {
	f.Add("42")
	f.Add("x: 10\nx")
	f.Add("fn greet name\n    \"Hello, \" ++ name ++ \"!\"\n\ngreet \"World\"")
	f.Add("print (2 + 2)")
	f.Add("if true\n    42\nelse\n    10")
	f.Add("match 3\n    | 1 -> 1\n    | _ -> 0")
	f.Add("fn double x\n    x * 2\n\n42\n    > double")
	f.Add("struct Point: x, y\n\np: Point 10 20\np.x + p.y")
	f.Add("while true\n    break")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("compiler/vm panic on input %q: %v", input, r)
			}
		}()
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return
		}
		c := compiler.New()
		if err := c.Compile(program); err != nil {
			return
		}
		v := vm.New(c.Bytecode())
		v.Run()
	})
}
