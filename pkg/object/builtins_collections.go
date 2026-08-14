package object

import (
	"fmt"
	"sort"
)

func bLen(args ...Object) Object {
	if len(args) != 1 {
		return err("len expects 1 argument")
	}
	switch a := args[0].(type) {
	case *String:
		return &Integer{Value: int64(len(a.Value))}
	case *Bytes:
		return &Integer{Value: int64(len(a.Value))}
	case *List:
		return &Integer{Value: int64(len(a.Elements))}
	case *Map:
		return &Integer{Value: int64(len(a.Pairs))}
	}
	t := "nil"
	if args[0] != nil {
		t = string(args[0].Type())
	}
	return err(fmt.Sprintf("len not supported for type %s", t))
}

func bPush(args ...Object) Object {
	if len(args) < 2 {
		return err("push expects at least 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("push: first argument must be a list")
	}
	l.Elements = append(l.Elements, args[1:]...)
	return l
}

func bPop(args ...Object) Object {
	if len(args) != 1 {
		return err("pop expects 1 argument")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("pop expects a list")
	}
	if len(l.Elements) == 0 {
		return NILOBJ
	}
	last := l.Elements[len(l.Elements)-1]
	l.Elements = l.Elements[:len(l.Elements)-1]
	return last
}

func bAt(args ...Object) Object {
	if len(args) != 2 {
		return err("at expects 2 arguments")
	}
	idx, ok := ToInt(args[1])
	if !ok {
		return err("at: Index must be a number")
	}
	switch c := args[0].(type) {
	case *List:
		if idx < 0 || idx >= int64(len(c.Elements)) {
			return NILOBJ
		}
		return c.Elements[idx]
	case *String:
		if idx < 0 || idx >= int64(len(c.Value)) {
			return NILOBJ
		}
		return &String{Value: string(c.Value[idx])}
	case *Bytes:
		if idx < 0 || idx >= int64(len(c.Value)) {
			return NILOBJ
		}
		return &Integer{Value: int64(c.Value[idx])}
	}
	return err("at expects list or string")
}

func bSliceList(args ...Object) Object {
	return bSlice(args...)
}

func bSort(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return err("sort expects 1 or 2 arguments (list[, comparator])")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("sort expects list")
	}
	sorted := make([]Object, len(l.Elements))
	copy(sorted, l.Elements)
	if len(args) == 2 {
		cmp := args[1]
		sort.SliceStable(sorted, func(i, j int) bool {
			return IsTruthy(callTwo(cmp, sorted[i], sorted[j]))
		})
		return &List{Elements: sorted}
	}
	allNumeric := true
	for _, e := range sorted {
		if _, ok := e.(*Integer); !ok {
			if _, ok := e.(*Float); !ok {
				allNumeric = false
				break
			}
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		if allNumeric {
			af, _ := ToFloat(sorted[i])
			bf, _ := ToFloat(sorted[j])
			return af < bf
		}
		return sorted[i].Inspect() < sorted[j].Inspect()
	})
	return &List{Elements: sorted}
}

func bMap(args ...Object) Object {
	if len(args) != 2 {
		return err("map expects 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("map: first argument must be a list")
	}
	result := make([]Object, len(l.Elements))
	for i, e := range l.Elements {
		r := callOne(args[1], e)
		if r != nil && r.Type() == ERROR {
			return r
		}
		result[i] = r
	}
	return &List{Elements: result}
}

func bFilter(args ...Object) Object {
	if len(args) != 2 {
		return err("filter expects 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("filter: first argument must be a list")
	}
	var result []Object
	for _, e := range l.Elements {
		r := callOne(args[1], e)
		if r != nil && r.Type() == ERROR {
			return r
		}
		if IsTruthy(r) {
			result = append(result, e)
		}
	}
	return &List{Elements: result}
}

func bReduce(args ...Object) Object {
	if len(args) != 3 {
		return err("reduce expects 3 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("reduce: first argument must be a list")
	}
	acc := args[2]
	for _, e := range l.Elements {
		acc = callTwo(args[1], acc, e)
		if acc != nil && acc.Type() == ERROR {
			return acc
		}
	}
	return acc
}

func bEach(args ...Object) Object {
	if len(args) != 2 {
		return err("each expects 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("each: first argument must be a list")
	}
	for _, e := range l.Elements {
		callOne(args[1], e)
	}
	return NILOBJ
}

func bUnique(args ...Object) Object {
	if len(args) != 1 {
		return err("unique expects 1 argument (list)")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("unique: argument must be a list")
	}
	seen := make(map[string]bool)
	result := &List{}
	for _, e := range l.Elements {
		key := string(e.Type()) + ":" + e.Inspect()
		if !seen[key] {
			seen[key] = true
			result.Elements = append(result.Elements, e)
		}
	}
	return result
}

func callOne(fn, arg Object) Object {
	return CallUserFunction(fn, arg)
}

func callTwo(fn, a, b Object) Object {
	return CallUserFunction(fn, a, b)
}

// bGo launches a function fire-and-forget in the background. It handles the
// VM's function representations (Closure and BuiltinInfo); the tree-walker
// installs its own `go` builtin in pkg/eval.
func bGo(args ...Object) Object {
	if len(args) < 1 {
		return err("go expects at least 1 argument (function)")
	}
	fn := args[0]
	fnArgs := args[1:]

	switch f := fn.(type) {
	case *Closure:
		if sp, ok := f.Executor.(UserFunctionSpawner); ok {
			sp.SpawnUserFunction(f, fnArgs...)
			return NILOBJ
		}
		return err("go: first argument must be a function")
	case *BuiltinInfo:
		go f.Fn(fnArgs...)
	default:
		return err("go: first argument must be a function")
	}
	return NILOBJ
}

func bRange(args ...Object) Object {
	if len(args) < 1 || len(args) > 3 {
		return err("range expects 1-3 arguments")
	}
	var start, end, step int64
	step = 1

	switch len(args) {
	case 1:
		n, ok := ToInt(args[0])
		if !ok {
			return err("range: Argument must be a number")
		}
		start = 0
		end = n
	case 2:
		s, ok1 := ToInt(args[0])
		e, ok2 := ToInt(args[1])
		if !ok1 || !ok2 {
			return err("range: arguments must be numbers")
		}
		start = s
		end = e
	case 3:
		s, ok1 := ToInt(args[0])
		e, ok2 := ToInt(args[1])
		st, ok3 := ToInt(args[2])
		if !ok1 || !ok2 || !ok3 {
			return err("range: arguments must be numbers")
		}
		start = s
		end = e
		step = st
	}

	if step == 0 {
		return err("range: step must not be zero")
	}

	var elems []Object
	for i := start; i < end; i += step {
		elems = append(elems, &Integer{Value: i})
	}
	return &List{Elements: elems}
}

func bKeys(args ...Object) Object {
	if len(args) != 1 {
		return err("keys expects 1 argument")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("keys expects a map")
	}
	keys := make([]Object, 0, len(m.Pairs))
	for k := range m.Pairs {
		keys = append(keys, &String{Value: k})
	}
	return &List{Elements: keys}
}

func bValues(args ...Object) Object {
	if len(args) != 1 {
		return err("values expects 1 argument")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("values expects a map")
	}
	vals := make([]Object, 0, len(m.Pairs))
	for _, v := range m.Pairs {
		vals = append(vals, v)
	}
	return &List{Elements: vals}
}

func bGet(args ...Object) Object {
	if len(args) != 2 {
		return err("get expects 2 arguments (Map/List, Key/Index)")
	}
	switch container := args[0].(type) {
	case *Map:
		key, ok := args[1].(*String)
		if !ok {
			return err("get: Map-Key must be a string")
		}
		val, exists := container.Pairs[key.Value]
		if !exists {
			return NILOBJ
		}
		return val
	case *List:
		idx, ok := ToInt(args[1])
		if !ok {
			return err("get: Listen-Index must be a number")
		}
		if idx < 0 || idx >= int64(len(container.Elements)) {
			return NILOBJ
		}
		return container.Elements[idx]
	}
	return err("get expects a map or list")
}

func bSet(args ...Object) Object {
	if len(args) != 3 {
		return err("set expects 3 arguments (Map/List, Key/Index, Value)")
	}
	switch container := args[0].(type) {
	case *Map:
		key, ok := args[1].(*String)
		if !ok {
			return err("set: Map-Key must be a string")
		}
		if container.Pairs == nil {
			container.Pairs = make(map[string]Object)
		}
		container.Pairs[key.Value] = args[2]
		return container
	case *List:
		idx, ok := ToInt(args[1])
		if !ok {
			return err("set: List-Index must be a number")
		}
		if idx < 0 || idx >= int64(len(container.Elements)) {
			return err("set: index out of bounds")
		}
		container.Elements[idx] = args[2]
		return container
	}
	return err("set: first argument must be a map or list")
}

func bMapDelete(args ...Object) Object {
	if len(args) != 2 {
		return err("map_delete expects 2 arguments (Map, Key)")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("map_delete: first argument must be a map")
	}
	key, ok := args[1].(*String)
	if !ok {
		return err("map_delete: key must be a string")
	}
	delete(m.Pairs, key.Value)
	return NILOBJ
}
