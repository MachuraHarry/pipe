package object

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func i(n int64) Object { return &Integer{Value: n} }

func TestFileOpenCloseReadWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "io.bin")

	h := bFileOpen(&String{Value: path}, &String{Value: "rw"})
	hi, ok := h.(*Integer)
	if !ok {
		t.Fatalf("file_open = %s, want handle number", h.Type())
	}
	handle := i(hi.Value)

	if bFileWrite(handle, i(0), &Bytes{Value: []byte("0123456789")}).Inspect() != "10" {
		t.Error("file_write returned wrong count")
	}
	if bFileWrite(handle, i(5), &Bytes{Value: []byte("ABCDE")}).Inspect() != "5" {
		t.Error("file_write overlap returned wrong count")
	}
	if bFileTruncate(handle, i(8)).Type() != NIL {
		t.Error("file_truncate should return nil")
	}
	if bFileSync(handle).Type() != NIL {
		t.Error("file_sync should return nil")
	}

	r := bFileRead(handle, i(0), i(8))
	rb, ok := r.(*Bytes)
	if !ok || !bytes.Equal(rb.Value, []byte("01234ABC")) {
		t.Errorf("file_read = %v, want [48 49 50 51 52 65 66 67]", r.Inspect())
	}

	// reading past EOF returns fewer bytes
	r2 := bFileRead(handle, i(6), i(100))
	rb2, ok := r2.(*Bytes)
	if !ok || len(rb2.Value) != 2 {
		t.Errorf("file_read past EOF = %s, want 2 bytes", r2.Inspect())
	}

	if bFileClose(handle).Type() != NIL {
		t.Error("file_close should return nil")
	}
	if bFileClose(handle).Type() != ERROR {
		t.Error("file_close on closed handle should error")
	}
	if bFileRead(handle, i(0), i(1)).Type() != ERROR {
		t.Error("file_read on closed handle should error")
	}
}

func TestFileOpenModes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.bin")

	if bFileOpen(&String{Value: path}, &String{Value: "bogus"}).Type() != ERROR {
		t.Error("file_open bogus mode should error")
	}
	// missing file with mode "r" should error
	if bFileOpen(&String{Value: filepath.Join(dir, "nope.bin")}, &String{Value: "r"}).Type() != ERROR {
		t.Error("file_open r on missing file should error")
	}
	h := bFileOpen(&String{Value: path}, &String{Value: "w"})
	if h.Type() == ERROR {
		t.Fatalf("file_open w failed: %s", h.Inspect())
	}
	bFileClose(i(h.(*Integer).Value))
	// read-only handle should reject writes
	h2 := bFileOpen(&String{Value: path}, &String{Value: "r"})
	hi, _ := h2.(*Integer)
	if hi == nil {
		t.Fatalf("file_open r failed: %s", h2.Inspect())
	}
	if bFileWrite(i(hi.Value), i(0), &Bytes{Value: []byte("x")}).Type() != ERROR {
		t.Error("file_write on read-only handle should error")
	}
	bFileClose(i(hi.Value))
}

func TestFileOpenSandbox(t *testing.T) {
	if os.Getenv("PIPE_TEST_SANDBOX") == "" {
		t.Skip("sandbox profile test opt-in")
	}
}
