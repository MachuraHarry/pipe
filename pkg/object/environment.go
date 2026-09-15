package object

type Environment struct {
	store map[string]Object
	outer *Environment
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) SetParent(parent *Environment) {
	e.outer = parent
}

func (e *Environment) HasLocal(name string) bool {
	_, ok := e.store[name]
	return ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

// Assign implements reassignment semantics for `name: value`: it mutates the
// nearest enclosing binding of name if one already exists in an intermediate
// (function-call) scope, so a closure's captured variables persist across
// calls instead of being shadowed by a fresh local on every call. It never
// writes into the root scope (outer == nil, i.e. true global/module scope)
// unless name is already declared in the current scope itself — this keeps
// "a function shadows a same-named global" behavior unchanged.
func (e *Environment) Assign(name string, val Object) Object {
	if e.HasLocal(name) {
		return e.Set(name, val)
	}
	for outer := e.outer; outer != nil && outer.outer != nil; outer = outer.outer {
		if outer.HasLocal(name) {
			return outer.Set(name, val)
		}
	}
	return e.Set(name, val)
}

func (e *Environment) Store() map[string]Object {
	return e.store
}

func (e *Environment) Delete(name string) {
	delete(e.store, name)
}

func (e *Environment) Copy() *Environment {
	s := make(map[string]Object, len(e.store))
	for k, v := range e.store {
		s[k] = v
	}
	return &Environment{store: s, outer: e.outer}
}

// Clone returns a snapshot of the whole environment chain. Used to isolate
// parallel pipeline branches (>>) from concurrent writes to the caller's
// environment.
func (e *Environment) Clone() *Environment {
	cp := e.Copy()
	if e.outer != nil {
		cp.outer = e.outer.Clone()
	}
	return cp
}
