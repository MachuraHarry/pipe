package eval

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/harry/pipe/pkg/ai"
	"github.com/harry/pipe/pkg/ast"
	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/object"
	"github.com/harry/pipe/pkg/parser"
)

type EvalContext struct {
	SourceFile      string
	callStack       []string
	importCache     map[string]*ast.Program
	importedFiles   map[string]struct{}
	exportedSymbols map[string]bool
	testFailed      bool
}

func NewEvalContext(sourceFile string) *EvalContext {
	ctx := &EvalContext{
		SourceFile:      sourceFile,
		importCache:     make(map[string]*ast.Program),
		importedFiles:   make(map[string]struct{}),
		exportedSymbols: make(map[string]bool),
	}
	object.SetCallUserFn(func(fn object.Object, args ...object.Object) object.Object {
		return ctx.applyFunction(fn, args)
	})
	return ctx
}

func (ctx *EvalContext) pushCall(name string) { ctx.callStack = append(ctx.callStack, name) }
func (ctx *EvalContext) popCall() {
	if len(ctx.callStack) > 0 {
		ctx.callStack = ctx.callStack[:len(ctx.callStack)-1]
	}
}
func (ctx *EvalContext) stackTrace() string { return "  in " + strings.Join(ctx.callStack, "\n  in ") }

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
		right = object.EnsureResolved(right)
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
		left = object.EnsureResolved(left)
		right = object.EnsureResolved(right)
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
		return builtin
	}
	return ctx.newErrorCode("E001", "undefined variable: %s", node.Value)
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

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	left = object.EnsureResolved(left)
	right = object.EnsureResolved(right)
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
		return newErrorCode("", "E002", "type mismatch: cannot apply '%s' between %s and %s", operator, left.Type(), right.Type())
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
			return newErrorCode("", "E003", "division by zero")
		}
		return &object.Integer{Value: l / r}
	case "%":
		if r == 0 {
			return newErrorCode("", "E003", "modulo by zero")
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
		return newErrorCode("", "E005", "operator '%s' not supported for %s and %s", operator, left.Type(), right.Type())
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
			return newErrorCode("", "E003", "division by zero")
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
		return newErrorCode("", "E005", "operator '%s' not supported for float", operator)
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
	case "<":
		return object.NativeBoolToBoolean(left.Value < right.Value)
	case ">":
		return object.NativeBoolToBoolean(left.Value > right.Value)
	case "<=":
		return object.NativeBoolToBoolean(left.Value <= right.Value)
	case ">=":
		return object.NativeBoolToBoolean(left.Value >= right.Value)
	default:
		return newErrorCode("", "E005", "operator '%s' not supported for strings", operator)
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
	for i, arg := range args {
		args[i] = object.EnsureResolved(arg)
	}
	switch f := fn.(type) {
	case *object.Function:
		fnCtx := ctx
		if f.EvalCtx != nil {
			if ec, ok := f.EvalCtx.(*EvalContext); ok {
				fnCtx = ec
			}
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
		return f.Fn(args...)

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
	branchEnv := env.Clone()

	go func() {
		result := ctx.evalPipeline(pe, left, branchEnv)
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
	if isError(left) {
		return left
	}

	switch obj := left.(type) {
	case *object.Map:
		if val, ok := obj.Pairs[de.Field]; ok {
			return val
		}
		return ctx.newError("field '%s' not found", de.Field)
	default:
		return ctx.newError("cannot use .%s on a %s — dot access requires a map", de.Field, left.Type())
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
		if fixed := ctx.tryAIFix(err, te.TryBlock, env); fixed != nil && fixed.Type() != object.ERROR {
			return fixed
		}
	}

	if te.CatchBlock == nil {
		return result
	}

	if te.CatchParam != nil {
		env.Set(te.CatchParam.Value, result)
	}
	return ctx.Eval(te.CatchBlock, env)
}

func (ctx *EvalContext) tryAIFix(err *object.Error, block *ast.BlockStatement, env *object.Environment) object.Object {
	code := extractErrorCode(err.Message)
	if !isAIFixable(code) {
		return nil
	}

	src := blockSource(block)
	prompt := fmt.Sprintf("Error: %s\nExpression: %s\nFix it.", err.Message, src)

	resp, aiErr := ai.Chat(ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: `You are a Pipe expression fixer. Pipe is a pipeline language with space-separated args.

RULES:
1. Return ONLY the corrected expression, no explanation
2. ALWAYS wrap in (parentheses): (to_num "42") * 3
3. Pipe builtins: to_num, to_str, abs, min, max, len, upper, lower, trim, split, join, push, pop, get, set, keys, values, sort, range, type_of, is_num, is_str, is_list, is_map, is_nil, parse_json, to_json, now, format_time
4. Pipe keywords: fn, match, if, else, while, for, break, continue, import, export, enum, defer, return, try, catch, true, false, nil
5. Pipe operators: + - * / % ** ++ == != < > <= >= && || ! >> > >>
6. Pipe syntax: args after function are space-separated: fn arg1 arg2
7. Pipe lists: [a, b, c]. Maps: {key: val}
8. Pipe dot-access: map.field. Index: list[0]
9. For type errors: if string looks numeric, use to_num. If number should be string, use to_str
10. For division by zero: replace divisor with max(divisor, 1) or (if divisor != 0 then divisor else 1)
11. For index errors: use get with default: get map "key" "default"
12. If truly unfixable (missing data), return: UNFIXABLE`},
			{Role: "user", Content: prompt},
		},
	})
	if aiErr != nil {
		return nil
	}

	fix := strings.TrimSpace(resp.Content)
	if fix == "" || strings.HasPrefix(fix, "UNFIXABLE") || strings.HasPrefix(fix, "unfixable") {
		return nil
	}

	return ctx.validateAndApply(fix, env)
}

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

	sandbox := env.Copy()
	result := ctx.Eval(es.Expression, sandbox)
	if result == nil || result.Type() == object.ERROR {
		return nil
	}

	return ctx.Eval(es.Expression, env)
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
	case "E002", "E003", "E006":
		return true
	}
	return false
}

func blockSource(block *ast.BlockStatement) string {
	if len(block.Statements) == 1 {
		if es, ok := block.Statements[0].(*ast.ExpressionStatement); ok {
			s := es.Expression.String()
			if len(s) > 2 && s[0] == '(' && s[len(s)-1] == ')' {
				s = s[1 : len(s)-1]
			}
			return s
		}
	}
	var parts []string
	for _, s := range block.Statements {
		parts = append(parts, fmt.Sprintf("%s", s))
	}
	return strings.Join(parts, "; ")
}

func (ctx *EvalContext) evalEnumStatement(es *ast.EnumStatement, env *object.Environment) object.Object {
	for i, val := range es.Values {
		env.Set(val, &object.Integer{Value: int64(i)})
	}
	return object.NILOBJ
}

func (ctx *EvalContext) evalTestStatement(ts *ast.TestStatement, env *object.Environment) object.Object {
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

func (ctx *EvalContext) resolveImportPath(path string) (string, error) {
	// Absolute or relative path that exists
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	// Try PIPE_PATH directories
	pipePath := os.Getenv("PIPE_PATH")
	if pipePath != "" {
		for _, dir := range strings.Split(pipePath, ":") {
			candidate := filepath.Join(dir, path)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("import not found: %s (PIPE_PATH=%s)", path, pipePath)
}

func (ctx *EvalContext) evalImportStatement(is *ast.ImportStatement, env *object.Environment) object.Object {
	resolvedPath, content, err := object.ResolveImportFrom(is.Path, ctx.SourceFile)
	if err != nil {
		return ctx.newError("%s", err)
	}

	// Flat imports: skip if already injected into this scope
	if is.Alias == "" {
		if _, ok := ctx.importedFiles[resolvedPath]; ok {
			return object.NILOBJ
		}
		ctx.importedFiles[resolvedPath] = struct{}{}
	}

	// Use cached parse result or parse fresh
	program, ok := ctx.importCache[resolvedPath]
	if !ok {
		l := lexer.New(content)
		p := parser.New(l)
		program = p.ParseProgram()
		if len(p.Errors()) > 0 {
			return ctx.newError("import parse error in %s: %v", resolvedPath, p.Errors())
		}
		ctx.importCache[resolvedPath] = program
	}

	prevExports := ctx.exportedSymbols
	ctx.exportedSymbols = make(map[string]bool)

	importEnv := object.NewEnvironment()
	result := ctx.Eval(program, importEnv)

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

	l, ok := list.(*object.List)
	if !ok {
		return ctx.newError("slice only on lists, not %s", list.Type())
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
	return newErrorCode("", "E006", "cannot index into a %s — only lists, maps, and strings support [ ]", left.Type())
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
	return newErrorCode(ctx.SourceFile, code, format, a...)
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
