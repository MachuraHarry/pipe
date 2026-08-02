package gen

import (
	"math/rand"
	"sort"

	"github.com/harry/pipe/pkg/ast"
)

type Generator struct {
	rng *rand.Rand
	ctx *context
	opts GenOptions
	w    weights
}

func New() *Generator {
	return &Generator{}
}

func Generate(opts GenOptions) *ast.Program {
	g := &Generator{
		rng:  rand.New(rand.NewSource(opts.Seed)),
		opts: opts,
		w:    defaultWeights(),
	}
	return g.gen(opts)
}

func (g *Generator) gen(opts GenOptions) *ast.Program {
	g.ctx = newContext(opts.Seed)
	g.ctx.pushScope()
	g.ctx.addBuiltins()

	numStmts := randInt(g.rng, 2, opts.MaxStmts+1)
	stmts := make([]ast.Statement, 0, numStmts)

	for i := 0; i < numStmts; i++ {
		stmt := g.genTopStmt()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	return &ast.Program{Statements: stmts}
}

func (g *Generator) genTopStmt() ast.Statement {
	g.ctx.loopDepth = 0

	total := g.w.fnDef + g.w.varDef + g.w.enumDef + g.w.exprStmt
	if total <= 0 {
		return nil
	}
	r := g.rng.Intn(total)
	if r < g.w.fnDef {
		return g.genFnDef()
	}
	r -= g.w.fnDef
	if r < g.w.varDef {
		return g.genVarDef()
	}
	r -= g.w.varDef
	if r < g.w.enumDef {
		return g.genEnumDef()
	}
	return g.genExprStmt()
}

func (g *Generator) pickFromScope(kind SymbolKind) (string, ScopeEntry, bool) {
	name := g.pickNameFromScope(kind)
	if name == "" {
		return "", ScopeEntry{}, false
	}
	entry, _ := g.ctx.resolve(name)
	return name, entry, true
}

func (g *Generator) pickFunc() (string, ScopeEntry, bool) {
	names := []string{}
	for _, s := range g.ctx.scopes {
		for name, entry := range s {
			if entry.Kind == KindFunction || entry.Kind == KindBuiltin {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return "", ScopeEntry{}, false
	}
	sort.Strings(names)
	n := g.rng.Intn(len(names))
	entry, _ := g.ctx.resolve(names[n])
	return names[n], entry, true
}

func (g *Generator) pickNameFromScope(kind SymbolKind) string {
	var names []string
	for _, s := range g.ctx.scopes {
		for name, entry := range s {
			if entry.Kind == kind {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return names[g.rng.Intn(len(names))]
}
