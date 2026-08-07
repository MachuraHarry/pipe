package object

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

func init() {
	Builtins = append(Builtins,
		BuiltinInfo{Name: "secure_random", Fn: bSecureRandom},
		BuiltinInfo{Name: "secure_random_int", Fn: bSecureRandomInt},
		BuiltinInfo{Name: "secure_random_range", Fn: bSecureRandomRange},
	)
}

func bSecureRandom(args ...Object) Object {
	if len(args) != 1 {
		return err("secure_random expects 1 argument (byte_count)")
	}
	count, ok := ToInt(args[0])
	if !ok || count <= 0 || count > 1024 {
		return err("secure_random: byte_count must be a number between 1 and 1024")
	}
	buf := make([]byte, count)
	if _, e := rand.Read(buf); e != nil {
		return err("secure_random: " + e.Error())
	}
	return &String{Value: hex.EncodeToString(buf)}
}

func bSecureRandomInt(args ...Object) Object {
	if len(args) > 0 {
		return err("secure_random_int takes no arguments")
	}
	buf := make([]byte, 8)
	if _, e := rand.Read(buf); e != nil {
		return err("secure_random_int: " + e.Error())
	}
	val := int64(buf[0])<<56 | int64(buf[1])<<48 | int64(buf[2])<<40 | int64(buf[3])<<32 |
		int64(buf[4])<<24 | int64(buf[5])<<16 | int64(buf[6])<<8 | int64(buf[7])
	return &Integer{Value: val}
}

func bSecureRandomRange(args ...Object) Object {
	if len(args) != 2 {
		return err("secure_random_range expects 2 arguments (min, max)")
	}
	min, ok1 := ToInt(args[0])
	max, ok2 := ToInt(args[1])
	if !ok1 || !ok2 {
		return err("secure_random_range: min and max must be numbers")
	}
	if min >= max {
		return err("secure_random_range: min must be less than max")
	}
	n, e := rand.Int(rand.Reader, big.NewInt(max-min))
	if e != nil {
		return err("secure_random_range: " + e.Error())
	}
	return &Integer{Value: min + n.Int64()}
}
