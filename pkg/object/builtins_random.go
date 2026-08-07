package object

import "math/rand"

func bRandom(args ...Object) Object {
	_ = args
	return &Float{Value: rand.Float64()}
}

func bRandomRange(args ...Object) Object {
	if len(args) != 2 {
		return err("random_range expects 2 arguments (Min, Max)")
	}
	min, ok1 := ToInt(args[0])
	max, ok2 := ToInt(args[1])
	if !ok1 || !ok2 {
		return err("random_range: Min und Max must be numbers")
	}
	if min >= max {
		return err("random_range: min must be less than max")
	}
	return &Integer{Value: min + rand.Int63n(max-min)}
}
