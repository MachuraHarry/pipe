package object

import (
	"io"
	"os"
	"testing"
)

func TestIntegerType(t *testing.T) {
	obj := &Integer{Value: 42}
	if obj.Type() != INTEGER {
		t.Errorf("Type = %s, want INTEGER", obj.Type())
	}
	if obj.Inspect() != "42" {
		t.Errorf("Inspect = %s, want 42", obj.Inspect())
	}
}

func TestFloatType(t *testing.T) {
	obj := &Float{Value: 3.14}
	if obj.Type() != FLOAT {
		t.Errorf("Type = %s, want FLOAT", obj.Type())
	}
	if obj.Inspect() != "3.14" {
		t.Errorf("Inspect = %s, want 3.14", obj.Inspect())
	}
}

func TestStringType(t *testing.T) {
	obj := &String{Value: "hello"}
	if obj.Type() != STRING {
		t.Errorf("Type = %s, want STRING", obj.Type())
	}
	if obj.Inspect() != "hello" {
		t.Errorf("Inspect = %s, want hello", obj.Inspect())
	}
}

func TestBooleanType(t *testing.T) {
	obj := &Boolean{Value: true}
	if obj.Type() != BOOLEAN {
		t.Errorf("Type = %s, want BOOLEAN", obj.Type())
	}
	if obj.Inspect() != "true" {
		t.Errorf("Inspect = %s, want true", obj.Inspect())
	}

	obj2 := &Boolean{Value: false}
	if obj2.Inspect() != "false" {
		t.Errorf("Inspect = %s, want false", obj2.Inspect())
	}
}

func TestNilType(t *testing.T) {
	obj := &NilObject{}
	if obj.Type() != NIL {
		t.Errorf("Type = %s, want NIL", obj.Type())
	}
	if obj.Inspect() != "nil" {
		t.Errorf("Inspect = %s, want nil", obj.Inspect())
	}
}

func TestNativeBoolToBoolean(t *testing.T) {
	b := NativeBoolToBoolean(true)
	if b.Value != true || b.Type() != BOOLEAN {
		t.Errorf("NativeBoolToBoolean(true) = %v, want true boolean", b)
	}
	b = NativeBoolToBoolean(false)
	if b.Value != false {
		t.Errorf("NativeBoolToBoolean(false) = %v, want false", b)
	}
}

func TestErrorType(t *testing.T) {
	obj := &Error{Message: "something went wrong"}
	if obj.Type() != ERROR {
		t.Errorf("Type = %s, want ERROR", obj.Type())
	}
	if obj.Inspect() != "something went wrong" {
		t.Errorf("Inspect = %s, want something went wrong", obj.Inspect())
	}
}

func TestListType(t *testing.T) {
	obj := &List{
		Elements: []Object{
			&Integer{Value: 1},
			&String{Value: "two"},
		},
	}
	if obj.Type() != LIST {
		t.Errorf("Type = %s, want LIST", obj.Type())
	}
	insp := obj.Inspect()
	if insp != "[1, two]" {
		t.Errorf("Inspect = %s, want [1, two]", insp)
	}

	empty := &List{Elements: []Object{}}
	if empty.Inspect() != "[]" {
		t.Errorf("empty List Inspect = %s, want []", empty.Inspect())
	}
}

func TestMapType(t *testing.T) {
	obj := MapFromGo(map[string]Object{
		"a": &Integer{Value: 1},
		"b": &String{Value: "two"},
	})
	if obj.Type() != MAP {
		t.Errorf("Type = %s, want MAP", obj.Type())
	}
	insp := obj.Inspect()
	if insp != `{a: 1, b: two}` {
		t.Errorf("Inspect = %s, want {a: 1, b: two} (deterministic sorted order)", insp)
	}

	empty := MapFromGo(map[string]Object{})
	if empty.Inspect() != "{}" {
		t.Errorf("empty Map Inspect = %s, want {}", empty.Inspect())
	}
}

func TestFunctionType(t *testing.T) {
	obj := &Function{
		Name: "add",
	}
	if obj.Type() != FUNCTION {
		t.Errorf("Type = %s, want FUNCTION", obj.Type())
	}
	if obj.Inspect() != "fn()" {
		t.Errorf("Inspect = %s, want fn()", obj.Inspect())
	}

	anon := &Function{}
	if anon.Inspect() != "fn()" {
		t.Errorf("Inspect = %s, want fn()", anon.Inspect())
	}
}

func TestBuiltinInfoType(t *testing.T) {
	obj := &BuiltinInfo{Name: "test", Fn: func(args ...Object) Object { return NILOBJ }}
	if obj.Type() != ObjectType("BUILTIN") {
		t.Errorf("Type = %s, want BUILTIN", obj.Type())
	}
	if obj.Inspect() != "builtin: test" {
		t.Errorf("Inspect = %s, want builtin: test", obj.Inspect())
	}
}

func TestCompiledFunctionType(t *testing.T) {
	obj := &CompiledFunction{}
	if obj.Type() != COMPILED_FUNCTION {
		t.Errorf("Type = %s, want COMPILED_FUNCTION", obj.Type())
	}
	if obj.Inspect() != "compiled function" {
		t.Errorf("Inspect = %s, want compiled function", obj.Inspect())
	}
}

func TestClosureType(t *testing.T) {
	obj := &Closure{}
	if obj.Type() != CLOSURE {
		t.Errorf("Type = %s, want CLOSURE", obj.Type())
	}
	if obj.Inspect() != "closure" {
		t.Errorf("Inspect = %s, want closure", obj.Inspect())
	}
}

func TestFutureType(t *testing.T) {
	obj := &Future{}
	if obj.Type() != FUTURE {
		t.Errorf("Type = %s, want FUTURE", obj.Type())
	}
}

func TestEnvironment(t *testing.T) {
	env := NewEnvironment()
	if env == nil {
		t.Fatal("NewEnvironment returned nil")
	}

	// Set and get
	val := &Integer{Value: 42}
	env.Set("x", val)
	got, ok := env.Get("x")
	if !ok {
		t.Fatal("Get(x) returned not ok")
	}
	if got.Inspect() != "42" {
		t.Errorf("Get(x) = %s, want 42", got.Inspect())
	}

	// Unknown variable
	_, ok = env.Get("nonexistent")
	if ok {
		t.Errorf("Get(nonexistent) should return false")
	}

	// Overwrite
	val2 := &String{Value: "forty-two"}
	env.Set("x", val2)
	got, ok = env.Get("x")
	if !ok || got.Inspect() != "forty-two" {
		t.Errorf("Get(x) after overwrite = %s, want forty-two", got.Inspect())
	}
}

func TestEnclosedEnvironment(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &Integer{Value: 10})
	outer.Set("y", &Integer{Value: 20})

	inner := NewEnclosedEnvironment(outer)
	if inner == nil {
		t.Fatal("NewEnclosedEnvironment returned nil")
	}

	// Read from outer
	got, ok := inner.Get("x")
	if !ok || got.Inspect() != "10" {
		t.Errorf("inner.Get(x) = %v, want 10", got)
	}

	// Shadow in inner
	inner.Set("x", &Integer{Value: 99})
	got, ok = inner.Get("x")
	if !ok || got.Inspect() != "99" {
		t.Errorf("inner.Get(x) after shadow = %s, want 99", got.Inspect())
	}
	// Outer should still have old value
	got, ok = outer.Get("x")
	if !ok || got.Inspect() != "10" {
		t.Errorf("outer.Get(x) after shadow = %s, want 10", got.Inspect())
	}
}

func TestEnsureResolved(t *testing.T) {
	resolved := EnsureResolved(&Integer{Value: 5})
	if resolved.Inspect() != "5" {
		t.Errorf("EnsureResolved(integer) = %s, want 5", resolved.Inspect())
	}

	// Future that resolves to integer
	future := NewFuture()
	future.Val = &Integer{Value: 42}
	close(future.Done)
	resolved = EnsureResolved(future)
	if resolved.Inspect() != "42" {
		t.Errorf("EnsureResolved(future) = %s, want 42", resolved.Inspect())
	}
}

func TestObjectTypeConstants(t *testing.T) {
	if INTEGER != ObjectType("INTEGER") {
		t.Errorf("INTEGER constant mismatch")
	}
	if FLOAT != ObjectType("FLOAT") {
		t.Errorf("FLOAT constant mismatch")
	}
	if STRING != ObjectType("STRING") {
		t.Errorf("STRING constant mismatch")
	}
	if BOOLEAN != ObjectType("BOOLEAN") {
		t.Errorf("BOOLEAN constant mismatch")
	}
	if NIL != ObjectType("NIL") {
		t.Errorf("NIL constant mismatch")
	}
	if FUNCTION != ObjectType("FUNCTION") {
		t.Errorf("FUNCTION constant mismatch")
	}
	if COMPILED_FUNCTION != ObjectType("COMPILED_FUNCTION") {
		t.Errorf("COMPILED_FUNCTION constant mismatch")
	}
	if CLOSURE != ObjectType("CLOSURE") {
		t.Errorf("CLOSURE constant mismatch")
	}
	if LIST != ObjectType("LIST") {
		t.Errorf("LIST constant mismatch")
	}
	if MAP != ObjectType("MAP") {
		t.Errorf("MAP constant mismatch")
	}
	if FUTURE != ObjectType("FUTURE") {
		t.Errorf("FUTURE constant mismatch")
	}
	if ERROR != ObjectType("ERROR") {
		t.Errorf("ERROR constant mismatch")
	}
	if ERROR != ObjectType("ERROR") {
		t.Errorf("ERROR constant mismatch")
	}
}

func TestTypeOfBuiltin(t *testing.T) {
	obj := MapFromGo(map[string]Object{
		"name": &String{Value: "test"},
	})
	result := bTypeOf(obj)
	s, ok := result.(*String)
	if !ok || s.Value != "MAP" {
		t.Errorf("bTypeOf(map) = %s, want MAP", result.Inspect())
	}

	result = bTypeOf(&Integer{Value: 1})
	s, ok = result.(*String)
	if !ok || s.Value != "INTEGER" {
		t.Errorf("bTypeOf(integer) = %s, want INTEGER", result.Inspect())
	}
}

func TestToStrBuiltin(t *testing.T) {
	result := bToStr(&Integer{Value: 42})
	s, ok := result.(*String)
	if !ok || s.Value != "42" {
		t.Errorf("bToStr(42) = %s, want 42", result.Inspect())
	}

	result = bToStr(&String{Value: "hello"})
	s, ok = result.(*String)
	if !ok || s.Value != "hello" {
		t.Errorf("bToStr(hello) = %s, want hello", result.Inspect())
	}

	result = bToStr(&Boolean{Value: true})
	s, ok = result.(*String)
	if !ok || s.Value != "true" {
		t.Errorf("bToStr(true) = %s, want true", result.Inspect())
	}

	result = bToStr(&NilObject{})
	s, ok = result.(*String)
	if !ok || s.Value != "nil" {
		t.Errorf("bToStr(nil) = %s, want nil", result.Inspect())
	}
}

func TestIsNumBuiltin(t *testing.T) {
	result := bIsNum(&Integer{Value: 42})
	b, ok := result.(*Boolean)
	if !ok || b.Value != true {
		t.Errorf("bIsNum(42) = %v, want true", result.Inspect())
	}

	result = bIsNum(&String{Value: "hello"})
	b, ok = result.(*Boolean)
	if !ok || b.Value != false {
		t.Errorf("bIsNum(hello) = %v, want false", result.Inspect())
	}
}

func TestIsStrBuiltin(t *testing.T) {
	result := bIsStr(&String{Value: "hello"})
	b, ok := result.(*Boolean)
	if !ok || b.Value != true {
		t.Errorf("bIsStr(hello) = %v, want true", result.Inspect())
	}

	result = bIsStr(&Integer{Value: 42})
	b, ok = result.(*Boolean)
	if !ok || b.Value != false {
		t.Errorf("bIsStr(42) = %v, want false", result.Inspect())
	}
}

func TestIsNilBuiltin(t *testing.T) {
	result := bIsNil(NILOBJ)
	b, ok := result.(*Boolean)
	if !ok || b.Value != true {
		t.Errorf("bIsNil(nil) = %v, want true", result.Inspect())
	}

	result = bIsNil(&Integer{Value: 1})
	b, ok = result.(*Boolean)
	if !ok || b.Value != false {
		t.Errorf("bIsNil(1) = %v, want false", result.Inspect())
	}
}

func TestStringBuiltins(t *testing.T) {
	// upper
	result := bUpper(&String{Value: "hello"})
	s, ok := result.(*String)
	if !ok || s.Value != "HELLO" {
		t.Errorf("bUpper(hello) = %s, want HELLO", result.Inspect())
	}
	// upper with wrong arg
	result = bUpper(&Integer{Value: 1})
	_, ok = result.(*Error)
	if !ok {
		t.Errorf("bUpper(1) should be error")
	}

	// lower
	result = bLower(&String{Value: "HELLO"})
	s, ok = result.(*String)
	if !ok || s.Value != "hello" {
		t.Errorf("bLower(HELLO) = %s, want hello", result.Inspect())
	}

	// trim
	result = bTrim(&String{Value: "  hello  "})
	s, ok = result.(*String)
	if !ok || s.Value != "hello" {
		t.Errorf("bTrim(spaces) = %s, want hello", result.Inspect())
	}

	// split
	result = bSplit(&String{Value: "a,b,c"}, &String{Value: ","})
	l, ok := result.(*List)
	if !ok || len(l.Elements) != 3 {
		t.Errorf("bSplit(a,b,c) = %v, want list of 3", result.Inspect())
	}

	// contains
	result = bContains(&String{Value: "hello world"}, &String{Value: "world"})
	b, ok := result.(*Boolean)
	if !ok || b.Value != true {
		t.Errorf("bContains(hello world, world) = %v, want true", result.Inspect())
	}

	result = bContains(&String{Value: "hello world"}, &String{Value: "xyz"})
	b, ok = result.(*Boolean)
	if !ok || b.Value != false {
		t.Errorf("bContains(hello world, xyz) = %v, want false", result.Inspect())
	}
}

func TestListBuiltins(t *testing.T) {
	// len
	list := &List{Elements: []Object{&Integer{Value: 1}, &Integer{Value: 2}}}
	result := bLen(list)
	i, ok := result.(*Integer)
	if !ok || i.Value != 2 {
		t.Errorf("bLen([1,2]) = %s, want 2", result.Inspect())
	}

	// push
	result = bPush(list, &Integer{Value: 3})
	newList, ok := result.(*List)
	if !ok || len(newList.Elements) != 3 {
		t.Errorf("bPush([1,2], 3) = %v, want list of 3", result.Inspect())
	}

	// pop
	list2 := &List{Elements: []Object{&Integer{Value: 10}, &Integer{Value: 20}}}
	result = bPop(list2)
	if result.Inspect() != "20" {
		t.Errorf("bPop([10,20]) = %s, want 20", result.Inspect())
	}
	if len(list2.Elements) != 1 {
		t.Errorf("after pop, len = %d, want 1", len(list2.Elements))
	}
}

func TestMathBuiltins(t *testing.T) {
	// abs
	result := bAbs(&Integer{Value: -5})
	i, ok := result.(*Integer)
	if !ok || i.Value != 5 {
		t.Errorf("bAbs(-5) = %s, want 5", result.Inspect())
	}

	// min
	result = bMin(&Integer{Value: 3}, &Integer{Value: 7})
	i, ok = result.(*Integer)
	if !ok || i.Value != 3 {
		t.Errorf("bMin(3,7) = %s, want 3", result.Inspect())
	}

	// max
	result = bMax(&Integer{Value: 3}, &Integer{Value: 7})
	i, ok = result.(*Integer)
	if !ok || i.Value != 7 {
		t.Errorf("bMax(3,7) = %s, want 7", result.Inspect())
	}

	// sqrt with integer
	result = bSqrt(&Integer{Value: 9})
	f, ok := result.(*Float)
	if !ok || f.Value != 3.0 {
		t.Errorf("bSqrt(9) = %s, want 3", result.Inspect())
	}

	// pow returns float
	result = bPow(&Integer{Value: 2}, &Integer{Value: 3})
	f, ok = result.(*Float)
	if !ok || f.Value != 8.0 {
		t.Errorf("bPow(2,3) = %s, want 8", result.Inspect())
	}
}

func TestMapBuiltins(t *testing.T) {
	m := MapFromGo(map[string]Object{"a": &Integer{Value: 1}})

	// get
	result := bGet(m, &String{Value: "a"})
	if result.Inspect() != "1" {
		t.Errorf("bGet(m, a) = %s, want 1", result.Inspect())
	}

	// get missing
	result = bGet(m, &String{Value: "b"})
	if result != NILOBJ {
		t.Errorf("bGet(m, b) = %s, want nil", result.Inspect())
	}

	// keys
	result = bKeys(m)
	list, ok := result.(*List)
	if !ok || len(list.Elements) != 1 {
		t.Errorf("bKeys(m) = %v, want list of 1", result.Inspect())
	}

	// values
	result = bValues(m)
	list, ok = result.(*List)
	if !ok || len(list.Elements) != 1 {
		t.Errorf("bValues(m) = %v, want list of 1", result.Inspect())
	}
}

func TestRangeBuiltin(t *testing.T) {
	result := bRange(&Integer{Value: 1}, &Integer{Value: 5})
	l, ok := result.(*List)
	if !ok || len(l.Elements) != 4 {
		t.Errorf("bRange(1,5) = %v, want list of 4", result.Inspect())
	}
	if l.Elements[0].Inspect() != "1" || l.Elements[3].Inspect() != "4" {
		t.Errorf("bRange(1,5) elements wrong: %v", l.Inspect())
	}

	result = bRange(&Integer{Value: 5})
	l, ok = result.(*List)
	if !ok || len(l.Elements) != 5 {
		t.Errorf("bRange(5) = %v, want list of 5", result.Inspect())
	}
}

func TestTimeBuiltins(t *testing.T) {
	result := bNow()
	_, ok := result.(*Integer)
	if !ok {
		t.Errorf("bNow() = %s, want integer timestamp", result.Inspect())
	}
}

func TestRandomBuiltin(t *testing.T) {
	result := bRandom()
	_, ok := result.(*Float)
	if !ok {
		t.Errorf("bRandom() = %s, want float", result.Inspect())
	}
}

func TestRandomRangeBuiltin(t *testing.T) {
	result := bRandomRange(&Integer{Value: 1}, &Integer{Value: 7})
	i, ok := result.(*Integer)
	if !ok || i.Value < 1 || i.Value >= 7 {
		t.Errorf("bRandomRange(1,7) = %s, want int in [1,6]", result.Inspect())
	}
}

func TestPrintRawExactOutput(t *testing.T) {
	if PrintHook != nil {
		t.Skip("PrintHook set by embedding host; direct stdout test not possible")
	}
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	bPrintRaw(&String{Value: "{\"a\":1}\n"})
	w.Close()
	out, _ := io.ReadAll(r)

	if got := string(out); got != "{\"a\":1}\n" {
		t.Fatalf("print_raw output = %q, want exact bytes without trailing space", got)
	}
}
