package eval

import (
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/parser"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

// FuzzRuntimeErrors is FuzzClosureScoping's sibling for a different bug
// surface: runtime error conditions (division/modulo by zero, list index
// out of bounds, arithmetic type mismatches) with and without try/catch
// recovery, checking the tree-walker and VM raise the byte-identical error
// (same code, same message) or byte-identical recovered value.
//
// Deliberately excluded: undefined-variable errors (E001). Those have a
// known, accepted, understood timing difference -- the tree-walker only
// finds a truly-undefined identifier at runtime, while the VM (correctly,
// see the compiler/vm crash-fix commit this session) rejects it at compile
// time -- which isn't a semantic divergence, just a different, still-
// deterministic point of failure. This generator only ever references
// variables it has itself just declared, so that class never comes up.
func FuzzRuntimeErrors(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 2})
	f.Add([]byte{3, 1, 0, 2, 3, 1, 0})
	f.Add([]byte{2, 2, 0, 0, 2, 2})
	f.Add([]byte{0, 1, 2, 3, 0, 1, 2, 3})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 || len(data) > 32 {
			t.Skip("empty or oversized seed")
		}
		g := &genState{data: data}
		src := g.genErrorProgram()

		l := lexer.New(src)
		p := parser.New(l)
		program := p.ParseProgram()
		if errs := p.Errors(); len(errs) > 0 {
			t.Fatalf("generated program failed to parse (generator bug, not a language bug):\n%s\n\nerrors:\n%s",
				src, strings.Join(errs, "\n"))
		}

		evalOut := genEvalResult(program)

		c := compiler.New()
		var vmOut string
		if err := c.Compile(program); err != nil {
			vmOut = "COMPILE_ERROR: " + err.Error()
		} else {
			v := vm.New(c.Bytecode())
			if err := v.Run(); err != nil {
				vmOut = "ERROR: " + err.Error()
			} else {
				vmOut = v.LastPoppedStackElem().Inspect()
			}
		}

		if evalOut != vmOut {
			t.Fatalf("tree-walker/VM divergence on generated program:\n%s\n\neval: %s\nvm:   %s", src, evalOut, vmOut)
		}
	})
}

// genErrorProgram builds a small program with a few known-typed globals
// (an int that may be 0, a short list, a string) and one risky expression
// combining two of them with an operator or index that may legitimately
// fail at runtime (division/modulo by zero, out-of-bounds index, a
// string-plus-int type mismatch) -- sometimes bare (the error becomes the
// program's result, matching how the existing crossmode tests treat a
// top-level error), sometimes wrapped in try/catch (recovered into a
// fixed string both engines must agree the catch actually ran).
func (g *genState) genErrorProgram() string {
	var b genBuilder

	n := g.pick(3) // 0, 1, or 2 -- 0 deliberately included to trigger div/mod-by-zero and index-out-of-bounds
	b.line(0, "num: %d", n)
	b.line(0, "items: [10, 20]")
	b.line(0, "word: \"hi\"")

	riskyExprs := []string{
		"100 / num",
		"100 % num",
		"items[num + 5]",
		"word ++ num",
		"num + word",
	}
	expr := riskyExprs[g.pick(len(riskyExprs))]

	if g.pick(2) == 0 {
		b.line(0, "try")
		b.line(1, "%s", expr)
		b.line(0, "catch e")
		b.line(1, "\"recovered\"")
	} else {
		b.line(0, "%s", expr)
	}

	return b.sb.String()
}
