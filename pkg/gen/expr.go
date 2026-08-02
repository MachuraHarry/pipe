package gen

import (
	"fmt"

	"github.com/harry/pipe/pkg/ast"
)

const exprRetries = 5

func (g *Generator) genExpr(prefType PipeType, depth int) ast.Expression {
	return g.genExprNoPipeline(prefType, depth)
}

func (g *Generator) genExprNoPipeline(prefType PipeType, depth int) ast.Expression {
	if depth <= 0 {
		return g.genLeaf(prefType)
	}
	for attempt := 0; attempt < exprRetries; attempt++ {
		e := g.tryGenSubExpr(prefType, depth)
		if e != nil {
			return e
		}
	}
	return g.genLeaf(prefType)
}

func (g *Generator) tryGenSubExpr(prefType PipeType, depth int) ast.Expression {
	r := g.rng.Intn(100)
	switch {
	case r < 30:
		return g.genLeaf(prefType)
	case r < 55:
		return g.genCall(prefType, depth)
	case r < 80:
		return g.genInfix(prefType, depth)
	case r < 88:
		return g.genPrefix(depth)
	case r < 93:
		return g.genList(depth)
	case r < 98:
		return g.genMap(depth)
	default:
		return g.genLeaf(prefType)
	}
}

func (g *Generator) genTopExpr(prefType PipeType, depth int) ast.Expression {
	if depth <= 0 {
		return g.genLeaf(prefType)
	}
	r := g.rng.Intn(100)
	switch {
	case r < 10 && g.opts.Pipelines:
		return g.genPipeline(depth)
	case r < 25:
		return g.genIfOrMatch(prefType, depth)
	case r < 40:
		return g.genCall(prefType, depth)
	case r < 60:
		return g.genInfix(prefType, depth)
	case r < 70:
		return g.genPrefix(depth)
	case r < 78:
		return g.genList(depth)
	case r < 85:
		return g.genMap(depth)
	default:
		return g.genLeaf(prefType)
	}
}

func (g *Generator) genLeaf(prefType PipeType) ast.Expression {
	if prefType == TypeAny && g.rng.Intn(3) > 0 {
		if name, _, _ := g.pickFromScope(KindVariable); name != "" {
			return &ast.Identifier{Value: name}
		}
		if name, _, _ := g.pickFromScope(KindParam); name != "" {
			return &ast.Identifier{Value: name}
		}
	}
	return g.genLiteral(prefType)
}

func (g *Generator) genLiteral(prefType PipeType) ast.Expression {
	switch prefType {
	case TypeInt:
		return &ast.IntegerLiteral{Value: int64(g.rng.Intn(1000))}
	case TypeFloat:
		return &ast.FloatLiteral{Value: float64(g.rng.Intn(1000)) + g.rng.Float64()}
	case TypeString:
		return &ast.StringLiteral{Value: g.randomString()}
	case TypeBool:
		return &ast.BooleanLiteral{Value: g.rng.Intn(2) == 0}
	default:
		types := []PipeType{TypeInt, TypeFloat, TypeString, TypeBool, TypeNil}
		t := types[g.rng.Intn(len(types))]
		switch t {
		case TypeInt:
			return &ast.IntegerLiteral{Value: int64(g.rng.Intn(1000))}
		case TypeFloat:
			return &ast.FloatLiteral{Value: float64(g.rng.Intn(1000)) + g.rng.Float64()}
		case TypeString:
			return &ast.StringLiteral{Value: g.randomString()}
		case TypeBool:
			return &ast.BooleanLiteral{Value: g.rng.Intn(2) == 0}
		default:
			return &ast.NilLiteral{}
		}
	}
}

func (g *Generator) genCall(prefType PipeType, depth int) ast.Expression {
	name, entry, ok := g.pickFunc()
	if !ok {
		return g.genLeaf(prefType)
	}
	arity := entry.Arity
	switch {
	case arity == 0:
		// 0-arity: just emit the identifier (pipe has no explicit 0-arg call syntax
		// that guarantees reparse with the parser. The identifier alone is a
		// reference to the function, not a call. We need at least one arg.)
		arity = 1
	case arity > 4:
		arity = randInt(g.rng, 1, 4)
	}

	args := make([]ast.Expression, arity)
	for i := 0; i < arity; i++ {
		args[i] = g.genExpr(TypeAny, depth-1)
	}
	return &ast.CallExpression{
		Function:  &ast.Identifier{Value: name},
		Arguments: args,
	}
}

func (g *Generator) genInfix(prefType PipeType, depth int) ast.Expression {
	ops := []string{"+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=", "++", "&&", "||"}
	op := ops[g.rng.Intn(len(ops))]
	var left, right ast.Expression
	switch op {
	case "+", "-", "*", "/", "%", "**":
		left = g.genNumeric(depth - 1)
		right = g.genNumeric(depth - 1)
	case "++":
		left = g.genLeaf(TypeString)
		right = g.genLeaf(TypeString)
	case "&&", "||":
		left = g.genExpr(TypeAny, depth-1)
		right = g.genExpr(TypeAny, depth-1)
	default:
		left = g.genExpr(TypeAny, depth-1)
		right = g.genExpr(TypeAny, depth-1)
	}
	return &ast.InfixExpression{
		Operator: op,
		Left:     left,
		Right:    right,
	}
}

func operandTypes(op string) (PipeType, PipeType) {
	switch op {
	case "+", "-", "*", "/", "%", "**":
		return TypeInt, TypeInt
	case "++":
		return TypeString, TypeString
	case "&&", "||":
		return TypeBool, TypeBool
	default:
		return TypeAny, TypeAny
	}
}

func (g *Generator) genPrefix(depth int) ast.Expression {
	op := "-"
	if g.rng.Intn(3) == 0 {
		op = "!"
	}
	rt := TypeInt
	if op == "!" {
		rt = TypeAny
	}
	var right ast.Expression
	if rt == TypeInt {
		right = g.genNumeric(depth - 1)
	} else {
		right = g.genExpr(rt, depth-1)
	}
	return &ast.PrefixExpression{
		Operator: op,
		Right:    right,
	}
}

func (g *Generator) genNumeric(depth int) ast.Expression {
	if depth <= 0 || g.rng.Intn(3) == 0 {
		return g.genLiteral(TypeInt)
	}
	if g.rng.Intn(3) == 0 {
		return g.genInfix(TypeInt, depth)
	}
	return g.genLeaf(TypeInt)
}

func (g *Generator) genList(depth int) ast.Expression {
	n := randInt(g.rng, 1, 6)
	elems := make([]ast.Expression, n)
	for i := 0; i < n; i++ {
		elems[i] = g.genExpr(TypeAny, depth-1)
	}
	return &ast.ListLiteral{Elements: elems}
}

func (g *Generator) genMap(depth int) ast.Expression {
	n := randInt(g.rng, 1, 4)
	pairs := make(map[string]ast.Expression, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key%d", i)
		pairs[key] = g.genExpr(TypeAny, depth-1)
	}
	return &ast.MapLiteral{Pairs: pairs}
}

func (g *Generator) genPipeline(depth int) ast.Expression {
	left := g.genExpr(TypeAny, depth-1)

	fnName, _, ok := g.pickFunc()
	if !ok {
		return left
	}
	arity := randInt(g.rng, 1, 3)
	args := make([]ast.Expression, arity)
	for i := 0; i < arity; i++ {
		args[i] = g.genExpr(TypeAny, depth-2)
	}
	return &ast.PipelineExpression{
		Left: left,
		Right: &ast.CallExpression{
			Function:  &ast.Identifier{Value: fnName},
			Arguments: args,
		},
	}
}

func (g *Generator) genIfOrMatch(prefType PipeType, depth int) ast.Expression {
	if g.rng.Intn(2) == 0 {
		return g.genMatch(depth)
	}
	return g.genIf(depth)
}

func (g *Generator) genIf(depth int) ast.Expression {
	cond := g.genExpr(TypeBool, depth-1)
	cons := g.genExpr(TypeAny, depth-1)

	ie := &ast.IfExpression{
		Condition:   cond,
		Consequence: &ast.BlockStatement{Statements: []ast.Statement{&ast.ExpressionStatement{Expression: cons}}},
	}
	if g.rng.Intn(3) == 0 {
		alt := g.genExpr(TypeAny, depth-1)
		ie.Alternative = &ast.BlockStatement{Statements: []ast.Statement{&ast.ExpressionStatement{Expression: alt}}}
	}
	return ie
}

func (g *Generator) genMatch(depth int) ast.Expression {
	val := g.genExpr(TypeAny, depth-1)
	nCases := randInt(g.rng, 1, 4)
	cases := make([]ast.MatchCase, nCases)
	for i := 0; i < nCases; i++ {
		var pat ast.Expression
		switch g.rng.Intn(4) {
		case 0:
			pat = &ast.IntegerLiteral{Value: int64(g.rng.Intn(10))}
		case 1:
			pat = &ast.StringLiteral{Value: g.randomString()}
		case 2:
			pat = &ast.BooleanLiteral{Value: g.rng.Intn(2) == 0}
		default:
			pat = &ast.Identifier{Value: "_"}
		}
		cases[i] = ast.MatchCase{
			Pattern: pat,
			Body:    g.genExpr(TypeAny, depth-1),
		}
	}
	return &ast.MatchExpression{
		Value: val,
		Cases: cases,
	}
}

func (g *Generator) randomString() string {
	adjs := []string{
		"hello", "world", "test", "data", "file", "text",
		"pipe", "code", "input", "output", "value", "key",
	}
	return adjs[g.rng.Intn(len(adjs))]
}
