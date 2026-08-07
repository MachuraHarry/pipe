package object

import "math"

func bAbs(args ...Object) Object {
	if len(args) != 1 {
		return err("abs expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		if v.Value < 0 {
			return &Integer{Value: -v.Value}
		}
		return &Integer{Value: v.Value}
	case *Float:
		return &Float{Value: math.Abs(v.Value)}
	}
	return err("abs expects a number")
}

func bMin(args ...Object) Object {
	if len(args) < 2 {
		return err("min expects at least 2 arguments")
	}
	f, ok := ToFloat(args[0])
	if !ok {
		return err("min: arguments must be numbers")
	}
	allInt := true
	for _, a := range args {
		if _, isI := a.(*Float); isI {
			allInt = false
		}
		af, ok := ToFloat(a)
		if !ok {
			return err("min: arguments must be numbers")
		}
		if af < f {
			f = af
		}
	}
	if allInt {
		return &Integer{Value: int64(f)}
	}
	return &Float{Value: f}
}

func bMax(args ...Object) Object {
	if len(args) < 2 {
		return err("max expects at least 2 arguments")
	}
	f, ok := ToFloat(args[0])
	if !ok {
		return err("max: arguments must be numbers")
	}
	allInt := true
	for _, a := range args {
		if _, isI := a.(*Float); isI {
			allInt = false
		}
		af, ok := ToFloat(a)
		if !ok {
			return err("max: arguments must be numbers")
		}
		if af > f {
			f = af
		}
	}
	if allInt {
		return &Integer{Value: int64(f)}
	}
	return &Float{Value: f}
}

func bPow(args ...Object) Object {
	if len(args) != 2 {
		return err("pow expects 2 arguments")
	}
	b, ok1 := ToFloat(args[0])
	e, ok2 := ToFloat(args[1])
	if !ok1 || !ok2 {
		return err("pow: arguments must be numbers")
	}
	return &Float{Value: math.Pow(b, e)}
}

func bSqrt(args ...Object) Object {
	if len(args) != 1 {
		return err("sqrt expects 1 argument")
	}
	v, ok := ToFloat(args[0])
	if !ok {
		return err("sqrt expects a number")
	}
	if v < 0 {
		return err("sqrt: negative number")
	}
	return &Float{Value: math.Sqrt(v)}
}

func bRound(args ...Object) Object {
	if len(args) != 1 {
		return err("round expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return &Integer{Value: v.Value}
	case *Float:
		return &Integer{Value: int64(math.Round(v.Value))}
	}
	return err("round expects a number")
}

func bCeil(args ...Object) Object {
	if len(args) != 1 {
		return err("ceil expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return &Integer{Value: v.Value}
	case *Float:
		return &Integer{Value: int64(math.Ceil(v.Value))}
	}
	return err("ceil expects a number")
}

func bFloor(args ...Object) Object {
	if len(args) != 1 {
		return err("floor expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return &Integer{Value: v.Value}
	case *Float:
		return &Integer{Value: int64(math.Floor(v.Value))}
	}
	return err("floor expects a number")
}
