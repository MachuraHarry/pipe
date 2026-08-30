package object

import (
	"fmt"
	"math"
	"strings"
)

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

func bAssertNear(args ...Object) Object {
	if len(args) != 2 && len(args) != 3 {
		return err("assert_near expects 2 or 3 arguments (expected, actual[, epsilon])")
	}
	a := toFloat(args[0])
	b := toFloat(args[1])
	eps := 1e-6
	if len(args) == 3 {
		eps = toFloat(args[2])
	}
	if math.Abs(a-b) > eps {
		return err(fmt.Sprintf("assertion failed: expected %s ≈ %s (epsilon %g)",
			args[0].Inspect(), args[1].Inspect(), eps))
	}
	return NILOBJ
}

func bAssertContains(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_contains expects 2 arguments (container, item)")
	}
	switch c := args[0].(type) {
	case *String:
		needle, ok := args[1].(*String)
		if !ok {
			return err("assert_contains: substring check requires a string item")
		}
		if !strings.Contains(c.Value, needle.Value) {
			return err(fmt.Sprintf("assertion failed: %s does not contain %s", c.Inspect(), needle.Inspect()))
		}
	case *List:
		for _, el := range c.Elements {
			if ValuesEqual(el, args[1]) {
				return NILOBJ
			}
		}
		return err(fmt.Sprintf("assertion failed: list does not contain %s", args[1].Inspect()))
	case *Map:
		needle, ok := args[1].(*String)
		if !ok {
			return err("assert_contains: map key check requires a string key")
		}
		if _, exists := c.Get(needle.Value); !exists {
			return err(fmt.Sprintf("assertion failed: map does not contain key %s", needle.Inspect()))
		}
	default:
		return err(fmt.Sprintf("assert_contains expects a string, list, or map, got %s", args[0].Type()))
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
