package object

import (
	"bytes"
	"encoding/hex"
	"hash/crc32"
	"sort"
	"strings"
)

func init() {
	Builtins = append(Builtins,
		BuiltinInfo{Name: "to_bytes", Fn: bToBytes},
		BuiltinInfo{Name: "from_bytes", Fn: bFromBytes},
		BuiltinInfo{Name: "bytes_append", Fn: bBytesAppend},
		BuiltinInfo{Name: "bytes_to_int", Fn: bBytesToInt},
		BuiltinInfo{Name: "int_to_bytes", Fn: bIntToBytes},
		BuiltinInfo{Name: "bytes_compare", Fn: bBytesCompare},
		BuiltinInfo{Name: "hex_encode", Fn: bHexEncode},
		BuiltinInfo{Name: "hex_decode", Fn: bHexDecode},
		BuiltinInfo{Name: "slice", Fn: bSlice},
		BuiltinInfo{Name: "substring", Fn: bSubstring},
		BuiltinInfo{Name: "index_of", Fn: bIndexOf},
		BuiltinInfo{Name: "bit_and", Fn: bBitAnd},
		BuiltinInfo{Name: "bit_or", Fn: bBitOr},
		BuiltinInfo{Name: "bit_xor", Fn: bBitXor},
		BuiltinInfo{Name: "bit_not", Fn: bBitNot},
		BuiltinInfo{Name: "bit_lshift", Fn: bBitLshift},
		BuiltinInfo{Name: "bit_rshift", Fn: bBitRshift},
		BuiltinInfo{Name: "crc32", Fn: bCrc32},
		BuiltinInfo{Name: "sorted_by", Fn: bSortedBy},
	)
}

// ---- Bytes conversion ----

func bToBytes(args ...Object) Object {
	if len(args) != 1 {
		return err("to_bytes expects 1 argument")
	}
	switch v := args[0].(type) {
	case *String:
		return &Bytes{Value: []byte(v.Value)}
	case *Bytes:
		return v
	case *List:
		out := make([]byte, 0, len(v.Elements))
		for _, e := range v.Elements {
			n, ok := ToInt(e)
			if !ok || n < 0 || n > 255 {
				return err("to_bytes: list must contain integers 0-255")
			}
			out = append(out, byte(n))
		}
		return &Bytes{Value: out}
	}
	return err("to_bytes expects string, bytes, or list of ints")
}

func bFromBytes(args ...Object) Object {
	if len(args) != 1 {
		return err("from_bytes expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Bytes:
		return &String{Value: string(v.Value)}
	case *String:
		return v
	}
	return err("from_bytes expects bytes or string")
}

func bBytesAppend(args ...Object) Object {
	if len(args) < 2 {
		return err("bytes_append expects at least 2 arguments")
	}
	var out []byte
	for _, a := range args {
		switch v := a.(type) {
		case *Bytes:
			out = append(out, v.Value...)
		case *String:
			out = append(out, v.Value...)
		default:
			return err("bytes_append: arguments must be bytes or strings")
		}
	}
	return &Bytes{Value: out}
}

func bBytesToInt(args ...Object) Object {
	if len(args) < 1 || len(args) > 3 {
		return err("bytes_to_int expects 1-3 arguments (bytes[, offset[, n]])")
	}
	b, ok := args[0].(*Bytes)
	if !ok {
		return err("bytes_to_int: first argument must be bytes")
	}
	off := int64(0)
	n := int64(len(b.Value))
	if len(args) >= 2 {
		var ok2 bool
		off, ok2 = ToInt(args[1])
		if !ok2 {
			return err("bytes_to_int: offset must be a number")
		}
		n = int64(len(b.Value)) - off
	}
	if len(args) == 3 {
		var ok3 bool
		n, ok3 = ToInt(args[2])
		if !ok3 {
			return err("bytes_to_int: n must be a number")
		}
	}
	if off < 0 || n < 0 || off+n > int64(len(b.Value)) {
		return err("bytes_to_int: out of range")
	}
	if n > 8 {
		n = 8
	}
	var val uint64
	for i := int64(0); i < n; i++ {
		val = val<<8 | uint64(b.Value[off+i])
	}
	return &Integer{Value: int64(val)}
}

func bIntToBytes(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return err("int_to_bytes expects 1-2 arguments (value[, n])")
	}
	val, ok := ToInt(args[0])
	if !ok {
		return err("int_to_bytes: value must be a number")
	}
	if val < 0 {
		return err("int_to_bytes: value must be non-negative")
	}
	u := uint64(val)
	n := int64(0)
	if len(args) == 2 {
		var ok2 bool
		n, ok2 = ToInt(args[1])
		if !ok2 {
			return err("int_to_bytes: n must be a number")
		}
		if n < 0 || n > 8 {
			return err("int_to_bytes: n must be 0-8")
		}
	} else {
		if u == 0 {
			n = 1
		} else {
			for u > 0 {
				n++
				u >>= 8
			}
		}
	}
	out := make([]byte, n)
	u = uint64(val)
	for i := n - 1; i >= 0; i-- {
		out[i] = byte(u)
		u >>= 8
	}
	return &Bytes{Value: out}
}

func bBytesCompare(args ...Object) Object {
	if len(args) != 2 {
		return err("bytes_compare expects 2 arguments")
	}
	var a, b []byte
	for i, x := range args {
		switch v := x.(type) {
		case *Bytes:
			if i == 0 {
				a = v.Value
			} else {
				b = v.Value
			}
		case *String:
			if i == 0 {
				a = []byte(v.Value)
			} else {
				b = []byte(v.Value)
			}
		default:
			return err("bytes_compare: arguments must be bytes or strings")
		}
	}
	return &Integer{Value: int64(bytes.Compare(a, b))}
}

// ---- Hex ----

func bHexEncode(args ...Object) Object {
	if len(args) != 1 {
		return err("hex_encode expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Bytes:
		return &String{Value: hex.EncodeToString(v.Value)}
	case *String:
		return &String{Value: hex.EncodeToString([]byte(v.Value))}
	}
	return err("hex_encode expects bytes or string")
}

func bHexDecode(args ...Object) Object {
	if len(args) != 1 {
		return err("hex_decode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("hex_decode expects a string")
	}
	dec, e := hex.DecodeString(s.Value)
	if e != nil {
		return err("hex_decode: " + e.Error())
	}
	return &Bytes{Value: dec}
}

// ---- Slicing (list / string / bytes) ----

func clampSlice(start, end int64, length int) (int, int) {
	if start < 0 {
		start = 0
	}
	if start > int64(length) {
		start = int64(length)
	}
	if end < 0 {
		end = 0
	}
	if end > int64(length) {
		end = int64(length)
	}
	if start > end {
		start = end
	}
	return int(start), int(end)
}

func bSlice(args ...Object) Object {
	if len(args) != 3 {
		return err("slice expects 3 arguments (value, start, end)")
	}
	start, ok := ToInt(args[1])
	if !ok {
		return err("slice: start must be a number")
	}
	end, ok2 := ToInt(args[2])
	if !ok2 {
		return err("slice: end must be a number")
	}
	switch v := args[0].(type) {
	case *List:
		s, e := clampSlice(start, end, len(v.Elements))
		out := make([]Object, e-s)
		copy(out, v.Elements[s:e])
		return &List{Elements: out}
	case *String:
		s, e := clampSlice(start, end, len(v.Value))
		return &String{Value: v.Value[s:e]}
	case *Bytes:
		s, e := clampSlice(start, end, len(v.Value))
		out := make([]byte, e-s)
		copy(out, v.Value[s:e])
		return &Bytes{Value: out}
	}
	return err("slice expects list, string, or bytes")
}

func bSubstring(args ...Object) Object {
	if len(args) != 3 {
		return err("substring expects 3 arguments (string, start, end)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("substring: first argument must be a string")
	}
	start, ok := ToInt(args[1])
	if !ok {
		return err("substring: start must be a number")
	}
	end, ok2 := ToInt(args[2])
	if !ok2 {
		return err("substring: end must be a number")
	}
	a, b := clampSlice(start, end, len(s.Value))
	return &String{Value: s.Value[a:b]}
}

func bIndexOf(args ...Object) Object {
	if len(args) != 2 {
		return err("index_of expects 2 arguments (string/list, needle)")
	}
	switch haystack := args[0].(type) {
	case *String:
		needle, ok := args[1].(*String)
		if !ok {
			return err("index_of: needle must be a string when haystack is a string")
		}
		return &Integer{Value: int64(strings.Index(haystack.Value, needle.Value))}
	case *List:
		if args[1] == nil {
			return &Integer{Value: -1}
		}
		for i, elem := range haystack.Elements {
			if elem == args[1] || (elem.Type() == args[1].Type() && elem.Inspect() == args[1].Inspect()) {
				return &Integer{Value: int64(i)}
			}
		}
		return &Integer{Value: -1}
	}
	return err("index_of: first argument must be a string or list")
}

// ---- Bitwise (int64 bit patterns) ----

func bitArgs(a, b Object) (int64, int64, bool) {
	x, ok := ToInt(a)
	if !ok {
		return 0, 0, false
	}
	y, ok2 := ToInt(b)
	if !ok2 {
		return 0, 0, false
	}
	return x, y, true
}

func bBitAnd(args ...Object) Object {
	if len(args) != 2 {
		return err("bit_and expects 2 arguments")
	}
	x, y, ok := bitArgs(args[0], args[1])
	if !ok {
		return err("bit_and: arguments must be numbers")
	}
	return &Integer{Value: x & y}
}

func bBitOr(args ...Object) Object {
	if len(args) != 2 {
		return err("bit_or expects 2 arguments")
	}
	x, y, ok := bitArgs(args[0], args[1])
	if !ok {
		return err("bit_or: arguments must be numbers")
	}
	return &Integer{Value: x | y}
}

func bBitXor(args ...Object) Object {
	if len(args) != 2 {
		return err("bit_xor expects 2 arguments")
	}
	x, y, ok := bitArgs(args[0], args[1])
	if !ok {
		return err("bit_xor: arguments must be numbers")
	}
	return &Integer{Value: x ^ y}
}

func bBitNot(args ...Object) Object {
	if len(args) != 1 {
		return err("bit_not expects 1 argument")
	}
	x, ok := ToInt(args[0])
	if !ok {
		return err("bit_not: argument must be a number")
	}
	return &Integer{Value: ^x}
}

func bBitLshift(args ...Object) Object {
	if len(args) != 2 {
		return err("bit_lshift expects 2 arguments")
	}
	x, y, ok := bitArgs(args[0], args[1])
	if !ok {
		return err("bit_lshift: arguments must be numbers")
	}
	if y < 0 || y >= 64 {
		return err("bit_lshift: shift must be 0-63")
	}
	return &Integer{Value: int64(uint64(x) << uint64(y))}
}

func bBitRshift(args ...Object) Object {
	if len(args) != 2 {
		return err("bit_rshift expects 2 arguments")
	}
	x, y, ok := bitArgs(args[0], args[1])
	if !ok {
		return err("bit_rshift: arguments must be numbers")
	}
	if y < 0 || y >= 64 {
		return err("bit_rshift: shift must be 0-63")
	}
	return &Integer{Value: int64(uint64(x) >> uint64(y))}
}

// ---- Checksums ----

func bCrc32(args ...Object) Object {
	if len(args) != 1 {
		return err("crc32 expects 1 argument")
	}
	var data []byte
	switch v := args[0].(type) {
	case *Bytes:
		data = v.Value
	case *String:
		data = []byte(v.Value)
	default:
		return err("crc32 expects bytes or string")
	}
	return &Integer{Value: int64(crc32.ChecksumIEEE(data))}
}

// ---- Key-based sorting ----

func compareObjects(a, b Object) int {
	if af, okA := ToFloat(a); okA {
		if bf, okB := ToFloat(b); okB {
			switch {
			case af < bf:
				return -1
			case af > bf:
				return 1
			default:
				return 0
			}
		}
	}
	as, bs := a.Inspect(), b.Inspect()
	switch {
	case as < bs:
		return -1
	case as > bs:
		return 1
	}
	return 0
}

func bSortedBy(args ...Object) Object {
	if len(args) != 2 {
		return err("sorted_by expects 2 arguments (list, keyFn)")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("sorted_by expects list")
	}
	keyFn := args[1]
	keys := make([]Object, len(l.Elements))
	for i, e := range l.Elements {
		keys[i] = callOne(keyFn, e)
	}
	idx := make([]int, len(l.Elements))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		return compareObjects(keys[idx[i]], keys[idx[j]]) < 0
	})
	out := make([]Object, len(l.Elements))
	for i, k := range idx {
		out[i] = l.Elements[k]
	}
	return &List{Elements: out}
}
