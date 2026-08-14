package object

import "fmt"

func bAssert(args ...Object) Object {
	if len(args) != 1 {
		return err("assert expects 1 argument (condition)")
	}
	if !IsTruthy(args[0]) {
		return err("assertion failed: value is not truthy")
	}
	return NILOBJ
}

func bAssertEq(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_eq expects 2 arguments (expected, actual)")
	}
	if !ValuesEqual(args[0], args[1]) {
		return err(fmt.Sprintf("assertion failed: expected %s, got %s", args[0].Inspect(), args[1].Inspect()))
	}
	return NILOBJ
}

func bAssertNotEq(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_not_eq expects 2 arguments (unexpected, actual)")
	}
	if ValuesEqual(args[0], args[1]) {
		return err(fmt.Sprintf("assertion failed: got %s, but expected different value", args[0].Inspect()))
	}
	return NILOBJ
}

func bAssertLt(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_lt expects 2 arguments (a, b)")
	}
	a := toFloat(args[0])
	b := toFloat(args[1])
	if a >= b {
		return err(fmt.Sprintf("assertion failed: expected %s < %s", args[0].Inspect(), args[1].Inspect()))
	}
	return NILOBJ
}

func bAssertGt(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_gt expects 2 arguments (a, b)")
	}
	a := toFloat(args[0])
	b := toFloat(args[1])
	if a <= b {
		return err(fmt.Sprintf("assertion failed: expected %s > %s", args[0].Inspect(), args[1].Inspect()))
	}
	return NILOBJ
}

func bAssertError(args ...Object) Object {
	if len(args) != 1 {
		return err("assert_error expects 1 argument (function)")
	}
	switch args[0].(type) {
	case *Function, *Closure:
	default:
		return err("assert_error expects a function (use { ... })")
	}
	result := CallUserFunction(args[0])
	if result == nil {
		return err("assertion failed: expected an error, but got nil")
	}
	if result.Type() != ERROR {
		return err(fmt.Sprintf("assertion failed: expected an error, but got %s", result.Inspect()))
	}
	return NILOBJ
}

func toFloat(o Object) float64 {
	switch v := o.(type) {
	case *Integer:
		return float64(v.Value)
	case *Float:
		return v.Value
	default:
		return 0
	}
}
