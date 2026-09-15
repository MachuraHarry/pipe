package eval

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

// FuzzClosureScoping is a generative differential fuzzer, not a robustness
// fuzzer like pkg/lexer|parser|compiler's FuzzXxx (which only check "does
// not panic" on arbitrary byte soup). It builds syntactically valid,
// semantically varied Pipe programs -- nested functions mixing shadow-
// mutate assignments (`x: x + 1`) and self-recursive definitions at
// different nesting depths and with different variable-name reuse -- and
// asserts the tree-walker and VM produce byte-identical output for each.
//
// This area was the richest bug source this session by far: closure
// mutable state, a self-shadowing read reading an uninitialized VM local,
// and a self-recursive LOCAL closure crashing the VM outright were all
// found here by hand. The generator targets exactly this surface,
// mechanically exploring nesting-depth and variable-reuse combinations a
// human wouldn't think to try one by one.
//
// A generated program is valid Pipe by construction (fixed indentation
// tracked by genBuilder, fixed statement shapes) -- a parse or compile
// error is itself a finding worth investigating, not something to skip.
// The one expected, accepted divergence -- sibling closures sharing a
// captured variable (pkg/vm's TestSiblingClosuresDoNotShareCapturedState)
// -- can't occur here: this generator only ever builds a single linear
// nesting chain, never two sibling closures over the same variable.
func FuzzClosureScoping(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3})
	f.Add([]byte{3, 1, 0, 2, 1, 3, 0})
	f.Add([]byte{2, 2, 2, 2, 2, 2, 2, 2})
	f.Add([]byte{0, 3, 1, 2, 3, 0, 1, 2, 3, 1})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 || len(data) > 64 {
			t.Skip("empty or oversized seed")
		}
		g := &genState{data: data}
		src := g.genProgram()

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

// genEvalResult runs program through the tree-walker and renders its
// result the same way genVMResult does for the VM (a plain Inspect(),
// or an ERROR: prefix for a runtime error), so the two are directly
// string-comparable.
func genEvalResult(program *ast.Program) string {
	ctx := NewEvalContext("")
	env := object.NewEnvironment()
	result := ctx.Eval(program, env)
	if result == nil {
		return "NIL_RESULT"
	}
	if result.Type() == object.ERROR {
		return "ERROR: " + result.Inspect()
	}
	return result.Inspect()
}

// genState is a byte-consuming cursor driving program construction. Each
// generator method reads a small, bounded number of bytes from the fuzz
// input to make a structural choice (which shape to emit, which variable
// name to reuse); running out of bytes deterministically falls back to 0,
// so every input (including the empty slice) produces a valid, terminating
// program rather than panicking or hanging.
type genState struct {
	data []byte
	pos  int
}

func (g *genState) byte() byte {
	if g.pos >= len(g.data) {
		return 0
	}
	b := g.data[g.pos]
	g.pos++
	return b
}

// pick returns a value in [0,n) derived from the next input byte.
func (g *genState) pick(n int) int {
	if n <= 0 {
		return 0
	}
	return int(g.byte()) % n
}

var genVarPool = []string{"x", "y", "n", "count"}

// genBuilder accumulates generated source with indentation tracking, since
// Pipe's grammar is indentation-sensitive like Python's.
type genBuilder struct {
	sb strings.Builder
}

func (b *genBuilder) line(indent int, format string, args ...interface{}) {
	b.sb.WriteString(strings.Repeat("    ", indent))
	fmt.Fprintf(&b.sb, format, args...)
	b.sb.WriteString("\n")
}

// genProgram builds one of a fixed set of statement shapes at each of 1-3
// nesting levels: a shadow-mutate assignment (`x: x + 1`, the exact
// pattern behind two bugs fixed this session: a VM read of an
// uninitialized local when the target didn't already exist outward, and
// correct write-through when it did), a fresh local declaration, or --
// only at the innermost level, since self-recursion is a leaf pattern in
// this generator -- a self-recursive counting function (the pattern
// behind the OpCurrentClosure crash fix). The outer function is called
// 1-3 times and each result collected into a list, which both backends
// must render identically.
func (g *genState) genProgram() string {
	depth := 1 + g.pick(3) // 1..3 nesting levels

	var b genBuilder
	// A couple of global bindings so inner levels have something real to
	// shadow, not just undefined names.
	numGlobals := 1 + g.pick(2)
	known := map[string]bool{}
	for i := 0; i < numGlobals; i++ {
		name := genVarPool[g.pick(len(genVarPool))]
		b.line(0, "%s: %d", name, g.pick(5))
		known[name] = true
	}

	b.line(0, "fn f0")
	g.genLevel(&b, 1, depth, 1, known)

	calls := 1 + g.pick(3)
	b.line(0, "results: []")
	for i := 0; i < calls; i++ {
		b.line(0, "push results (f0())")
	}
	b.line(0, "results")
	return b.sb.String()
}

// genLevel emits the body of nesting level `level` (1-based) at the given
// indent, recursing into level+1 until `depth` is reached, where it emits
// a leaf: either a plain literal return or a self-recursive counting
// function exercising OpCurrentClosure. `known` is every variable name
// bound so far (globals plus outer levels), passed by value so each
// branch's additions stay local to it, not leak sideways to a sibling
// branch that never ran.
func (g *genState) genLevel(b *genBuilder, level, depth, indent int, known map[string]bool) {
	knownNames := make([]string, 0, len(known))
	for n := range known {
		knownNames = append(knownNames, n)
	}
	// Deterministic order: map iteration order is randomized in Go, which
	// would make the same fuzz input produce different programs between
	// runs and break corpus reproducibility.
	sort.Strings(knownNames)

	var name string
	choice := g.pick(3)
	if choice == 0 && len(knownNames) > 0 {
		// Shadow-mutate an already-bound name: reassigns it if found in an
		// enclosing scope (the fixed self-referential-shadow-read bug),
		// never introduces a genuinely undefined identifier (that's a
		// separate, already-understood TW-runtime-vs-VM-compile-time
		// error-timing difference, not a semantic divergence worth
		// rediscovering here on every run).
		name = knownNames[g.pick(len(knownNames))]
		b.line(indent, "%s: %s + 1", name, name)
	} else if choice == 1 {
		// Fresh local, deliberately shadowing any outer binding of the
		// same name with a NEW value rather than reading+incrementing it.
		name = genVarPool[g.pick(len(genVarPool))]
		b.line(indent, "%s: %d", name, 10+g.pick(5))
		known[name] = true
	} else {
		// A nested nop assignment to a different (possibly fresh) name,
		// just to vary shape.
		name = genVarPool[g.pick(len(genVarPool))]
		b.line(indent, "%s: %d", name, g.pick(3))
		known[name] = true
	}

	if level >= depth {
		g.genLeaf(b, indent, name)
		return
	}

	b.line(indent, "fn f%d", level)
	g.genLevel(b, level+1, depth, indent+1, known)
	b.line(indent, "f%d()", level)
}

// genLeaf emits the innermost level's return value: either the tracked
// variable itself, or a self-recursive counting function called with a
// small fixed argument -- the pattern that crashed the VM before
// OpCurrentClosure (a local self-recursive `name: fn n: ...` nested inside
// another function).
func (g *genState) genLeaf(b *genBuilder, indent int, outerVar string) {
	if g.pick(2) == 0 {
		b.line(indent, "%s", outerVar)
		return
	}

	fnName := "rec"
	arg := 2 + g.pick(4) // 2..5, keeps recursion shallow and fast
	b.line(indent, "%s: fn n", fnName)
	b.line(indent+1, "if n <= 1")
	b.line(indent+2, "1")
	b.line(indent+1, "else")
	b.line(indent+2, "n * %s(n - 1)", fnName)
	b.line(indent, "%s(%s)", fnName, strconv.Itoa(arg))
}
