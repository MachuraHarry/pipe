package eval

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/harry/pulse/pkg/ast"
	"github.com/harry/pulse/pkg/lexer"
	"github.com/harry/pulse/pkg/object"
	"github.com/harry/pulse/pkg/parser"
)

var callStack []string
var importCache = make(map[string]struct{})

func pushCall(name string)  { callStack = append(callStack, name) }
func popCall()              { if len(callStack) > 0 { callStack = callStack[:len(callStack)-1] } }
func stackTrace() string    { return "  in " + strings.Join(callStack, "\n  in ") }

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch n := node.(type) {
	case *ast.Program:
		return evalProgram(n.Statements, env)

	case *ast.ExpressionStatement:
		return Eval(n.Expression, env)

	case *ast.VarStatement:
		val := Eval(n.Value, env)
		if isError(val) {
			return val
		}
		env.Set(n.Name.Value, val)
		return val

	case *ast.BlockStatement:
		return evalBlockStatement(n, env)

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
		return evalIdentifier(n, env)

	case *ast.PrefixExpression:
		right := Eval(n.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(n.Operator, right)

	case *ast.InfixExpression:
		left := Eval(n.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(n.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(n.Operator, left, right)

	case *ast.IfExpression:
		return evalIfExpression(n, env)

	case *ast.WhileExpression:
		return evalWhileExpression(n, env)

	case *ast.ForExpression:
		return evalForExpression(n, env)

	case *ast.BreakStatement:
		return &BreakValue{}

	case *ast.ReturnStatement:
		return &ReturnValue{Value: Eval(n.Value, env)}

	case *ast.ContinueStatement:
		return &ContinueValue{}

	case *ast.ImportStatement:
		return evalImportStatement(n, env)

	case *ast.MatchExpression:
		return evalMatchExpression(n, env)

	case *ast.FnStatement:
		return evalFnStatement(n, env)

	case *ast.FnLiteral:
		return evalFnLiteral(n, env)

	case *ast.CallExpression:
		fn := Eval(n.Function, env)
		if isError(fn) {
			return fn
		}
		args := evalExpressions(n.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(fn, args)

	case *ast.PipelineExpression:
		left := Eval(n.Left, env)
		if isError(left) {
			return left
		}
		// For pipeline: thread left as first argument to right function
		return evalPipeline(n, left, env)

	case *ast.ListLiteral:
		elements := evalExpressions(n.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.List{Elements: elements}

	case *ast.MapLiteral:
		return evalMapLiteral(n, env)

	case *ast.DotExpression:
		return evalDotExpression(n, env)

	case *ast.TryExpression:
		return evalTryExpression(n, env)

	case *ast.SliceExpression:
		return evalSliceExpression(n, env)

	default:
		return newError("unbekannter AST-Knoten: %T", node)
	}
}

func evalProgram(stmts []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, stmt := range stmts {
		result = Eval(stmt, env)
		if isError(result) {
			return result
		}
	}

	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, stmt := range block.Statements {
		result = Eval(stmt, env)
		if result != nil {
			switch result.Type() {
			case object.ERROR:
				return result
			case "RETURN":
				return result
			case "BREAK":
				return result
			case "CONTINUE":
				return result
			}
		}
	}

	return result
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}
	return newError("undefinierte Variable: %s", node.Value)
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "-":
		return evalMinusPrefixOperator(right)
	case "!":
		return object.NativeBoolToBoolean(!object.IsTruthy(right))
	default:
		return newError("unbekannter Operator: %s%s", operator, right.Type())
	}
}

func evalMinusPrefixOperator(right object.Object) object.Object {
	switch v := right.(type) {
	case *object.Integer:
		return &object.Integer{Value: -v.Value}
	case *object.Float:
		return &object.Float{Value: -v.Value}
	default:
		return newError("unbekannter Operator: -%s", right.Type())
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
		return newError("Typ-Fehler: %s %s %s", left.Type(), operator, right.Type())
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
			return newError("Division durch Null")
		}
		return &object.Integer{Value: l / r}
	case "%":
		if r == 0 {
			return newError("Modulo durch Null")
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
		return newError("unbekannter Operator: %s %s %s", left.Type(), operator, right.Type())
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
			return newError("Division durch Null")
		}
		return &object.Float{Value: l / r}
		case "**":
			return &object.Float{Value: math.Pow(l, r)}
		case "%":
		return newError("Modulo nicht für Float definiert")
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
		return newError("unbekannter Operator: %s %s %s", left.Type(), operator, right.Type())
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
		return newError("unbekannter Operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if object.IsTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	}
	return object.NILOBJ
}

func evalMatchExpression(me *ast.MatchExpression, env *object.Environment) object.Object {
	value := Eval(me.Value, env)
	if isError(value) {
		return value
	}

	for _, c := range me.Cases {
		pattern := Eval(c.Pattern, env)
		if isError(pattern) {
			// Wildcard _ matches anything
			if ident, ok := c.Pattern.(*ast.Identifier); ok && ident.Value == "_" {
				return Eval(c.Body, env)
			}
			continue
		}

		if valuesEqual(value, pattern) {
			return Eval(c.Body, env)
		}

		// Wildcard _ always matches
		if ident, ok := c.Pattern.(*ast.Identifier); ok && ident.Value == "_" {
			return Eval(c.Body, env)
		}
	}

	return object.NILOBJ
}

func evalFnStatement(fn *ast.FnStatement, env *object.Environment) object.Object {
	fnObj := &object.Function{
		Name:       fn.Name.Value,
		Parameters: fn.Parameters,
		Body:       fn.Body,
		Env:        env,
	}
	env.Set(fn.Name.Value, fnObj)
	return fnObj
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch f := fn.(type) {
	case *object.Function:
		pushCall("fn(" + fnName(f) + ")")
		extendedEnv := extendFunctionEnv(f, args)
		evaluated := Eval(f.Body, extendedEnv)
		trace := stackTrace()
		popCall()
		result := unwrapReturnValue(evaluated)
		if result != nil && result.Type() == object.ERROR {
			if !strings.Contains(result.Inspect(), "  in ") {
				return newError("%s\n%s", result.Inspect(), trace)
			}
		}
		return result

	case *Builtin:
		return f.Fn(args...)

	default:
		return newError("keine Funktion: %s", fn.Type())
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

func evalPipeline(pe *ast.PipelineExpression, left object.Object, env *object.Environment) object.Object {
	// If right side is a call expression, thread left as first argument
	if callExpr, ok := pe.Right.(*ast.CallExpression); ok {
		fn := Eval(callExpr.Function, env)
		if isError(fn) {
			return fn
		}
		args := evalExpressions(callExpr.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		allArgs := append([]object.Object{left}, args...)
		return applyFunction(fn, allArgs)
	}

	// If right side is an identifier (function name), call it with left as arg
	rightFn := Eval(pe.Right, env)
	if isError(rightFn) {
		return rightFn
	}

	switch fn := rightFn.(type) {
	case *object.Function:
		args := []object.Object{left}
		extendedEnv := extendFunctionEnv(fn, args)
		result := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(result)
	case *Builtin:
		return fn.Fn(append([]object.Object{left})...)
	}

	return newError("Pipeline: rechte Seite ist keine Funktion")
}

func evalMapLiteral(ml *ast.MapLiteral, env *object.Environment) object.Object {
	pairs := make(map[string]object.Object)

	for key, valExpr := range ml.Pairs {
		val := Eval(valExpr, env)
		if isError(val) {
			return val
		}
		pairs[key] = val
	}

	return &object.Map{Pairs: pairs}
}

func evalDotExpression(de *ast.DotExpression, env *object.Environment) object.Object {
	left := Eval(de.Left, env)
	if isError(left) {
		return left
	}

	switch obj := left.(type) {
	case *object.Map:
		if val, ok := obj.Pairs[de.Field]; ok {
			return val
		}
		return newError("Feld '%s' nicht gefunden", de.Field)
	default:
		return newError("Punkt-Zugriff nur auf Maps möglich, nicht %s", left.Type())
	}
}

func evalExpressions(exprs []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exprs {
		evaluated := Eval(e, env)
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

func evalTryExpression(te *ast.TryExpression, env *object.Environment) object.Object {
	result := Eval(te.TryBlock, env)
	if result != nil && result.Type() == object.ERROR {
		if te.CatchParam != nil {
			env.Set(te.CatchParam.Value, result)
		}
		return Eval(te.CatchBlock, env)
	}
	return result
}

func evalFnLiteral(fl *ast.FnLiteral, env *object.Environment) object.Object {
	return &object.Function{
		Name:       "lambda",
		Parameters: fl.Parameters,
		Body:       fl.Body,
		Env:        env,
	}
}

func evalWhileExpression(we *ast.WhileExpression, env *object.Environment) object.Object {
	for {
		condition := Eval(we.Condition, env)
		if isError(condition) {
			return condition
		}
		if !object.IsTruthy(condition) {
			return object.NILOBJ
		}

		result := Eval(we.Body, env)
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

func evalForExpression(fe *ast.ForExpression, env *object.Environment) object.Object {
	if fe.IsForIn {
		return evalForInExpression(fe, env)
	}
	return newError("for-Schleifen noch nicht vollständig implementiert")
}

func evalForInExpression(fe *ast.ForExpression, env *object.Environment) object.Object {
	iterable := Eval(fe.Iterable, env)
	if isError(iterable) {
		return iterable
	}

	list, ok := iterable.(*object.List)
	if !ok {
		return newError("for-in erwartet eine List, nicht %s", iterable.Type())
	}

	iterName := fe.Iterator.Value
	for _, elem := range list.Elements {
		env.Set(iterName, elem)

		result := Eval(fe.Body, env)
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

func evalImportStatement(is *ast.ImportStatement, env *object.Environment) object.Object {
	// Import cache: skip if already imported
	if _, ok := importCache[is.Path]; ok {
		return object.NILOBJ
	}
	importCache[is.Path] = struct{}{}

	data, err := os.ReadFile(is.Path)
	if err != nil {
		return newError("import fehlgeschlagen: %s", err)
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return newError("import parse-fehler in %s: %v", is.Path, p.Errors())
	}

	return Eval(program, env)
}

func evalSliceExpression(se *ast.SliceExpression, env *object.Environment) object.Object {
	list := Eval(se.List, env)
	if isError(list) {
		return list
	}

	l, ok := list.(*object.List)
	if !ok {
		return newError("Slice nur auf Listen möglich, nicht %s", list.Type())
	}

	startIdx := int64(0)
	endIdx := int64(len(l.Elements))

	if se.Start != nil {
		start := Eval(se.Start, env)
		if isError(start) {
			return start
		}
		s, ok := object.ToInt(start)
		if !ok {
			return newError("Slice-Start muss Zahl sein")
		}
		startIdx = s
	}

	if se.End != nil {
		end := Eval(se.End, env)
		if isError(end) {
			return end
		}
		e, ok := object.ToInt(end)
		if !ok {
			return newError("Slice-Ende muss Zahl sein")
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
			return newError("Listen-Index muss Zahl sein")
		}
		if idx < 0 || idx >= int64(len(container.Elements)) {
			return newError("Index %d außerhalb (Länge %d)", idx, len(container.Elements))
		}
		return container.Elements[idx]
	case *object.Map:
		key, ok := right.(*object.String)
		if !ok {
			return newError("Map-Key muss String sein")
		}
		val, exists := container.Pairs[key.Value]
		if !exists {
			return object.NILOBJ
		}
		return val
	case *object.String:
		idx, ok := object.ToInt(right)
		if !ok {
			return newError("String-Index muss Zahl sein")
		}
		s := container.Value
		if idx < 0 || idx >= int64(len(s)) {
			return newError("Index %d außerhalb", idx)
		}
		return &object.String{Value: string(s[idx])}
	}
	return newError("Index-Zugriff nicht unterstützt für %s", left.Type())
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

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR
	}
	return false
}
