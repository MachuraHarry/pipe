package eval

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/harry/pipe/pkg/ast"
	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/object"
	"github.com/harry/pipe/pkg/parser"
)

type EvalContext struct {
	SourceFile      string
	callStack       []string
	importCache     map[string]struct{}
	exportedSymbols map[string]bool
}

func NewEvalContext(sourceFile string) *EvalContext {
	return &EvalContext{
		SourceFile:      sourceFile,
		importCache:     make(map[string]struct{}),
		exportedSymbols: make(map[string]bool),
	}
}

func (ctx *EvalContext) pushCall(name string)  { ctx.callStack = append(ctx.callStack, name) }
func (ctx *EvalContext) popCall()              { if len(ctx.callStack) > 0 { ctx.callStack = ctx.callStack[:len(ctx.callStack)-1] } }
func (ctx *EvalContext) stackTrace() string    { return "  in " + strings.Join(ctx.callStack, "\n  in ") }

func (ctx *EvalContext) Eval(node ast.Node, env *object.Environment) object.Object {
	switch n := node.(type) {
	case *ast.Program:
		return ctx.evalProgram(n.Statements, env)

	case *ast.ExpressionStatement:
		return ctx.Eval(n.Expression, env)

	case *ast.VarStatement:
		val := ctx.Eval(n.Value, env)
		if isError(val) {
			return val
		}
		env.Set(n.Name.Value, val)
		return val

	case *ast.BlockStatement:
		return ctx.evalBlockStatement(n, env)

	case *ast.IntegerLiteral:
		return &object.Integer{Value: n.Value}

	case *ast.FloatLiteral:
		return &object.Float{Value: n.Value}

	case *ast.StringLiteral:
		return &object.String{Value: n.Value}

	case *ast.BooleanLiteral:
		return object.NativeBoolToBoolean(n.Value)

	case *ast.NilLiteral:
		return object.NILOBJ

	case *ast.Identifier:
		return ctx.evalIdentifier(n, env)

	case *ast.PrefixExpression:
		right := ctx.Eval(n.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(n.Operator, right)

	case *ast.InfixExpression:
		left := ctx.Eval(n.Left, env)
		if isError(left) {
			return left
		}
		right := ctx.Eval(n.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(n.Operator, left, right)

	case *ast.IfExpression:
		return ctx.evalIfExpression(n, env)

	case *ast.WhileExpression:
		return ctx.evalWhileExpression(n, env)

	case *ast.ForExpression:
		return ctx.evalForExpression(n, env)

	case *ast.BreakStatement:
		return &BreakValue{}

	case *ast.DeferStatement:
		return &DeferredExpr{Expr: n.Expression, Env: env}

	case *ast.ExportStatement:
		ctx.exportedSymbols[n.FnName] = true
		return ctx.Eval(n.Fn, env)

	case *ast.EnumStatement:
		return ctx.evalEnumStatement(n, env)

	case *ast.ReturnStatement:
		return &ReturnValue{Value: ctx.Eval(n.Value, env)}

	case *ast.ContinueStatement:
		return &ContinueValue{}

	case *ast.ImportStatement:
		return ctx.evalImportStatement(n, env)

	case *ast.MatchExpression:
		return ctx.evalMatchExpression(n, env)

	case *ast.FnStatement:
		return ctx.evalFnStatement(n, env)

	case *ast.FnLiteral:
		return ctx.evalFnLiteral(n, env)

	case *ast.CallExpression:
		fn := ctx.Eval(n.Function, env)
		if isError(fn) {
			return fn
		}
		args := ctx.evalExpressions(n.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return ctx.applyFunction(fn, args)

	case *ast.PipelineExpression:
		left := ctx.Eval(n.Left, env)
		if isError(left) {
			return left
		}
		return ctx.evalPipeline(n, left, env)

	case *ast.ListLiteral:
		elements := ctx.evalExpressions(n.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.List{Elements: elements}

	case *ast.MapLiteral:
		return ctx.evalMapLiteral(n, env)

	case *ast.DotExpression:
		return ctx.evalDotExpression(n, env)

	case *ast.TryExpression:
		return ctx.evalTryExpression(n, env)

	case *ast.SliceExpression:
		return ctx.evalSliceExpression(n, env)

	default:
		return ctx.newError("unbekannter AST-Knoten: %T", node)
	}
}

func (ctx *EvalContext) evalProgram(stmts []ast.Statement, env *object.Environment) object.Object {
	var result object.Object
	var defers []*DeferredExpr

	for _, stmt := range stmts {
		result = ctx.Eval(stmt, env)
		if result != nil {
			if d, ok := result.(*DeferredExpr); ok {
				defers = append(defers, d)
				continue
			}
			if isError(result) {
				runDefers(defers)
				return result
			}
		}
	}

	runDefers(defers)
	return result
}

func (ctx *EvalContext) evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object
	var defers []*DeferredExpr

	for _, stmt := range block.Statements {
		result = ctx.Eval(stmt, env)
		if result != nil {
			if d, ok := result.(*DeferredExpr); ok {
				defers = append(defers, d)
				continue
			}
			switch result.Type() {
			case object.ERROR:
				runDefers(defers)
				return result
			case "RETURN":
				runDefers(defers)
				return result
			case "BREAK":
				runDefers(defers)
				return result
			case "CONTINUE":
				runDefers(defers)
				return result
			}
		}
	}

	runDefers(defers)
	return result
}

func runDefers(defers []*DeferredExpr) {
	for i := len(defers) - 1; i >= 0; i-- {
		d := defers[i]
		// Create a fresh throwaway context for defer execution
		ctx := NewEvalContext("<defer>")
		ctx.Eval(d.Expr, d.Env)
	}
}

type DeferredExpr struct {
	Expr ast.Expression
	Env  *object.Environment
}

func (d *DeferredExpr) Type() object.ObjectType { return "DEFER" }
func (d *DeferredExpr) Inspect() string         { return "deferred" }

func (ctx *EvalContext) evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}
	return ctx.newError("undefinierte Variable: %s", node.Value)
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "-":
		return evalMinusPrefixOperator(right)
	case "!":
		return object.NativeBoolToBoolean(!object.IsTruthy(right))
	default:
		return newErrorSt("unbekannter Operator: %s%s", operator, right.Type())
	}
}

func evalMinusPrefixOperator(right object.Object) object.Object {
	switch v := right.(type) {
	case *object.Integer:
		return &object.Integer{Value: -v.Value}
	case *object.Float:
		return &object.Float{Value: -v.Value}
	default:
		return newErrorSt("unbekannter Operator: -%s", right.Type())
	}
}

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	if operator == "[]" {
		return evalIndexExpression(left, right)
	}
	if operator == "&&" {
		if !object.IsTruthy(left) {
			return left
		}
		return right
	}
	if operator == "||" {
		if object.IsTruthy(left) {
			return left
		}
		return right
	}
	switch {
	case left.Type() == object.INTEGER && right.Type() == object.INTEGER:
		return evalIntegerInfix(operator, left.(*object.Integer), right.(*object.Integer))
	case left.Type() == object.FLOAT && right.Type() == object.FLOAT:
		return evalFloatInfix(operator, left.(*object.Float), right.(*object.Float))
	case left.Type() == object.INTEGER && right.Type() == object.FLOAT:
		return evalFloatInfix(operator, &object.Float{Value: float64(left.(*object.Integer).Value)}, right.(*object.Float))
	case left.Type() == object.FLOAT && right.Type() == object.INTEGER:
		return evalFloatInfix(operator, left.(*object.Float), &object.Float{Value: float64(right.(*object.Integer).Value)})
	case left.Type() == object.STRING && right.Type() == object.STRING:
		return evalStringInfix(operator, left.(*object.String), right.(*object.String))
	case operator == "==":
		return object.NativeBoolToBoolean(left == right)
	case operator == "!=":
		return object.NativeBoolToBoolean(left != right)
	default:
		return newErrorSt("Typ-Fehler: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfix(operator string, left, right *object.Integer) object.Object {
	l, r := left.Value, right.Value
	switch operator {
	case "+":
		return &object.Integer{Value: l + r}
	case "-":
		return &object.Integer{Value: l - r}
	case "*":
		return &object.Integer{Value: l * r}
	case "/":
		if r == 0 {
			return newErrorSt("Division durch Null")
		}
		return &object.Integer{Value: l / r}
	case "%":
		if r == 0 {
			return newErrorSt("Modulo durch Null")
		}
		return &object.Integer{Value: l % r}
	case "**":
		return &object.Integer{Value: int64(math.Pow(float64(l), float64(r)))}
	case "<":
		return object.NativeBoolToBoolean(l < r)
	case ">":
		return object.NativeBoolToBoolean(l > r)
	case "<=":
		return object.NativeBoolToBoolean(l <= r)
	case ">=":
		return object.NativeBoolToBoolean(l >= r)
	case "==":
		return object.NativeBoolToBoolean(l == r)
	case "!=":
		return object.NativeBoolToBoolean(l != r)
	default:
		return newErrorSt("unbekannter Operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalFloatInfix(operator string, left, right *object.Float) object.Object {
	l, r := left.Value, right.Value
	switch operator {
	case "+":
		return &object.Float{Value: l + r}
	case "-":
		return &object.Float{Value: l - r}
	case "*":
		return &object.Float{Value: l * r}
	case "/":
		if r == 0 {
			return newErrorSt("Division durch Null")
		}
		return &object.Float{Value: l / r}
	case "**":
		return &object.Float{Value: math.Pow(l, r)}
	case "%":
		return newErrorSt("Modulo nicht für Float definiert")
	case "<":
		return object.NativeBoolToBoolean(l < r)
	case ">":
		return object.NativeBoolToBoolean(l > r)
	case "<=":
		return object.NativeBoolToBoolean(l <= r)
	case ">=":
		return object.NativeBoolToBoolean(l >= r)
	case "==":
		return object.NativeBoolToBoolean(l == r)
	case "!=":
		return object.NativeBoolToBoolean(l != r)
	default:
		return newErrorSt("unbekannter Operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalStringInfix(operator string, left, right *object.String) object.Object {
	switch operator {
	case "++":
		return &object.String{Value: left.Value + right.Value}
	case "==":
		return object.NativeBoolToBoolean(left.Value == right.Value)
	case "!=":
		return object.NativeBoolToBoolean(left.Value != right.Value)
	default:
		return newErrorSt("unbekannter Operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func (ctx *EvalContext) evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := ctx.Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if object.IsTruthy(condition) {
		return ctx.Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return ctx.Eval(ie.Alternative, env)
	}
	return object.NILOBJ
}

func (ctx *EvalContext) evalMatchExpression(me *ast.MatchExpression, env *object.Environment) object.Object {
	value := ctx.Eval(me.Value, env)
	if isError(value) {
		return value
	}

	for _, c := range me.Cases {
		pattern := ctx.Eval(c.Pattern, env)
		if isError(pattern) {
			if ident, ok := c.Pattern.(*ast.Identifier); ok && ident.Value == "_" {
				return ctx.Eval(c.Body, env)
			}
			continue
		}

		if valuesEqual(value, pattern) {
			return ctx.Eval(c.Body, env)
		}

		if ident, ok := c.Pattern.(*ast.Identifier); ok && ident.Value == "_" {
			return ctx.Eval(c.Body, env)
		}
	}

	return object.NILOBJ
}

func (ctx *EvalContext) evalFnStatement(fn *ast.FnStatement, env *object.Environment) object.Object {
	fnObj := &object.Function{
		Name:       fn.Name.Value,
		Parameters: fn.Parameters,
		Body:       fn.Body,
		Env:        env,
		EvalCtx:    ctx,
	}
	env.Set(fn.Name.Value, fnObj)
	return fnObj
}

func (ctx *EvalContext) applyFunction(fn object.Object, args []object.Object) object.Object {
	switch f := fn.(type) {
	case *object.Function:
		fnCtx := ctx
		if f.EvalCtx != nil {
			if ec, ok := f.EvalCtx.(*EvalContext); ok {
				fnCtx = ec
			}
		}
		fnCtx.pushCall("fn(" + fnName(f) + ")")
		extendedEnv := extendFunctionEnv(f, args)
		evaluated := fnCtx.Eval(f.Body, extendedEnv)
		trace := fnCtx.stackTrace()
		fnCtx.popCall()
		result := unwrapReturnValue(evaluated)
		if result != nil && result.Type() == object.ERROR {
			if !strings.Contains(result.Inspect(), "  in ") {
				return ctx.newError("%s\n%s", result.Inspect(), trace)
			}
		}
		return result

	case *Builtin:
		return f.Fn(args...)

	default:
		return ctx.newError("keine Funktion: %s", fn.Type())
	}
}

func fnName(f *object.Function) string {
	if f.Name != "" {
		return f.Name
	}
	return "lambda"
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		env.Set(param.Value, args[i])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func (ctx *EvalContext) evalPipeline(pe *ast.PipelineExpression, left object.Object, env *object.Environment) object.Object {
	if callExpr, ok := pe.Right.(*ast.CallExpression); ok {
		fn := ctx.Eval(callExpr.Function, env)
		if isError(fn) {
			return fn
		}
		args := ctx.evalExpressions(callExpr.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		if callExpr.PipedArg {
			return ctx.applyFunction(fn, args)
		}
		allArgs := append([]object.Object{left}, args...)
		return ctx.applyFunction(fn, allArgs)
	}

	rightFn := ctx.Eval(pe.Right, env)
	if isError(rightFn) {
		return rightFn
	}

	switch fn := rightFn.(type) {
	case *object.Function:
		args := []object.Object{left}
		extendedEnv := extendFunctionEnv(fn, args)
		result := ctx.Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(result)
	case *Builtin:
		return fn.Fn(append([]object.Object{left})...)
	}

	return ctx.newError("Pipeline: rechte Seite ist keine Funktion")
}

func (ctx *EvalContext) evalMapLiteral(ml *ast.MapLiteral, env *object.Environment) object.Object {
	pairs := make(map[string]object.Object)

	for key, valExpr := range ml.Pairs {
		val := ctx.Eval(valExpr, env)
		if isError(val) {
			return val
		}
		pairs[key] = val
	}

	return &object.Map{Pairs: pairs}
}

func (ctx *EvalContext) evalDotExpression(de *ast.DotExpression, env *object.Environment) object.Object {
	left := ctx.Eval(de.Left, env)
	if isError(left) {
		return left
	}

	switch obj := left.(type) {
	case *object.Map:
		if val, ok := obj.Pairs[de.Field]; ok {
			return val
		}
		return ctx.newError("Feld '%s' nicht gefunden", de.Field)
	default:
		return ctx.newError("Punkt-Zugriff nur auf Maps möglich, nicht %s", left.Type())
	}
}

func (ctx *EvalContext) evalExpressions(exprs []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exprs {
		evaluated := ctx.Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func valuesEqual(a, b object.Object) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch a := a.(type) {
	case *object.Integer:
		return a.Value == b.(*object.Integer).Value
	case *object.Float:
		return a.Value == b.(*object.Float).Value
	case *object.String:
		return a.Value == b.(*object.String).Value
	case *object.Boolean:
		return a.Value == b.(*object.Boolean).Value
	default:
		return false
	}
}

func (ctx *EvalContext) evalTryExpression(te *ast.TryExpression, env *object.Environment) object.Object {
	result := ctx.Eval(te.TryBlock, env)
	if result != nil && result.Type() == object.ERROR {
		if te.CatchParam != nil {
			env.Set(te.CatchParam.Value, result)
		}
		return ctx.Eval(te.CatchBlock, env)
	}
	return result
}

func (ctx *EvalContext) evalEnumStatement(es *ast.EnumStatement, env *object.Environment) object.Object {
	for i, val := range es.Values {
		env.Set(val, &object.Integer{Value: int64(i)})
	}
	return object.NILOBJ
}

func (ctx *EvalContext) evalFnLiteral(fl *ast.FnLiteral, env *object.Environment) object.Object {
	return &object.Function{
		Name:       "lambda",
		Parameters: fl.Parameters,
		Body:       fl.Body,
		Env:        env,
		EvalCtx:    ctx,
	}
}

func (ctx *EvalContext) evalWhileExpression(we *ast.WhileExpression, env *object.Environment) object.Object {
	for {
		condition := ctx.Eval(we.Condition, env)
		if isError(condition) {
			return condition
		}
		if !object.IsTruthy(condition) {
			return object.NILOBJ
		}

		result := ctx.Eval(we.Body, env)
		if result == nil {
			continue
		}
		switch result.Type() {
		case "BREAK":
			return object.NILOBJ
		case "CONTINUE":
			continue
		case "RETURN":
			return result
		case object.ERROR:
			return result
		}
	}
}

func (ctx *EvalContext) evalForExpression(fe *ast.ForExpression, env *object.Environment) object.Object {
	if fe.IsForIn {
		return ctx.evalForInExpression(fe, env)
	}
	return ctx.newError("for-Schleifen noch nicht vollständig implementiert")
}

func (ctx *EvalContext) evalForInExpression(fe *ast.ForExpression, env *object.Environment) object.Object {
	iterable := ctx.Eval(fe.Iterable, env)
	if isError(iterable) {
		return iterable
	}

	list, ok := iterable.(*object.List)
	if !ok {
		return ctx.newError("for-in erwartet eine List, nicht %s", iterable.Type())
	}

	iterName := fe.Iterator.Value
	for _, elem := range list.Elements {
		env.Set(iterName, elem)

		result := ctx.Eval(fe.Body, env)
		if result == nil {
			continue
		}
		switch result.Type() {
		case "BREAK":
			return object.NILOBJ
		case "CONTINUE":
			continue
		case "RETURN":
			return result
		case object.ERROR:
			return result
		}
	}
	return object.NILOBJ
}

func (ctx *EvalContext) evalImportStatement(is *ast.ImportStatement, env *object.Environment) object.Object {
	if _, ok := ctx.importCache[is.Path]; ok {
		return object.NILOBJ
	}
	ctx.importCache[is.Path] = struct{}{}

	data, err := os.ReadFile(is.Path)
	if err != nil {
		return ctx.newError("import fehlgeschlagen: %s", err)
	}

	prevExports := ctx.exportedSymbols
	ctx.exportedSymbols = make(map[string]bool)

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		ctx.exportedSymbols = prevExports
		return ctx.newError("import parse-fehler in %s: %v", is.Path, p.Errors())
	}

	result := ctx.Eval(program, env)

	hasExports := len(ctx.exportedSymbols) > 0
	for name := range env.Store() {
		if hasExports && !ctx.exportedSymbols[name] {
			env.Delete(name)
		}
	}

	ctx.exportedSymbols = prevExports
	return result
}

func (ctx *EvalContext) evalSliceExpression(se *ast.SliceExpression, env *object.Environment) object.Object {
	list := ctx.Eval(se.List, env)
	if isError(list) {
		return list
	}

	l, ok := list.(*object.List)
	if !ok {
		return ctx.newError("Slice nur auf Listen möglich, nicht %s", list.Type())
	}

	startIdx := int64(0)
	endIdx := int64(len(l.Elements))

	if se.Start != nil {
		start := ctx.Eval(se.Start, env)
		if isError(start) {
			return start
		}
		s, ok := object.ToInt(start)
		if !ok {
			return ctx.newError("Slice-Start muss Zahl sein")
		}
		startIdx = s
	}

	if se.End != nil {
		end := ctx.Eval(se.End, env)
		if isError(end) {
			return end
		}
		e, ok := object.ToInt(end)
		if !ok {
			return ctx.newError("Slice-Ende muss Zahl sein")
		}
		endIdx = e
	}

	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > int64(len(l.Elements)) {
		endIdx = int64(len(l.Elements))
	}
	if startIdx > endIdx {
		startIdx = endIdx
	}

	return &object.List{Elements: l.Elements[startIdx:endIdx]}
}

func evalIndexExpression(left, right object.Object) object.Object {
	switch container := left.(type) {
	case *object.List:
		idx, ok := object.ToInt(right)
		if !ok {
			return newErrorSt("Listen-Index muss Zahl sein")
		}
		if idx < 0 || idx >= int64(len(container.Elements)) {
			return newErrorSt("Index %d außerhalb (Länge %d)", idx, len(container.Elements))
		}
		return container.Elements[idx]
	case *object.Map:
		key, ok := right.(*object.String)
		if !ok {
			return newErrorSt("Map-Key muss String sein")
		}
		val, exists := container.Pairs[key.Value]
		if !exists {
			return object.NILOBJ
		}
		return val
	case *object.String:
		idx, ok := object.ToInt(right)
		if !ok {
			return newErrorSt("String-Index muss Zahl sein")
		}
		s := container.Value
		if idx < 0 || idx >= int64(len(s)) {
			return newErrorSt("Index %d außerhalb", idx)
		}
		return &object.String{Value: string(s[idx])}
	}
	return newErrorSt("Index-Zugriff nicht unterstützt für %s", left.Type())
}

type ReturnValue struct {
	Value object.Object
}

func (rv *ReturnValue) Type() object.ObjectType { return "RETURN" }
func (rv *ReturnValue) Inspect() string         { return rv.Value.Inspect() }

type BreakValue struct{}

func (bv *BreakValue) Type() object.ObjectType { return "BREAK" }
func (bv *BreakValue) Inspect() string         { return "break" }

type ContinueValue struct{}

func (cv *ContinueValue) Type() object.ObjectType { return "CONTINUE" }
func (cv *ContinueValue) Inspect() string         { return "continue" }

func (ctx *EvalContext) newError(format string, a ...interface{}) *object.Error {
	msg := fmt.Sprintf(format, a...)
	return &object.Error{Message: ctx.SourceFile + ": " + msg}
}

func newErrorSt(format string, a ...interface{}) *object.Error {
	msg := fmt.Sprintf(format, a...)
	return &object.Error{Message: msg}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR
	}
	return false
}
