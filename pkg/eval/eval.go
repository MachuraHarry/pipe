package eval

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

type ModuleInstance struct {
	Env             *object.Environment
	ExportedSymbols map[string]bool
}

type EvalContext struct {
	SourceFile      string
	callStack       []string
	importCache     map[string]*ast.Program
	importedModules map[string]*ModuleInstance
	importStack     []string
	exportedSymbols map[string]bool
	testFailed      bool
	lastPos         ast.Position // position of the most recently evaluated node
}

func NewEvalContext(sourceFile string) *EvalContext {
	ctx := &EvalContext{
		SourceFile:      sourceFile,
		importCache:     make(map[string]*ast.Program),
		importedModules: make(map[string]*ModuleInstance),
		exportedSymbols: make(map[string]bool),
	}
	return ctx
}

// CallUserFunction satisfies object.UserFunctionExecutor: builtins such as
// map/filter/reduce call back into this tree-walker context.
func (ctx *EvalContext) CallUserFunction(fn object.Object, args ...object.Object) object.Object {
	return ctx.applyFunction(fn, args)
}

func (ctx *EvalContext) pushCall(name string) { ctx.callStack = append(ctx.callStack, name) }
func (ctx *EvalContext) popCall() {
	if len(ctx.callStack) > 0 {
		ctx.callStack = ctx.callStack[:len(ctx.callStack)-1]
	}
}

const maxTraceFrames = 40

func (ctx *EvalContext) stackTrace() string {
	frames := ctx.callStack
	if len(frames) > maxTraceFrames {
		frames = frames[:maxTraceFrames]
		return "  in " + strings.Join(frames, "\n  in ") + fmt.Sprintf("\n  ... (%d more)", len(ctx.callStack)-maxTraceFrames)
	}
	return "  in " + strings.Join(frames, "\n  in ")
}

func (ctx *EvalContext) Eval(node ast.Node, env *object.Environment) object.Object {
	if pn, ok := node.(interface{ Pos() ast.Position }); ok {
		if pos := pn.Pos(); pos.Line > 0 {
			ctx.lastPos = pos
		}
	}
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
		right = object.EnsureResolved(right)
		return evalPrefixExpression(n.Operator, right)

	case *ast.InfixExpression:
		left := ctx.Eval(n.Left, env)
		if isError(left) {
			return left
		}
		left = object.EnsureResolved(left)

		if n.Operator == "&&" {
			if !object.IsTruthy(left) {
				return left
			}
			right := ctx.Eval(n.Right, env)
			if isError(right) {
				return right
			}
			right = object.EnsureResolved(right)
			return right
		}
		if n.Operator == "||" {
			if object.IsTruthy(left) {
				return left
			}
			right := ctx.Eval(n.Right, env)
			if isError(right) {
				return right
			}
			right = object.EnsureResolved(right)
			return right
		}

		right := ctx.Eval(n.Right, env)
		if isError(right) {
			return right
		}
		right = object.EnsureResolved(right)
		return ctx.evalInfixExpression(n.Operator, left, right)

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
		if n.Fn != nil {
			ctx.exportedSymbols[n.FnName] = true
			return ctx.Eval(n.Fn, env)
		}
		if n.Var != nil {
			ctx.exportedSymbols[n.VarName] = true
			return ctx.Eval(n.Var, env)
		}
		if n.Enum != nil {
			for _, val := range n.Enum.Values {
				ctx.exportedSymbols[val] = true
			}
			return ctx.Eval(n.Enum, env)
		}
		return object.NILOBJ

	case *ast.TestStatement:
		return ctx.evalTestStatement(n, env)

	case *ast.EnumStatement:
		return ctx.evalEnumStatement(n, env)

	case *ast.StructStatement:
		return ctx.evalStructStatement(n, env)

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
		var fn object.Object
		if ident, ok := n.Function.(*ast.Identifier); ok && len(n.Arguments) > 0 {
			if builtin, ok := builtins[ident.Value]; ok && builtin.Arity == 0 {
				fn = builtin
			}
		}
		if fn == nil {
			fn = ctx.Eval(n.Function, env)
		}
		if isError(fn) {
			return fn
		}
		// Struct construction via positional args: Point(10, 20)
		if def, ok := fn.(*object.StructDef); ok {
			return ctx.evalStructConstructor(def, n.Arguments, env)
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
		left = object.EnsureResolved(left)
		if n.Parallel {
			return ctx.evalParallelPipeline(n, left, env)
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
		return ctx.newError("unknown AST node: %T", node)
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

	if ctx.testFailed {
		return ctx.newError("some tests failed")
	}

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
		if builtin.Arity == 0 {
			return builtin.Fn()
		}
		return builtin
	}
	return ctx.newErrorCode("E001", "undefined variable: %s%s", node.Value, suggestName(node.Value, env))
}

// suggestName returns a " — did you mean 'x'?" hint for a typo'd name, using
// Levenshtein distance against all visible variables and builtins. Returns ""
// when no candidate is close enough.
func suggestName(name string, env *object.Environment) string {
	if name == "" || env == nil {
		return ""
	}
	best, bestDist := "", 4
	consider := func(cand string) {
		if cand == "" || cand == name {
			return
		}
		if d := levenshtein(name, cand); d < bestDist {
			best, bestDist = cand, d
		}
	}
	for k := range env.Store() {
		consider(k)
	}
	for k := range builtins {
		consider(k)
	}
	if best == "" {
		return ""
	}
	return fmt.Sprintf(" — did you mean '%s'?", best)
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = minInt(prev[j]+1, minInt(cur[j-1]+1, prev[j-1]+cost))
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "-":
		return evalMinusPrefixOperator(right)
	case "!":
		return object.NativeBoolToBoolean(!object.IsTruthy(right))
	default:
		return newErrorSt("unknown prefix operator '%s' for %s", operator, right.Type())
	}
}

func evalMinusPrefixOperator(right object.Object) object.Object {
	switch v := right.(type) {
	case *object.Integer:
		return &object.Integer{Value: -v.Value}
	case *object.Float:
		return &object.Float{Value: -v.Value}
	default:
		return newErrorSt("cannot negate a %s with '-'", right.Type())
	}
}

func (ctx *EvalContext) evalInfixExpression(operator string, left, right object.Object) object.Object {
	left = object.EnsureResolved(left)
	right = object.EnsureResolved(right)
	if operator == "[]" {
		return evalIndexExpression(ctx, left, right)
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
		return evalIntegerInfix(ctx, operator, left.(*object.Integer), right.(*object.Integer))
	case left.Type() == object.FLOAT && right.Type() == object.FLOAT:
		return evalFloatInfix(ctx, operator, left.(*object.Float), right.(*object.Float))
	case left.Type() == object.INTEGER && right.Type() == object.FLOAT:
		return evalFloatInfix(ctx, operator, &object.Float{Value: float64(left.(*object.Integer).Value)}, right.(*object.Float))
	case left.Type() == object.FLOAT && right.Type() == object.INTEGER:
		return evalFloatInfix(ctx, operator, left.(*object.Float), &object.Float{Value: float64(right.(*object.Integer).Value)})
	case left.Type() == object.STRING && right.Type() == object.STRING:
		return evalStringInfix(ctx, operator, left.(*object.String), right.(*object.String))
	case left.Type() == object.BYTES && right.Type() == object.BYTES:
		return evalBytesInfix(ctx, operator, left.(*object.Bytes), right.(*object.Bytes))
	case operator == "++" && left.Type() == object.BYTES && right.Type() == object.STRING:
		return concatBytesString(left.(*object.Bytes).Value, []byte(right.(*object.String).Value))
	case operator == "++" && left.Type() == object.STRING && right.Type() == object.BYTES:
		return concatBytesString([]byte(left.(*object.String).Value), right.(*object.Bytes).Value)
	case operator == "==":
		return object.NativeBoolToBoolean(left == right)
	case operator == "!=":
		return object.NativeBoolToBoolean(left != right)
	default:
		return ctx.newErrorCode("E002", "type mismatch: cannot apply '%s' between %s and %s", operator, left.Type(), right.Type())
	}
}

func evalIntegerInfix(ctx *EvalContext, operator string, left, right *object.Integer) object.Object {
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
			return ctx.newErrorCode("E003", "division by zero")
		}
		return &object.Integer{Value: l / r}
	case "%":
		if r == 0 {
			return ctx.newErrorCode("E003", "modulo by zero")
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
		return ctx.newErrorCode("E005", "operator '%s' not supported for %s and %s", operator, left.Type(), right.Type())
	}
}

func evalFloatInfix(ctx *EvalContext, operator string, left, right *object.Float) object.Object {
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
			return ctx.newErrorCode("E003", "division by zero")
		}
		return &object.Float{Value: l / r}
	case "**":
		return &object.Float{Value: math.Pow(l, r)}
	case "%":
		return newErrorSt("'%%' is not defined for float — use integer values or to_num()")
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
		return ctx.newErrorCode("E005", "operator '%s' not supported for float", operator)
	}
}

func evalStringInfix(ctx *EvalContext, operator string, left, right *object.String) object.Object {
	switch operator {
	case "++":
		return &object.String{Value: left.Value + right.Value}
	case "==":
		return object.NativeBoolToBoolean(left.Value == right.Value)
	case "!=":
		return object.NativeBoolToBoolean(left.Value != right.Value)
	case "<":
		return object.NativeBoolToBoolean(left.Value < right.Value)
	case ">":
		return object.NativeBoolToBoolean(left.Value > right.Value)
	case "<=":
		return object.NativeBoolToBoolean(left.Value <= right.Value)
	case ">=":
		return object.NativeBoolToBoolean(left.Value >= right.Value)
	default:
		return ctx.newErrorCode("E005", "operator '%s' not supported for strings", operator)
	}
}

func concatBytesString(l, r []byte) object.Object {
	out := make([]byte, 0, len(l)+len(r))
	out = append(out, l...)
	out = append(out, r...)
	return &object.Bytes{Value: out}
}

func evalBytesInfix(ctx *EvalContext, operator string, left, right *object.Bytes) object.Object {
	switch operator {
	case "++":
		out := make([]byte, 0, len(left.Value)+len(right.Value))
		out = append(out, left.Value...)
		out = append(out, right.Value...)
		return &object.Bytes{Value: out}
	case "==":
		return object.NativeBoolToBoolean(bytes.Equal(left.Value, right.Value))
	case "!=":
		return object.NativeBoolToBoolean(!bytes.Equal(left.Value, right.Value))
	case "<", ">", "<=", ">=":
		c := bytes.Compare(left.Value, right.Value)
		switch operator {
		case "<":
			return object.NativeBoolToBoolean(c < 0)
		case ">":
			return object.NativeBoolToBoolean(c > 0)
		case "<=":
			return object.NativeBoolToBoolean(c <= 0)
		case ">=":
			return object.NativeBoolToBoolean(c >= 0)
		}
	}
	return ctx.newErrorCode("E005", "operator '%s' not supported for bytes", operator)
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
		matched := false
		if isError(pattern) {
			if isWildcardPattern(c.Pattern) {
				matched = true
			}
		} else if valuesEqual(value, pattern) || isWildcardPattern(c.Pattern) {
			matched = true
		}
		if !matched {
			continue
		}
		if !ctx.evalMatchGuard(c, env) {
			continue
		}
		return ctx.Eval(c.Body, env)
	}

	return object.NILOBJ
}

// isWildcardPattern reports whether an expression is the `_` wildcard, which
// matches any value without evaluating it as a variable.
func isWildcardPattern(expr ast.Expression) bool {
	if ident, ok := expr.(*ast.Identifier); ok {
		return ident.Value == "_"
	}
	return false
}

// evalMatchGuard reports whether a matched case's guard passes. A guard that
// raises an error or is falsy makes the case not match, so matching falls
// through to the next case — mirroring the bytecode VM.
func (ctx *EvalContext) evalMatchGuard(c ast.MatchCase, env *object.Environment) bool {
	if c.Guard == nil {
		return true
	}
	guard := ctx.Eval(c.Guard, env)
	if isError(guard) {
		return false
	}
	return object.IsTruthy(guard)
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

// isAwaitBuiltin reports whether fn is the await builtin, which must receive
// its Future argument unresolved so a timeout can be applied.
func isAwaitBuiltin(fn object.Object) bool {
	if b, ok := fn.(*Builtin); ok {
		return b.Name == object.AwaitBuiltinName
	}
	return object.IsAwaitBuiltin(fn)
}

func (ctx *EvalContext) applyFunction(fn object.Object, args []object.Object) object.Object {
	if !isAwaitBuiltin(fn) {
		for i, arg := range args {
			args[i] = object.EnsureResolved(arg)
		}
	}
	switch f := fn.(type) {
	case *object.Function:
		fnCtx := ctx
		if f.EvalCtx != nil {
			if ec, ok := f.EvalCtx.(*EvalContext); ok {
				fnCtx = ec
			}
		}
		if len(fnCtx.callStack) >= object.MaxCallDepth {
			return ctx.newErrorCode("E008", "call stack depth exceeded (%d)", object.MaxCallDepth)
		}
		fnCtx.pushCall("fn(" + fnName(f) + ")")
		name := fnName(f)

		// Tail call optimization: if body ends with recursive call to self, loop
		isTailRecursive := false
		var tailCallExpr *ast.CallExpression
		if len(f.Body.Statements) > 0 {
			if last, ok := f.Body.Statements[len(f.Body.Statements)-1].(*ast.ExpressionStatement); ok {
				if ce, ok := last.Expression.(*ast.CallExpression); ok {
					if ident, ok := ce.Function.(*ast.Identifier); ok {
						if ident.Value == name {
							isTailRecursive = true
							tailCallExpr = ce
						}
					}
				}
			}
		}

		extendedEnv := extendFunctionEnv(f, args)
		var evaluated object.Object

		if isTailRecursive && tailCallExpr != nil {
			for {
				// Evaluate all statements except the last (which is the recursive call)
				for i := 0; i < len(f.Body.Statements)-1; i++ {
					result := fnCtx.Eval(f.Body.Statements[i], extendedEnv)
					if result != nil && (result.Type() == object.ERROR || result.Type() == "RETURN") {
						evaluated = result
						break
					}
				}
				if evaluated != nil {
					break
				}

				// Evaluate tail call args in current env
				callArgs := fnCtx.evalExpressions(tailCallExpr.Arguments, extendedEnv)
				if len(callArgs) == 1 && isError(callArgs[0]) {
					evaluated = callArgs[0]
					break
				}

				// Update parameters with new args (reuse the same env)
				for i, param := range f.Parameters {
					extendedEnv.Set(param.Value, callArgs[i])
				}
				// Continue loop
			}
		} else {
			evaluated = fnCtx.Eval(f.Body, extendedEnv)
		}

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
		ctx.pushCall("fn(" + f.Name + ")")
		result := f.Fn(args...)
		if result != nil && result.Type() == object.ERROR {
			st := ctx.stackTrace()
			if st != "" && !strings.Contains(result.Inspect(), "  in ") {
				result = ctx.newError("%s\n%s", result.Inspect(), st)
			}
		}
		ctx.popCall()
		return result

	case *object.BuiltinInfo:
		result := f.Fn(args...)
		if result != nil && result.Type() == object.ERROR {
			return ctx.newError("%s", result.Inspect())
		}
		return result

	default:
		return ctx.newErrorCode("E004", "not a function: %s", fn.Type())
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
		if i < len(args) {
			env.Set(param.Value, args[i])
		} else {
			env.Set(param.Value, object.NILOBJ)
		}
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

type TailCall struct {
	Target *object.Function
	Args   []object.Object
}

func (tc *TailCall) Type() object.ObjectType { return "TAILCALL" }
func (tc *TailCall) Inspect() string         { return "tailcall" }

func (tc *TailCall) SameFunction(f *object.Function) bool {
	if tc.Target == nil || f == nil {
		return false
	}
	return tc.Target == f
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
		return fn.Fn([]object.Object{left}...)
	}

	return ctx.newError("pipeline: right side of '>' is %s, not a function — pipeline requires a function call on the right", rightFn.Type())
}

func (ctx *EvalContext) evalParallelPipeline(pe *ast.PipelineExpression, left object.Object, env *object.Environment) object.Object {
	future := object.NewFuture()
	// Clone the captured environment so the background goroutine reads a
	// snapshot instead of racing the caller, which keeps writing to it.
	branchEnv := env.Clone()

	go func() {
		// A fresh EvalContext keeps the background call stack and lastPos
		// isolated from the caller's (the tree-walker is not goroutine-safe).
		result := NewEvalContext(ctx.SourceFile).evalPipeline(pe, left, branchEnv)
		future.Val = result
		close(future.Done)
	}()

	return future
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
	if err, ok := left.(*object.Error); ok {
		if de.Field == "message" {
			return &object.String{Value: err.Message}
		}
		return left
	}
	if isError(left) {
		return left
	}

	switch obj := left.(type) {
	case *object.StructInstance:
		if val, ok := obj.Values[de.Field]; ok {
			return val
		}
		return ctx.newError("struct %s has no field '%s'", obj.Def.Name, de.Field)
	case *object.Map:
		if val, ok := obj.Pairs[de.Field]; ok {
			return val
		}
		return ctx.newError("field '%s' not found", de.Field)
	default:
		return ctx.newError("cannot use .%s on a %s — dot access requires a struct or map", de.Field, left.Type())
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
	if result == nil || result.Type() != object.ERROR {
		return result
	}

	err, isErr := result.(*object.Error)
	if !isErr {
		return result
	}

	if te.AIFix {
		fixed := ctx.tryAIFix(err, te.TryBlock, env)
		if fixed != nil && fixed.Type() != object.ERROR {
			return fixed
		}
		ai.LogTryAIFix(extractErrorCode(err.Message), blockSource(te.TryBlock), err.Message, 0, false)
		if te.CatchBlock == nil {
			return result
		}
	}

	if te.CatchBlock == nil {
		return result
	}

	if te.CatchParam != nil {
		env.Set(te.CatchParam.Value, &object.String{Value: err.Message})
	}
	return ctx.Eval(te.CatchBlock, env)
}

func (ctx *EvalContext) tryAIFix(err *object.Error, block *ast.BlockStatement, env *object.Environment) object.Object {
	code := extractErrorCode(err.Message)
	if !isAIFixable(code) {
		return nil
	}

	src := blockSource(block)

	for attempt := 1; attempt <= 3; attempt++ {
		profile := object.ActiveProfile.Load()
		if profile.Name != "none" {
			if canErr := profile.CanAI(); canErr != nil {
				ai.LogTryAIFix(code, src, canErr.Error(), attempt, false)
				return nil
			}
		} else if object.Sandbox.Enabled && !object.Sandbox.AllowAI {
			ai.LogTryAIFix(code, src, "AI fix blocked by sandbox", attempt, false)
			return nil
		}

		var prompt string
		if attempt == 1 {
			prompt = fmt.Sprintf("Error: %s\nExpression: %s\nFix it.", err.Message, src)
		} else {
			prompt = fmt.Sprintf("Error: %s\nExpression: %s\nPrevious fix failed. Try a different approach.\nFix it.", err.Message, src)
		}

		resp, aiErr := ai.Chat(ai.ChatRequest{
			Messages: []ai.Message{
				{Role: "system", Content: aiFixSystemPrompt},
				{Role: "user", Content: prompt},
			},
		})
		if aiErr != nil {
			return nil
		}

		fix := strings.TrimSpace(resp.Content)
		if fix == "" || strings.HasPrefix(fix, "UNFIXABLE") || strings.HasPrefix(fix, "unfixable") {
			ai.LogTryAIFix(code, src, fix, attempt, false)
			return nil
		}

		result := ctx.validateAndApply(fix, env)
		if result != nil && result.Type() != object.ERROR {
			ai.LogTryAIFix(code, src, fix, attempt, true)
			return result
		}

		ai.LogTryAIFix(code, src, fix, attempt, false)

		if errObj, ok := result.(*object.Error); ok {
			err = errObj
		}
	}

	return nil
}

var aiFixSystemPrompt = `You are a Pipe expression fixer. Pipe is a pipeline language with space-separated args.

RULES:
1. Return ONLY the corrected expression, no explanation, no markdown.
2. ALWAYS wrap the result in (parentheses): (to_num "42") * 3
3. If the fix is multi-statement, write each statement on its own line.
4. Do NOT add comments or explanations.

PIPE BUILTINS:
  Conversion: to_num, to_str
  Math: abs, min, max, pow, sqrt, round
  String: len, upper, lower, trim, split, join, contains, at
  List: len, push, pop, at, sort, range, map, filter, reduce, each, slice_list
  Map: get, set, keys, values
  Type: type_of, is_num, is_str, is_list, is_map, is_nil
  JSON: parse_json, to_json
  Regex: regex_match, regex_replace
  Time: now, format_time
  Random: random, random_range
  Hash: sha256, md5, sha1, sha512
  Encoding: base64_encode, base64_decode

PIPE KEYWORDS:
  fn, match, if, else, elif, while, for, break, continue, return, defer
  import, export, enum, true, false, nil, try, catch, try_ai, test, assert, assert_eq

PIPE OPERATORS:
  Arithmetic: + - * / % **
  Comparison: == != < > <= >=
  Logic: && || ! not
  String: ++ (concat)
  Pipeline: > (pipe) >> (parallel)

PIPE SYNTAX:
  Function call: fn_name arg1 arg2       (space-separated, no commas)
  Wrap in parens if nested: (fn arg1 arg2)
  Lists: [elem1, elem2, elem3]           (comma-separated)
  Maps: {key1: val1, key2: val2}         (key: value, comma-separated)
  Index access: list[index]  or  map["key"]
  Assignment: name: value
  Strings: "double-quoted only"

ERROR FIXING STRATEGIES:
  Type errors (E002): If string looks numeric, use to_num. If number should be string, use to_str.
  Division by zero (E003): Replace divisor with max(divisor, 1) or guard with if.
  Not a function (E004): Wrap in parentheses or use a builtin instead.
  Operator not supported (E005): Convert one operand to the other's type.
  Cannot index (E006): Check list length with len first, or use get with map.
  Undefined variable (E001): Use a literal default value like 0, "", nil, or [].

If truly impossible to fix, return exactly: UNFIXABLE`

func (ctx *EvalContext) validateAndApply(fix string, env *object.Environment) object.Object {
	fix = strings.TrimSpace(fix)
	if len(fix) > 2 && fix[0] == '(' && fix[len(fix)-1] == ')' {
		fix = strings.TrimSpace(fix[1 : len(fix)-1])
	}

	l := lexer.New(fix)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 || len(program.Statements) == 0 {
		return nil
	}

	es, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return nil
	}

	prevProfile := object.ActiveProfile.Load()
	object.ActiveProfile.Store(newTryAIRing2Profile(prevProfile))
	defer func() { object.ActiveProfile.Store(prevProfile) }()

	sandbox := env.Copy()
	result := ctx.Eval(es.Expression, sandbox)
	if result == nil || result.Type() == object.ERROR {
		return nil
	}

	return ctx.Eval(es.Expression, env)
}

// newTryAIRing2Profile builds the sandboxed profile used to re-run an
// AI-generated fix expression. It ratchets down FS/network/exec access and
// inherits the caller's budget and call limits, so a nested AI call in a fix
// cannot spend beyond the caller's caps.
func newTryAIRing2Profile(prev *object.SandboxProfile) *object.SandboxProfile {
	return &object.SandboxProfile{
		Name:         "try_ai_ring2",
		FSAccess:     object.FSNone,
		Network:      false,
		Exec:         false,
		AI:           true,
		Budget:       prev.Budget,
		MaxToolCalls: prev.MaxToolCalls,
		Timeout:      prev.Timeout,
		AuditLog:     prev.AuditLog,
	}
}

func extractErrorCode(msg string) string {
	if idx := strings.Index(msg, "E0"); idx >= 0 {
		end := idx + 4
		if end > len(msg) {
			end = len(msg)
		}
		return msg[idx:end]
	}
	return ""
}

func isAIFixable(code string) bool {
	switch code {
	case "E001", "E002", "E003", "E004", "E005", "E006":
		return true
	}
	return false
}

func blockSource(block *ast.BlockStatement) string {
	if len(block.Statements) == 0 {
		return "(empty)"
	}
	if len(block.Statements) == 1 {
		if es, ok := block.Statements[0].(*ast.ExpressionStatement); ok {
			s := es.Expression.String()
			if len(s) > 2 && s[0] == '(' && s[len(s)-1] == ')' {
				s = s[1 : len(s)-1]
			}
			return s
		}
		// Handle non-expression statements (if, for, etc.)
		return fmt.Sprintf("%s", block.Statements[0])
	}
	var parts []string
	for _, stmt := range block.Statements {
		parts = append(parts, fmt.Sprintf("%s", stmt))
	}
	return strings.Join(parts, "\n")
}

func (ctx *EvalContext) evalEnumStatement(es *ast.EnumStatement, env *object.Environment) object.Object {
	for i, val := range es.Values {
		env.Set(val, &object.Integer{Value: int64(i)})
	}
	return object.NILOBJ
}

func (ctx *EvalContext) evalStructStatement(ss *ast.StructStatement, env *object.Environment) object.Object {
	def := &object.StructDef{
		Name:     ss.Name,
		Fields:   make([]string, 0, len(ss.Fields)),
		Defaults: make(map[string]object.Object),
	}
	for _, f := range ss.Fields {
		def.Fields = append(def.Fields, f.Name)
		if f.Default != nil {
			def.Defaults[f.Name] = ctx.Eval(f.Default, env)
		}
	}
	env.Set(ss.Name, def)
	return def
}

func (ctx *EvalContext) evalStructConstructor(def *object.StructDef, argExprs []ast.Expression, env *object.Environment) object.Object {
	instance := &object.StructInstance{
		Def:    def,
		Values: make(map[string]object.Object),
	}
	for k, v := range def.Defaults {
		instance.Values[k] = v
	}
	for i, argExpr := range argExprs {
		if i >= len(def.Fields) {
			return ctx.newError("struct %s accepts at most %d arguments, got %d", def.Name, len(def.Fields), len(argExprs))
		}
		val := ctx.Eval(argExpr, env)
		if isError(val) {
			return val
		}
		instance.Values[def.Fields[i]] = val
	}
	return instance
}

func (ctx *EvalContext) evalTestStatement(ts *ast.TestStatement, env *object.Environment) object.Object {
	// Setup/teardown hooks are silent: they run like a normal block and
	// propagate errors (a failing setup aborts the file, a failing teardown
	// fails it after the tests). evalBlockStatement short-circuits at the
	// first error, which the VM mirrors with OpTestAbortIfError probes.
	if ts.Hook != "" {
		return ctx.Eval(ts.Body, env)
	}
	name := ""
	if ts.Name != nil {
		name = ts.Name.Value
	}
	result := ctx.Eval(ts.Body, env)
	if isError(result) {
		fmt.Printf("  FAIL %s (%s)\n", name, result.Inspect())
		ctx.testFailed = true
		return object.NILOBJ
	}
	fmt.Printf("  PASS %s\n", name)
	return result
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

	// C-style for
	if fe.Init != nil {
		result := ctx.Eval(fe.Init, env)
		if isError(result) {
			return result
		}
	}

	for {
		if fe.Condition != nil {
			cond := ctx.Eval(fe.Condition, env)
			if isError(cond) {
				return cond
			}
			if !object.IsTruthy(cond) {
				break
			}
		}

		result := ctx.Eval(fe.Body, env)
		if result != nil {
			switch result.Type() {
			case "BREAK":
				return object.NILOBJ
			case "CONTINUE":
				// fall through to update
			case "RETURN":
				return result
			case object.ERROR:
				return result
			}
		}

		if fe.Update != nil {
			updResult := ctx.Eval(fe.Update, env)
			if isError(updResult) {
				return updResult
			}
		}
	}

	return object.NILOBJ
}

func (ctx *EvalContext) evalForInExpression(fe *ast.ForExpression, env *object.Environment) object.Object {
	iterable := ctx.Eval(fe.Iterable, env)
	if isError(iterable) {
		return iterable
	}

	list, ok := iterable.(*object.List)
	if !ok {
		return ctx.newError("for-in expects a list, not %s", iterable.Type())
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
	resolvedPath, content, err := object.ResolveImportFrom(is.Path, ctx.SourceFile)
	if err != nil {
		return ctx.newError("%s", err)
	}

	// Flat imports: if already loaded, re-inject exports from shared instance
	if is.Alias == "" {
		if mi, ok := ctx.importedModules[resolvedPath]; ok {
			for name, val := range mi.Env.Store() {
				if len(mi.ExportedSymbols) == 0 || mi.ExportedSymbols[name] {
					env.Set(name, val)
				}
			}
			return object.NILOBJ
		}
	}

	// Use cached parse result or parse fresh
	program, ok := ctx.importCache[resolvedPath]
	if !ok {
		var err error
		program, err = object.ParseContent(content)
		if err != nil {
			return ctx.newError("import parse error in %s: %v", resolvedPath, err)
		}
		ctx.importCache[resolvedPath] = program
	}

	// Circular import detection: if the module is already being evaluated
	// further up the import chain, this is a cycle (a.pipe -> b.pipe -> a.pipe).
	for _, inProgress := range ctx.importStack {
		if inProgress == resolvedPath {
			chain := append(append([]string{}, ctx.importStack...), resolvedPath)
			return ctx.newErrorCode("E009", "circular import: %s", strings.Join(chain, " -> "))
		}
	}
	ctx.importStack = append(ctx.importStack, resolvedPath)
	defer func() { ctx.importStack = ctx.importStack[:len(ctx.importStack)-1] }()

	prevExports := ctx.exportedSymbols
	ctx.exportedSymbols = make(map[string]bool)

	importEnv := object.NewEnvironment()
	result := ctx.Eval(program, importEnv)

	// Save as shared module instance for future imports
	if is.Alias == "" {
		exports := make(map[string]bool)
		for k, v := range ctx.exportedSymbols {
			exports[k] = v
		}
		ctx.importedModules[resolvedPath] = &ModuleInstance{
			Env:             importEnv,
			ExportedSymbols: exports,
		}
	}

	hasExports := len(ctx.exportedSymbols) > 0

	if is.Alias != "" {
		nsObj := &object.Map{Pairs: make(map[string]object.Object)}
		for name, val := range importEnv.Store() {
			if !hasExports || ctx.exportedSymbols[name] {
				nsObj.Pairs[name] = val
			}
		}
		env.Set(is.Alias, nsObj)
	} else {
		for name, val := range importEnv.Store() {
			if !hasExports || ctx.exportedSymbols[name] {
				env.Set(name, val)
			}
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

	var length int
	switch v := list.(type) {
	case *object.List:
		length = len(v.Elements)
	case *object.String:
		length = len(v.Value)
	case *object.Bytes:
		length = len(v.Value)
	default:
		return ctx.newError("slice only on lists, strings or bytes, not %s", list.Type())
	}

	startIdx := int64(0)
	endIdx := int64(length)

	if se.Start != nil {
		start := ctx.Eval(se.Start, env)
		if isError(start) {
			return start
		}
		s, ok := object.ToInt(start)
		if !ok {
			return ctx.newError("slice start must be a number")
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
			return ctx.newError("slice end must be a number")
		}
		endIdx = e
	}

	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > int64(length) {
		endIdx = int64(length)
	}
	if startIdx > endIdx {
		startIdx = endIdx
	}

	switch v := list.(type) {
	case *object.List:
		return &object.List{Elements: v.Elements[startIdx:endIdx]}
	case *object.String:
		return &object.String{Value: v.Value[startIdx:endIdx]}
	case *object.Bytes:
		out := make([]byte, endIdx-startIdx)
		copy(out, v.Value[startIdx:endIdx])
		return &object.Bytes{Value: out}
	}
	return object.NILOBJ
}

func evalIndexExpression(ctx *EvalContext, left, right object.Object) object.Object {
	switch container := left.(type) {
	case *object.List:
		idx, ok := object.ToInt(right)
		if !ok {
			return newErrorSt("list index must be a number")
		}
		if idx < 0 || idx >= int64(len(container.Elements)) {
			return object.NILOBJ
		}
		return container.Elements[idx]
	case *object.Map:
		key, ok := right.(*object.String)
		if !ok {
			return newErrorSt("map key must be a string")
		}
		val, exists := container.Pairs[key.Value]
		if !exists {
			return object.NILOBJ
		}
		return val
	case *object.String:
		idx, ok := object.ToInt(right)
		if !ok {
			return newErrorSt("string index must be a number")
		}
		s := container.Value
		if idx < 0 || idx >= int64(len(s)) {
			return object.NILOBJ
		}
		return &object.String{Value: string(s[idx])}
	}
	return ctx.newErrorCode("E006", "cannot index into a %s — only lists, maps, and strings support [ ]", left.Type())
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

func (ctx *EvalContext) newErrorCode(code, format string, a ...interface{}) *object.Error {
	msg := fmt.Sprintf(format, a...)
	prefix := ""
	if ctx.SourceFile != "" {
		prefix = ctx.SourceFile + ": "
	}
	return &object.Error{
		Message: prefix + code + ": " + msg,
		Code:    code,
		File:    ctx.SourceFile,
		Line:    ctx.lastPos.Line,
		Col:     ctx.lastPos.Col,
	}
}

func newErrorSt(format string, a ...interface{}) *object.Error {
	msg := fmt.Sprintf(format, a...)
	return &object.Error{Message: msg}
}

func newErrorCode(sourceFile, code, format string, a ...interface{}) *object.Error {
	msg := fmt.Sprintf(format, a...)
	if sourceFile != "" {
		return &object.Error{Message: sourceFile + ": " + code + ": " + msg}
	}
	return &object.Error{Message: code + ": " + msg}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR
	}
	return false
}

func tryAIEvalFromSource(source string) object.Object {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 || len(program.Statements) == 0 {
		return &object.Error{Message: "_try_ai_eval: parse error"}
	}

	es, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return &object.Error{Message: "_try_ai_eval: expected expression"}
	}

	tryExpr := &ast.TryExpression{
		AIFix:    true,
		TryBlock: &ast.BlockStatement{Statements: []ast.Statement{es}},
	}

	ctx := NewEvalContext("<try_ai_vm>")
	ctx.SourceFile = "<try_ai_vm>"
	env := object.NewEnvironment()

	return ctx.Eval(tryExpr, env)
}
