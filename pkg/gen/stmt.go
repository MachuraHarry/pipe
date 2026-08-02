package gen

import (
	"github.com/harry/pipe/pkg/ast"
)

func (g *Generator) genFnDef() ast.Statement {
	name := g.ctx.names.nextFn()

	g.ctx.pushScope()

	nParams := g.rng.Intn(4)
	params := make([]*ast.Identifier, nParams)
	for i := 0; i < nParams; i++ {
		pName := g.ctx.names.paramName(i)
		params[i] = &ast.Identifier{Value: pName}
		g.ctx.define(pName, ScopeEntry{Kind: KindParam, Type: TypeAny})
	}

	body := g.genFnBody()

	g.ctx.popScope()

	arity := nParams
	g.ctx.define(name, ScopeEntry{Kind: KindFunction, Type: TypeFn, Arity: arity})

	return &ast.FnStatement{
		Name:       &ast.Identifier{Value: name},
		Parameters: params,
		Body:       body,
	}
}

func (g *Generator) genFnBody() *ast.BlockStatement {
	block := &ast.BlockStatement{}
	g.ctx.fnDepth++
	nStmts := randInt(g.rng, 1, 6)
	for i := 0; i < nStmts; i++ {
		stmt := g.genFnStmt()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
	}
	g.ctx.fnDepth--
	return block
}

func (g *Generator) genFnStmt() ast.Statement {
	r := g.rng.Intn(10)
	switch {
	case r < 4:
		return g.genVarDef()
	case r < 7:
		return g.genExprStmt()
	case r < 8 && g.ctx.fnDepth > 0:
		return g.genReturnStmt()
	case r < 9:
		return g.genIfStmt()
	default:
		return g.genExprStmt()
	}
}

func (g *Generator) genReturnStmt() ast.Statement {
	return &ast.ReturnStatement{Value: g.genExprNoPipeline(TypeAny, g.opts.MaxDepth)}
}

func (g *Generator) genVarDef() ast.Statement {
	name := g.ctx.names.nextVar()
	value := g.genExprNoPipeline(TypeAny, g.opts.MaxDepth)
	g.ctx.define(name, ScopeEntry{Kind: KindVariable, Type: TypeAny})
	return &ast.VarStatement{
		Name:  &ast.Identifier{Value: name},
		Value: value,
	}
}

func (g *Generator) genEnumDef() ast.Statement {
	name := g.ctx.names.nextFn()
	nVals := randInt(g.rng, 2, 5)
	values := make([]string, nVals)
	prefixes := []string{"Red", "Blue", "Green", "Small", "Large", "High", "Low", "On", "Off", "Open", "Closed", "Active", "Inactive"}
	for i := 0; i < nVals; i++ {
		v := prefixes[g.rng.Intn(len(prefixes))]
		values[i] = v
		g.ctx.define(v, ScopeEntry{Kind: KindEnumValue, Type: TypeInt})
	}
	return &ast.EnumStatement{
		Name:   name,
		Values: values,
	}
}

func (g *Generator) genExprStmt() ast.Statement {
	expr := g.genTopExpr(TypeAny, g.opts.MaxDepth)
	return &ast.ExpressionStatement{Expression: expr}
}

func (g *Generator) genIfStmt() ast.Statement {
	cond := g.genExpr(TypeBool, g.opts.MaxDepth)
	cons := g.genExpr(TypeAny, g.opts.MaxDepth)

	ifExpr := &ast.IfExpression{
		Condition: cond,
		Consequence: &ast.BlockStatement{
			Statements: []ast.Statement{&ast.ExpressionStatement{Expression: cons}},
		},
	}

	if g.rng.Intn(3) == 0 {
		alt := g.genExpr(TypeAny, g.opts.MaxDepth)
		ifExpr.Alternative = &ast.BlockStatement{
			Statements: []ast.Statement{&ast.ExpressionStatement{Expression: alt}},
		}
	}

	return &ast.ExpressionStatement{Expression: ifExpr}
}
