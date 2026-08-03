package gen

import (
	"math/rand"
	"sort"

	"github.com/MachuraHarry/pipe/pkg/ast"
)

type Generator struct {
	rng  *rand.Rand
	ctx  *context
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

	numStmts := randInt(g.rng, 3, opts.MaxStmts+1)
	stmts := make([]ast.Statement, 0, numStmts+2)

	addedPrint := false

	for i := 0; i < numStmts; i++ {
		if i == numStmts-1 {
			if !addedPrint && g.rng.Intn(2) == 0 {
				stmts = append(stmts, g.genPrintStmt())
				addedPrint = true
				continue
			}
		}
		stmt := g.genTopStmt()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	if !addedPrint {
		stmts = append(stmts, g.genPrintStmt())
	}

	return &ast.Program{Statements: stmts}
}

func (g *Generator) genPrintStmt() ast.Statement {
	target := g.genPrintTarget()
	return &ast.ExpressionStatement{
		Expression: &ast.CallExpression{
			Function:  &ast.Identifier{Value: "print"},
			Arguments: []ast.Expression{target},
		},
	}
}

func (g *Generator) genPrintTarget() ast.Expression {
	switch g.rng.Intn(10) {
	case 0, 1, 2:
		if name, _, _ := g.pickFromScope(KindVariable); name != "" {
			return &ast.Identifier{Value: name}
		}
		return g.genExprNoPipeline(TypeAny, 3)
	case 3:
		return &ast.InfixExpression{
			Operator: "++",
			Left:     &ast.StringLiteral{Value: demoStrings[g.rng.Intn(len(demoStrings))]},
			Right:    g.genLeaf(TypeString),
		}
	case 4:
		return &ast.CallExpression{
			Function:  &ast.Identifier{Value: "to_str"},
			Arguments: []ast.Expression{g.genExprNoPipeline(TypeAny, 2)},
		}
	case 5:
		return &ast.CallExpression{
			Function:  &ast.Identifier{Value: "to_json"},
			Arguments: []ast.Expression{g.genExprNoPipeline(TypeAny, 2)},
		}
	case 6:
		return &ast.CallExpression{
			Function:  &ast.Identifier{Value: "len"},
			Arguments: []ast.Expression{g.genList(3)},
		}
	case 7:
		return &ast.StringLiteral{Value: demoStrings[g.rng.Intn(len(demoStrings))]}
	case 8:
		return &ast.CallExpression{
			Function:  &ast.Identifier{Value: "type_of"},
			Arguments: []ast.Expression{g.genExprNoPipeline(TypeAny, 2)},
		}
	default:
		if name, _, _ := g.pickFromScope(KindVariable); name != "" {
			return &ast.Identifier{Value: name}
		}
		return g.genLiteral(TypeInt)
	}
}

var demoStrings = []string{
	"Hello Pipe!", "Hi there!", "Result:", "Done!", "OK",
	"Sum =", "Max =", "Count =", "Value:", "Name:",
	"Processing…", "✓", "---", "=======", ">",
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
