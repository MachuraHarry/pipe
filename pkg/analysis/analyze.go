package analysis

import (
	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

// Analyzer walks a Pipe AST and builds the symbol/scope model. The scope rules
// mirror the runtime: only global and function scopes exist; blocks, loops and
// try/catch share the innermost enclosing scope.
type Analyzer struct {
	result   *Analysis
	cur      *Scope
	maxStack []ast.Position // max position seen per open function scope
}

// noteMax records a source position so function-scope ends can be computed.
func (a *Analyzer) noteMax(pos ast.Position) {
	if len(a.maxStack) == 0 {
		return
	}
	top := &a.maxStack[len(a.maxStack)-1]
	if pos.Line > top.Line || (pos.Line == top.Line && pos.Col > top.Col) {
		*top = pos
	}
}

// Analyze parses source and returns the symbol/scope analysis. Parse errors
// are returned as well so callers can produce diagnostics. Docstrings (`--!`
// comments) directly above a definition are attached to the symbol's Doc.
func Analyze(source string) (*Analysis, []string) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	errs := p.Errors()

	a := &Analyzer{
		result: &Analysis{},
	}
	a.result.Root = a.newScope(nil, ScopeGlobal)
	a.cur = a.result.Root

	for _, stmt := range program.Statements {
		a.walkStmt(stmt)
	}

	AttachDocstrings(source, a.result)
	return a.result, errs
}

// AnalyzeProgram analyzes an already-parsed program.
func AnalyzeProgram(program *ast.Program) *Analysis {
	a := &Analyzer{
		result: &Analysis{},
	}
	a.result.Root = a.newScope(nil, ScopeGlobal)
	a.cur = a.result.Root

	for _, stmt := range program.Statements {
		a.walkStmt(stmt)
	}
	return a.result
}

func (a *Analyzer) newScope(parent *Scope, typ ScopeType) *Scope {
	s := &Scope{
		Parent:  parent,
		Type:    typ,
		Symbols: make(map[string]*Symbol),
		End:     ast.Position{Line: -1, Col: -1},
	}
	a.result.Scopes = append(a.result.Scopes, s)
	return s
}

// pushFunction opens a new function scope. Params become symbols in it.
func (a *Analyzer) pushFunction(params []*ast.Identifier, start ast.Position) {
	scope := a.newScope(a.cur, ScopeFunction)
	scope.Start = start
	a.cur = scope
	a.maxStack = append(a.maxStack, ast.Position{Line: -1, Col: -1})
	for _, p := range params {
		if p == nil {
			continue
		}
		a.declare(p.Value, KindParameter, p.Pos(), "")
	}
}

// popFunction closes the current function scope, recording its end position.
func (a *Analyzer) popFunction() {
	if len(a.maxStack) > 0 {
		a.cur.End = a.maxStack[len(a.maxStack)-1]
		a.maxStack = a.maxStack[:len(a.maxStack)-1]
	}
	a.cur = a.cur.Parent
}

func (a *Analyzer) declare(name string, kind SymbolKind, pos ast.Position, doc string) *Symbol {
	sym := &Symbol{
		Name:   name,
		Kind:   kind,
		Pos:    pos,
		End:    endFromPos(pos, len(name)),
		Doc:    doc,
		Usages: []*Reference{},
	}
	a.cur.Symbols[name] = sym
	a.result.Symbols = append(a.result.Symbols, sym)
	a.noteMax(pos)
	return sym
}

// declareOrReuse treats `x: v` as declaration-or-assignment (Pipe semantics):
// a later statement with the same name in the same scope reuses the symbol.
func (a *Analyzer) declareOrReuse(name string, kind SymbolKind, pos ast.Position, doc string) *Symbol {
	if existing, ok := a.cur.Symbols[name]; ok {
		return existing
	}
	return a.declare(name, kind, pos, doc)
}

func (a *Analyzer) reference(name string, pos ast.Position) {
	end := endFromPos(pos, len(name))
	a.noteMax(end)
	ref := &Reference{Name: name, Pos: pos, End: end}

	if sym := a.cur.Lookup(name); sym != nil {
		ref.Symbol = sym
		sym.Usages = append(sym.Usages, ref)
	} else if name == "_" {
		// wildcard: not a symbol
		return
	} else if bsym := builtinSymbol(name); bsym != nil {
		ref.Symbol = bsym
	} else {
		a.result.Unresolved = append(a.result.Unresolved, ref)
	}
	a.result.References = append(a.result.References, ref)
}

// ---- statements ----

func (a *Analyzer) walkStmt(stmt ast.Statement) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *ast.FnStatement:
		a.walkFnStatement(s)
	case *ast.VarStatement:
		a.walkExpr(s.Value)
		a.declareOrReuse(s.Name.Value, KindVariable, s.Name.Pos(), "")
	case *ast.ExpressionStatement:
		a.walkExpr(s.Expression)
	case *ast.ReturnStatement:
		a.walkExpr(s.Value)
	case *ast.DeferStatement:
		a.walkExpr(s.Expression)
	case *ast.ImportStatement:
		a.walkImport(s)
	case *ast.ExportStatement:
		a.walkExport(s)
	case *ast.EnumStatement:
		a.walkEnum(s)
	case *ast.TestStatement:
		a.walkBlock(s.Body)
	case *ast.BlockStatement:
		a.walkBlock(s)
	case *ast.BreakStatement, *ast.ContinueStatement:
		// nothing
	}
}

func (a *Analyzer) walkBlock(block *ast.BlockStatement) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		a.walkStmt(stmt)
	}
}

func (a *Analyzer) walkFnStatement(fn *ast.FnStatement) {
	if fn.Name != nil {
		a.declare(fn.Name.Value, KindFunction, fn.Name.Pos(), "")
	}
	a.pushFunction(fn.Parameters, fn.Name.Pos())
	a.walkBlock(fn.Body)
	a.popFunction()
}

func (a *Analyzer) walkImport(imp *ast.ImportStatement) {
	name := imp.Alias
	if name == "" {
		name = moduleNameFromPath(imp.Path)
	}
	if name != "" {
		a.declare(name, KindModule, ast.Position{Line: imp.Line, Col: imp.Col}, "import \""+imp.Path+"\"")
	}
}

func (a *Analyzer) walkExport(ex *ast.ExportStatement) {
	if ex.Fn != nil {
		a.walkFnStatement(ex.Fn)
	}
	if ex.Var != nil {
		a.walkExpr(ex.Var.Value)
		a.declareOrReuse(ex.Var.Name.Value, KindVariable, ex.Var.Name.Pos(), "")
	}
	if ex.Enum != nil {
		a.walkEnum(ex.Enum)
	}
}

func (a *Analyzer) walkEnum(en *ast.EnumStatement) {
	pos := ast.Position{Line: en.Line, Col: en.Col}
	a.declare(en.Name, KindEnum, pos, "")
	for i, v := range en.Values {
		vpos := pos
		if i < len(en.ValuePos) {
			vpos = en.ValuePos[i]
		}
		a.declare(v, KindEnumMember, vpos, "member of "+en.Name)
	}
}

// moduleNameFromPath derives an identifier from an import path
// (e.g. "log-analyzer@1.0.0" -> "log-analyzer").
func moduleNameFromPath(path string) string {
	name := path
	if i := indexByte(name, '@'); i >= 0 {
		name = name[:i]
	}
	if i := lastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ---- expressions ----

func (a *Analyzer) walkExpr(expr ast.Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.Identifier:
		a.reference(e.Value, e.Pos())
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral,
		*ast.BooleanLiteral, *ast.NilLiteral:
		// nothing
	case *ast.PrefixExpression:
		a.walkExpr(e.Right)
	case *ast.InfixExpression:
		a.walkExpr(e.Left)
		a.walkExpr(e.Right)
	case *ast.PipelineExpression:
		a.walkExpr(e.Left)
		a.walkExpr(e.Right)
	case *ast.CallExpression:
		a.walkExpr(e.Function)
		for _, arg := range e.Arguments {
			a.walkExpr(arg)
		}
	case *ast.ListLiteral:
		for _, el := range e.Elements {
			a.walkExpr(el)
		}
	case *ast.MapLiteral:
		for _, v := range e.Pairs {
			a.walkExpr(v)
		}
	case *ast.DotExpression:
		a.walkExpr(e.Left)
	case *ast.IfExpression:
		a.walkExpr(e.Condition)
		a.walkBlock(e.Consequence)
		a.walkBlock(e.Alternative)
	case *ast.MatchExpression:
		a.walkExpr(e.Value)
		for _, c := range e.Cases {
			if id, ok := c.Pattern.(*ast.Identifier); ok && id.Value == "_" {
				continue
			}
			a.walkExpr(c.Pattern)
			a.walkExpr(c.Body)
		}
	case *ast.WhileExpression:
		a.walkExpr(e.Condition)
		a.walkBlock(e.Body)
	case *ast.ForExpression:
		a.walkFor(e)
	case *ast.FnLiteral:
		a.pushFunction(e.Parameters, e.Pos())
		a.walkBlock(e.Body)
		a.popFunction()
	case *ast.SliceExpression:
		a.walkExpr(e.List)
		a.walkExpr(e.Start)
		a.walkExpr(e.End)
	case *ast.TryExpression:
		a.walkBlock(e.TryBlock)
		if e.CatchParam != nil {
			a.declareOrReuse(e.CatchParam.Value, KindParameter, e.CatchParam.Pos(), "")
		}
		a.walkBlock(e.CatchBlock)
	}
}

func (a *Analyzer) walkFor(fe *ast.ForExpression) {
	if fe.IsForIn {
		a.walkExpr(fe.Iterable)
		if fe.Iterator != nil {
			a.declareOrReuse(fe.Iterator.Value, KindVariable, fe.Iterator.Pos(), "")
		}
		a.walkBlock(fe.Body)
		return
	}
	a.walkStmt(fe.Init)
	a.walkExpr(fe.Condition)
	a.walkStmt(fe.Update)
	a.walkBlock(fe.Body)
}
