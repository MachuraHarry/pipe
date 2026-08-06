package object

import (
	"bytes"
	"testing"
)

func TestBytesType(t *testing.T) {
	obj := &Bytes{Value: []byte{0x48, 0x69}}
	if obj.Type() != BYTES {
		t.Errorf("Type = %s, want BYTES", obj.Type())
	}
	if obj.Inspect() != "4869" {
		t.Errorf("Inspect = %s, want 4869", obj.Inspect())
	}
}

func TestToBytesFromBytes(t *testing.T) {
	b := bToBytes(&String{Value: "Hi"})
	bb, ok := b.(*Bytes)
	if !ok {
		t.Fatalf("to_bytes returned %s, want Bytes", b.Type())
	}
	if !bytes.Equal(bb.Value, []byte{0x48, 0x69}) {
		t.Errorf("to_bytes = %v, want [72 105]", bb.Value)
	}

	s := bFromBytes(bb)
	if s.Inspect() != "Hi" {
		t.Errorf("from_bytes = %s, want Hi", s.Inspect())
	}
}

func TestToBytesList(t *testing.T) {
	b := bToBytes(&List{Elements: []Object{&Integer{Value: 1}, &Integer{Value: 255}}})
	bb, ok := b.(*Bytes)
	if !ok {
		t.Fatalf("to_bytes list = %s, want Bytes", b.Type())
	}
	if !bytes.Equal(bb.Value, []byte{1, 255}) {
		t.Errorf("to_bytes list = %v, want [1 255]", bb.Value)
	}
	if bToBytes(&List{Elements: []Object{&Integer{Value: 256}}}).Type() != ERROR {
		t.Error("to_bytes with 256 should error")
	}
}

func TestBytesAppend(t *testing.T) {
	b := bBytesAppend(&Bytes{Value: []byte{1}}, &Bytes{Value: []byte{2}}, &String{Value: "ab"})
	bb, ok := b.(*Bytes)
	if !ok {
		t.Fatalf("bytes_append = %s, want Bytes", b.Type())
	}
	want := []byte{1, 2, 'a', 'b'}
	if !bytes.Equal(bb.Value, want) {
		t.Errorf("bytes_append = %v, want %v", bb.Value, want)
	}
}

func TestIntToBytesRoundtrip(t *testing.T) {
	cases := []int64{0, 1, 255, 256, 123456789, 0x0102030405060708}
	for _, n := range cases {
		b := bIntToBytes(&Integer{Value: n})
		bb, ok := b.(*Bytes)
		if !ok {
			t.Fatalf("int_to_bytes(%d) = %s, want Bytes", n, b.Type())
		}
		got := bBytesToInt(bb)
		if got.Inspect() != itoa(n) {
			t.Errorf("roundtrip(%d) = %s", n, got.Inspect())
		}
	}
	// negative -> error
	if bIntToBytes(&Integer{Value: -1}).Type() != ERROR {
		t.Error("int_to_bytes(-1) should error")
	}
}

func TestBytesToIntOffset(t *testing.T) {
	// offset 1, n 2 of [0xAA, 0x01, 0x02, 0xBB] -> 0x0102 = 258
	b := &Bytes{Value: []byte{0xAA, 0x01, 0x02, 0xBB}}
	got := bBytesToInt(b, &Integer{Value: 1}, &Integer{Value: 2})
	if got.Inspect() != "258" {
		t.Errorf("bytes_to_int(offset=1,n=2) = %s, want 258", got.Inspect())
	}
	// n max 8
	if bBytesToInt(b, &Integer{Value: 0}, &Integer{Value: 9}).Type() != ERROR {
		t.Error("bytes_to_int n>8 should error")
	}
}

func TestBytesCompare(t *testing.T) {
	if bBytesCompare(&Bytes{Value: []byte{1}}, &Bytes{Value: []byte{2}}).Inspect() != "-1" {
		t.Error("compare(1,2) should be -1")
	}
	if bBytesCompare(&Bytes{Value: []byte{1}}, &Bytes{Value: []byte{1}}).Inspect() != "0" {
		t.Error("compare(1,1) should be 0")
	}
	if bBytesCompare(&Bytes{Value: []byte{2}}, &Bytes{Value: []byte{1}}).Inspect() != "1" {
		t.Error("compare(2,1) should be 1")
	}
}

func TestHexEncodeDecode(t *testing.T) {
	b := &Bytes{Value: []byte{0x48, 0x69}}
	if bHexEncode(b).Inspect() != "4869" {
		t.Errorf("hex_encode = %s, want 4869", bHexEncode(b).Inspect())
	}
	d := bHexDecode(&String{Value: "4869"})
	db, ok := d.(*Bytes)
	if !ok || !bytes.Equal(db.Value, []byte{0x48, 0x69}) {
		t.Errorf("hex_decode = %v, want [72 105]", d.Inspect())
	}
	if bHexDecode(&String{Value: "zz"}).Type() != ERROR {
		t.Error("hex_decode invalid should error")
	}
}

func TestSliceGeneric(t *testing.T) {
	// string
	got := bSlice(&String{Value: "hello"}, &Integer{Value: 1}, &Integer{Value: 3})
	if got.Inspect() != "el" {
		t.Errorf("slice string = %s, want el", got.Inspect())
	}
	// bytes
	b := bSlice(&Bytes{Value: []byte("hello")}, &Integer{Value: 1}, &Integer{Value: 3})
	bb, ok := b.(*Bytes)
	if !ok || !bytes.Equal(bb.Value, []byte("el")) {
		t.Errorf("slice bytes = %v, want [101 108]", b.Inspect())
	}
	// clamping
	l := bSlice(&List{Elements: []Object{&Integer{Value: 1}, &Integer{Value: 2}, &Integer{Value: 3}}},
		&Integer{Value: -5}, &Integer{Value: 99})
	ll, ok := l.(*List)
	if !ok || len(ll.Elements) != 3 {
		t.Errorf("slice clamp = %s, want full list", l.Inspect())
	}
}

func TestSubstring(t *testing.T) {
	if bSubstring(&String{Value: "hello"}, &Integer{Value: 1}, &Integer{Value: 3}).Inspect() != "el" {
		t.Error("substring(hello,1,3) should be el")
	}
	if bSubstring(&String{Value: "hello"}, &Integer{Value: 4}, &Integer{Value: 99}).Inspect() != "o" {
		t.Error("substring clamp should be o")
	}
}

func TestIndexOf(t *testing.T) {
	if bIndexOf(&String{Value: "hello world"}, &String{Value: "world"}).Inspect() != "6" {
		t.Error("index_of(hello world, world) should be 6")
	}
	if bIndexOf(&String{Value: "hello"}, &String{Value: "x"}).Inspect() != "-1" {
		t.Error("index_of missing should be -1")
	}
}

func TestBitOps(t *testing.T) {
	if bBitAnd(&Integer{Value: 6}, &Integer{Value: 3}).Inspect() != "2" {
		t.Error("bit_and(6,3) should be 2")
	}
	if bBitOr(&Integer{Value: 6}, &Integer{Value: 3}).Inspect() != "7" {
		t.Error("bit_or(6,3) should be 7")
	}
	if bBitXor(&Integer{Value: 6}, &Integer{Value: 3}).Inspect() != "5" {
		t.Error("bit_xor(6,3) should be 5")
	}
	if bBitNot(&Integer{Value: 5}).Inspect() != "-6" {
		t.Error("bit_not(5) should be -6")
	}
	if bBitLshift(&Integer{Value: 1}, &Integer{Value: 10}).Inspect() != "1024" {
		t.Error("bit_lshift(1,10) should be 1024")
	}
	if bBitRshift(&Integer{Value: 256}, &Integer{Value: 4}).Inspect() != "16" {
		t.Error("bit_rshift(256,4) should be 16")
	}
	// shift >= 64 -> error
	if bBitLshift(&Integer{Value: 1}, &Integer{Value: 64}).Type() != ERROR {
		t.Error("bit_lshift(1,64) should error")
	}
	// negative shift -> error
	if bBitLshift(&Integer{Value: 1}, &Integer{Value: -1}).Type() != ERROR {
		t.Error("bit_lshift negative should error")
	}
}

func TestCRC32(t *testing.T) {
	// CRC-32 IEEE of "hello" (known value, incl. Python zlib.crc32)
	if bCrc32(&String{Value: "hello"}).Inspect() != "907060870" {
		t.Errorf("crc32(hello) = %s, want 907060870", bCrc32(&String{Value: "hello"}).Inspect())
	}
	if bCrc32(&Bytes{Value: []byte("hello")}).Inspect() != "907060870" {
		t.Error("crc32(bytes hello) mismatch")
	}
}

func TestSortedBy(t *testing.T) {
	keyLen := &BuiltinInfo{Name: "keylen", Fn: bLen}
	strings := &List{Elements: []Object{
		&String{Value: "ccc"}, &String{Value: "a"}, &String{Value: "bb"},
	}}
	got := bSortedBy(strings, keyLen)
	l, ok := got.(*List)
	if !ok {
		t.Fatalf("sorted_by = %s, want list", got.Type())
	}
	if l.Elements[0].Inspect() != "a" || l.Elements[2].Inspect() != "ccc" {
		t.Errorf("sorted_by by len = %s", got.Inspect())
	}
}

func TestSortComparator(t *testing.T) {
	cmpDesc := &BuiltinInfo{Name: "cmpdesc", Fn: func(args ...Object) Object {
		a, _ := ToInt(args[0])
		b, _ := ToInt(args[1])
		return NativeBoolToBoolean(b < a)
	}}
	l := &List{Elements: []Object{
		&Integer{Value: 3}, &Integer{Value: 1}, &Integer{Value: 2},
	}}
	got := bSort(l, cmpDesc)
	ll, ok := got.(*List)
	if !ok {
		t.Fatalf("sort comparator = %s, want list", got.Type())
	}
	want := "321"
	if got.Inspect() != "["+stringAt(want, 0)+", "+stringAt(want, 1)+", "+stringAt(want, 2)+"]" {
		t.Errorf("sort comparator = %s", got.Inspect())
	}
	_ = ll
}

func stringAt(s string, i int) string {
	return string(s[i])
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
