package eval

import (
	"github.com/harry/pulse/pkg/object"
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

	// Complex builtins that need the evaluator
	builtins["map"] = &Builtin{Fn: bMap}
	builtins["filter"] = &Builtin{Fn: bFilter}
	builtins["reduce"] = &Builtin{Fn: bReduce}
	builtins["each"] = &Builtin{Fn: bEach}
	builtins["go"] = &Builtin{Fn: bGo}
}

func bMap(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("map erwartet 2 Argumente (Liste, Funktion)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newError("map: erstes Argument muss eine List sein")
	}
	result := make([]object.Object, len(list.Elements))
	for i, elem := range list.Elements {
		result[i] = applyFunction(args[1], []object.Object{elem})
		if result[i] != nil && result[i].Type() == object.ERROR {
			return result[i]
		}
	}
	return &object.List{Elements: result}
}

func bFilter(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("filter erwartet 2 Argumente (Liste, Funktion)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newError("filter: erstes Argument muss eine List sein")
	}
	var result []object.Object
	for _, elem := range list.Elements {
		r := applyFunction(args[1], []object.Object{elem})
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
		return newError("reduce erwartet 3 Argumente (Liste, Funktion, Startwert)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newError("reduce: erstes Argument muss eine List sein")
	}
	acc := args[2]
	for _, elem := range list.Elements {
		acc = applyFunction(args[1], []object.Object{acc, elem})
		if acc != nil && acc.Type() == object.ERROR {
			return acc
		}
	}
	return acc
}

func bEach(args ...object.Object) object.Object {
	if len(args) != 2 {
		return newError("each erwartet 2 Argumente (Liste, Funktion)")
	}
	list, ok := args[0].(*object.List)
	if !ok {
		return newError("each: erstes Argument muss eine List sein")
	}
	for _, elem := range list.Elements {
		applyFunction(args[1], []object.Object{elem})
	}
	return object.NILOBJ
}

func bGo(args ...object.Object) object.Object {
	if len(args) < 1 {
		return newError("go erwartet mindestens 1 Argument (Funktion)")
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
			Eval(f.Body, extEnv)
		}()
	case *Builtin:
		go f.Fn(fnArgs...)
	default:
		return newError("go: erstes Argument muss eine Funktion sein")
	}
	return object.NILOBJ
}
