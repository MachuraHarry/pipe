package cache

import (
	"os"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/object"
)

func TestWriteAndLoadCache(t *testing.T) {
	ins := compiler.Make(compiler.OpConstant, 0)
	ins = append(ins, compiler.Make(compiler.OpHalt)...)
	bc := &compiler.Bytecode{
		Instructions: compiler.Instructions(ins),
		Constants: []object.Object{
			&object.Integer{Value: 42},
			&object.String{Value: "hello"},
		},
	}

	path := "/tmp/test.pipec"
	defer os.Remove(path)

	if err := WriteCache(path, bc); err != nil {
		t.Fatalf("WriteCache: %s", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("cache file not created: %s", err)
	}
	if info.Size() == 0 {
		t.Fatal("cache file is empty")
	}
}

func TestLoadOrCompile(t *testing.T) {
	source := "42\n\"hello\"\n"
	path := "/tmp/test_load.pipe"
	cachePath := path + "c"
	defer os.Remove(path)
	defer os.Remove(cachePath)

	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	bc, cached, err := LoadOrCompile(path)
	if err != nil {
		t.Fatalf("LoadOrCompile: %s", err)
	}
	if cached {
		t.Error("first call should not be cached")
	}
	if bc == nil {
		t.Fatal("got nil bytecode")
	}
	if len(bc.Instructions) == 0 {
		t.Error("empty instructions")
	}
}

func TestLoadOrCompileModifiedSource(t *testing.T) {
	source := "42\n"
	path := "/tmp/test_mod.pipe"
	cachePath := path + "c"
	defer os.Remove(path)
	defer os.Remove(cachePath)

	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	bc, _, err := LoadOrCompile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bc == nil {
		t.Fatal("nil bytecode")
	}

	modified := "99\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatal(err)
	}

	bc2, _, err := LoadOrCompile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bc2 == nil {
		t.Fatal("nil bytecode after modification")
	}
}
