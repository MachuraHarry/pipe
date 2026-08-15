package eval

import (
	"github.com/MachuraHarry/pipe/pkg/object"
)

type Builtin struct {
	Name  string
	Fn    func(args ...object.Object) object.Object
	Arity int
}

func (b *Builtin) Type() object.ObjectType { return "BUILTIN" }
func (b *Builtin) Inspect() string         { return "builtin function" }

var builtins = map[string]*Builtin{}

var zeroArityBuiltins = map[string]bool{
	"now":               true,
	"time_ms":           true,
	"random":            true,
	"secure_random_int": true,
	"try_ai_log":        true,
	"args":              true,
	"read_stdin":        true,
	"ai_cost":           true,
	"ai_tokens":         true,
	"ai_cache_hits":     true,
	"ai_cache_misses":   true,
	"audit_log":         true,
	"budget_spent":      true,
	"mcp_serve_stdio":   true,
	"mcp_tools":         true,
	"mcp_resources":     true,
	"mcp_prompts":       true,
	"mutex":             true,
}

func init() {
	for _, bi := range object.Builtins {
		bi := bi
		builtins[bi.Name] = &Builtin{Name: bi.Name, Fn: bi.Fn, Arity: -1}
	}
	for name := range zeroArityBuiltins {
		if b, ok := builtins[name]; ok {
			b.Arity = 0
		}
	}

	builtins["map"] = &Builtin{Name: "map", Fn: bMap, Arity: 2}
	builtins["filter"] = &Builtin{Name: "filter", Fn: bFilter, Arity: 2}
	builtins["reduce"] = &Builtin{Name: "reduce", Fn: bReduce, Arity: 3}
	builtins["each"] = &Builtin{Name: "each", Fn: bEach, Arity: 2}
	builtins["go"] = &Builtin{Name: "go", Fn: bGo, Arity: 1}
	builtins["spawn"] = &Builtin{Name: "spawn", Fn: bSpawn, Arity: 1}

	object.TryAIEvalFn = tryAIEvalFromSource
}

func bMap(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newErr("map expects 2 arguments (list, function)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("map: first argument must be a list")
	}
	fn := args[1]
	result := make([]object.Object, len(list.Elements))
	for i, elem := range list.Elements {
		result[i] = applyFn(fn, []object.Object{elem})
		if result[i] != nil && result[i].Type() == object.ERROR {
			return result[i]
		}
	}
	return &object.List{Elements: result}
}

func bFilter(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newErr("filter expects 2 arguments (list, function)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("filter: first argument must be a list")
	}
	fn := args[1]
	var result []object.Object
	for _, elem := range list.Elements {
		r := applyFn(fn, []object.Object{elem})
		if r != nil && r.Type() == object.ERROR {
			return r
		}
		if object.IsTruthy(r) {
			result = append(result, elem)
		}
	}
	return &object.List{Elements: result}
}

func bReduce(args ...object.Object) object.Object {
	if len(args) != 3 {
		return newErr("reduce expects 3 arguments (list, function, initial value)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("reduce: first argument must be a list")
	}
	fn := args[1]
	acc := args[2]
	for _, elem := range list.Elements {
		acc = applyFn(fn, []object.Object{acc, elem})
		if acc != nil && acc.Type() == object.ERROR {
			return acc
		}
	}
	return acc
}

func bEach(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newErr("each expects 2 arguments (list, function)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("each: first argument must be a list")
	}
	fn := args[1]
	for _, elem := range list.Elements {
		applyFn(fn, []object.Object{elem})
	}
	return object.NILOBJ
}

func bGo(args ...object.Object) object.Object {
	if len(args) < 1 {
		return newErr("go expects at least 1 argument (function)")
	}
	fn := args[0]
	fnArgs := args[1:]

	switch f := fn.(type) {
	case *object.Function:
		// Clone the captured environment so the background goroutine reads a
		// snapshot instead of racing the caller, which keeps writing to it.
		branchEnv := f.Env.Clone()
		go func() {
			extEnv := object.NewEnclosedEnvironment(branchEnv)
			for i, p := range f.Parameters {
				if i < len(fnArgs) {
					extEnv.Set(p.Value, fnArgs[i])
				}
			}
			NewEvalContext("<go>").Eval(f.Body, extEnv)
		}()
	case *Builtin:
		go f.Fn(fnArgs...)
	default:
		return newErr("go: first argument must be a function")
	}
	return object.NILOBJ
}

func bSpawn(args ...object.Object) object.Object {
	if len(args) < 1 {
		return newErr("spawn expects at least 1 argument (function)")
	}
	fn := args[0]
	fnArgs := args[1:]

	switch f := fn.(type) {
	case *object.Function:
		future := object.NewFuture()
		// Clone the captured environment so the background goroutine reads a
		// snapshot instead of racing the caller, which keeps writing to it.
		branchEnv := f.Env.Clone()
		go func() {
			extEnv := object.NewEnclosedEnvironment(branchEnv)
			for i, p := range f.Parameters {
				if i < len(fnArgs) {
					extEnv.Set(p.Value, fnArgs[i])
				}
			}
			// A fresh EvalContext keeps the background call stack isolated
			// from the caller's (the tree-walker is not goroutine-safe).
			future.Val = NewEvalContext("<spawn>").Eval(f.Body, extEnv)
			close(future.Done)
		}()
		return future
	case *Builtin:
		future := object.NewFuture()
		go func() {
			future.Val = f.Fn(fnArgs...)
			close(future.Done)
		}()
		return future
	default:
		return newErr("spawn: first argument must be a function")
	}
}

func applyFn(fn object.Object, args []object.Object) object.Object {
	switch f := fn.(type) {
	case *object.Function:
		fnCtx := getEvalCtx(f)
		return fnCtx.applyFunction(f, args)
	case *Builtin:
		return f.Fn(args...)
	}
	return newErr("not callable")
}

func getEvalCtx(f *object.Function) *EvalContext {
	if ctx, ok := f.EvalCtx.(*EvalContext); ok {
		return ctx
	}
	return NewEvalContext("<lambda>")
}

func newErr(format string, a ...interface{}) *object.Error {
	msg := object.FormatMsg(format, a...)
	return &object.Error{Message: msg}
}
