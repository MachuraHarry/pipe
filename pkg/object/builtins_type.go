package object

import (
	"math"
	"strconv"
	"strings"
)

func bTypeOf(args ...Object) Object {
	if len(args) != 1 {
		return err("type_of expects 1 argument")
	}
	if args[0] == nil {
		return &String{Value: "NIL"}
	}
	return &String{Value: string(args[0].Type())}
}

func bIsNum(args ...Object) Object {
	if len(args) != 1 {
		return err("is_num expects 1 argument")
	}
	_, i := args[0].(*Integer)
	_, f := args[0].(*Float)
	return NativeBoolToBoolean(i || f)
}

func bIsStr(args ...Object) Object {
	if len(args) != 1 {
		return err("is_str expects 1 argument")
	}
	_, ok := args[0].(*String)
	return NativeBoolToBoolean(ok)
}

func bIsList(args ...Object) Object {
	if len(args) != 1 {
		return err("is_list expects 1 argument")
	}
	_, ok := args[0].(*List)
	return NativeBoolToBoolean(ok)
}

func bIsMap(args ...Object) Object {
	if len(args) != 1 {
		return err("is_map expects 1 argument")
	}
	_, ok := args[0].(*Map)
	return NativeBoolToBoolean(ok)
}

func bIsNil(args ...Object) Object {
	if len(args) != 1 {
		return err("is_nil expects 1 argument")
	}
	return NativeBoolToBoolean(args[0] == NILOBJ)
}

func bToStr(args ...Object) Object {
	if len(args) != 1 {
		return err("to_str expects 1 argument")
	}
	if args[0] == nil {
		return &String{Value: "nil"}
	}
	return &String{Value: args[0].Inspect()}
}

func bToNum(args ...Object) Object {
	if len(args) != 1 {
		return err("to_num expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return v
	case *Float:
		return v
	case *String:
		f, e := strconv.ParseFloat(v.Value, 64)
		if e != nil {
			return err("to_num: '" + v.Value + "' is not a number")
		}
		if f >= math.MinInt64 && f <= math.MaxInt64 && f == float64(int64(f)) && !strings.Contains(v.Value, ".") {
			return &Integer{Value: int64(f)}
		}
		return &Float{Value: f}
	case *Boolean:
		if v.Value {
			return &Integer{Value: 1}
		}
		return &Integer{Value: 0}
	}
	return err("to_num not possible")
}
