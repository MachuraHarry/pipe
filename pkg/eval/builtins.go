package eval

import (
	"github.com/harry/pipe/pkg/object"
)

type Builtin struct {
	Fn func(args ...object.Object) object.Object
}

func (b *Builtin) Type() object.ObjectType { return "BUILTIN" }
func (b *Builtin) Inspect() string         { return "builtin function" }

var builtins = map[string]*Builtin{}

func init() {
	for _, bi := range object.Builtins {
		bi := bi
		builtins[bi.Name] = &Builtin{Fn: bi.Fn}
	}

	builtins["map"] = &Builtin{Fn: bMap}
	builtins["filter"] = &Builtin{Fn: bFilter}
	builtins["reduce"] = &Builtin{Fn: bReduce}
	builtins["each"] = &Builtin{Fn: bEach}
	builtins["go"] = &Builtin{Fn: bGo}
}

func bMap(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newErr("map erwartet 2 Argumente (Liste, Funktion)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("map: erstes Argument muss eine List sein")
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
		return newErr("filter erwartet 2 Argumente (Liste, Funktion)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("filter: erstes Argument muss eine List sein")
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
		return newErr("reduce erwartet 3 Argumente (Liste, Funktion, Startwert)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("reduce: erstes Argument muss eine List sein")
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
		return newErr("each erwartet 2 Argumente (Liste, Funktion)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newErr("each: erstes Argument muss eine List sein")
	}
	fn := args[1]
	for _, elem := range list.Elements {
		applyFn(fn, []object.Object{elem})
	}
	return object.NILOBJ
}

func bGo(args ...object.Object) object.Object {
	if len(args) < 1 {
		return newErr("go erwartet mindestens 1 Argument (Funktion)")
	}
	fn := args[0]
	fnArgs := args[1:]

	switch f := fn.(type) {
	case *object.Function:
		go func() {
			extEnv := object.NewEnclosedEnvironment(f.Env)
			for i, p := range f.Parameters {
				if i < len(fnArgs) {
					extEnv.Set(p.Value, fnArgs[i])
				}
			}
			fnCtx := getEvalCtx(f)
			fnCtx.Eval(f.Body, extEnv)
		}()
	case *Builtin:
		go f.Fn(fnArgs...)
	default:
		return newErr("go: erstes Argument muss eine Funktion sein")
	}
	return object.NILOBJ
}

func applyFn(fn object.Object, args []object.Object) object.Object {
	switch f := fn.(type) {
	case *object.Function:
		fnCtx := getEvalCtx(f)
		return fnCtx.applyFunction(f, args)
	case *Builtin:
		return f.Fn(args...)
	}
	return newErr("nicht aufrufbar")
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
