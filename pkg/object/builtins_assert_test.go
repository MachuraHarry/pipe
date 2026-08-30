package object

import "testing"

func assertOutcome(t *testing.T, res Object, wantErr bool) {
	t.Helper()
	if wantErr && res.Type() != ERROR {
		t.Errorf("expected an error result, got %v", res)
	}
	if !wantErr && res.Type() == ERROR {
		t.Errorf("expected pass, got %s", res.Inspect())
	}
}

func TestAssertNear(t *testing.T) {
	tests := []struct {
		args    []Object
		wantErr bool
	}{
		{[]Object{&Float{Value: 3.14159}, &Float{Value: 3.14159}}, false},
		{[]Object{&Float{Value: 3.1415926}, &Float{Value: 3.14159}, &Float{Value: 1e-4}}, false},
		{[]Object{&Integer{Value: 10}, &Integer{Value: 11}, &Integer{Value: 2}}, false},
		{[]Object{&Integer{Value: 1}, &Integer{Value: 1}}, false},
		{[]Object{&Float{Value: 0.0}, &Float{Value: 0.5}}, true},
		{[]Object{&Float{Value: 3.14159}, &Float{Value: 3.2}}, true},
		{[]Object{&Integer{Value: 1}}, true}, // wrong arity
	}
	for _, tt := range tests {
		res := bAssertNear(tt.args...)
		assertOutcome(t, res, tt.wantErr)
	}
}

func TestAssertContains(t *testing.T) {
	hi := &String{Value: "hello world"}
	lst := &List{Elements: []Object{&Integer{Value: 1}, &String{Value: "two"}}}
	mp := MapFromGo(map[string]Object{"name": &String{Value: "pipe"}})

	tests := []struct {
		args    []Object
		wantErr bool
	}{
		{[]Object{hi, &String{Value: "world"}}, false},
		{[]Object{hi, &String{Value: "xyz"}}, true},
		{[]Object{lst, &String{Value: "two"}}, false},
		{[]Object{lst, &Integer{Value: 2}}, true},
		{[]Object{mp, &String{Value: "name"}}, false},
		{[]Object{mp, &String{Value: "missing"}}, true},
		{[]Object{hi, &Integer{Value: 1}}, true},                 // substring check needs a string
		{[]Object{&Integer{Value: 5}, &Integer{Value: 5}}, true}, // unsupported container
		{[]Object{hi}, true},                                     // wrong arity
	}
	for _, tt := range tests {
		res := bAssertContains(tt.args...)
		assertOutcome(t, res, tt.wantErr)
	}
}
