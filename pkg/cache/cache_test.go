package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/object"
)

func TestWriteLoadRoundTrip(t *testing.T) {
	ins := compiler.Make(compiler.OpConstant, 0)
	ins = append(ins, compiler.Make(compiler.OpConstant, 1)...)
	ins = append(ins, compiler.Make(compiler.OpHalt)...)
	bc := &compiler.Bytecode{
		Instructions: compiler.Instructions(ins),
		Constants: []object.Object{
			&object.Integer{Value: 42},
			&object.String{Value: "hello"},
			&object.CompiledFunction{
				Instructions: compiler.Instructions([]byte{1, 2, 3, 4}),
				Lines:        []int{1, 2, 3},
				NumLocals:    2,
			},
		},
	}
	deps := "0123456789abcdef0123456789abcdef"
	path := filepath.Join(t.TempDir(), "test.pipec")

	if err := writeCache(path, deps, bc); err != nil {
		t.Fatalf("writeCache: %s", err)
	}

	loaded, err := loadCache(path, deps)
	if err != nil {
		t.Fatalf("loadCache: %s", err)
	}
	if len(loaded.Instructions) != len(bc.Instructions) {
		t.Errorf("instructions: got %d want %d", len(loaded.Instructions), len(bc.Instructions))
	}
	if len(loaded.Constants) != 3 {
		t.Fatalf("constants: got %d want 3", len(loaded.Constants))
	}
	if v, ok := loaded.Constants[0].(*object.Integer); !ok || v.Value != 42 {
		t.Errorf("constant[0]: got %#v", loaded.Constants[0])
	}
	if s, ok := loaded.Constants[1].(*object.String); !ok || s.Value != "hello" {
		t.Errorf("constant[1]: got %#v", loaded.Constants[1])
	}
	cf, ok := loaded.Constants[2].(*object.CompiledFunction)
	if !ok || cf.NumLocals != 2 || len(cf.Lines) != 3 || len(cf.Instructions.(compiler.Instructions)) != 4 {
		t.Errorf("constant[2]: got %#v", loaded.Constants[2])
	}

	if _, err := loadCache(path, "fedcba9876543210fedcba9876543210"); err == nil {
		t.Error("loadCache with a different deps hash must fail")
	}
}

func TestLoadOrCompileCachesOnSecondCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.pipe")
	if err := os.WriteFile(path, []byte("42\n\"hello\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	bc1, cached, err := LoadOrCompile(path)
	if err != nil {
		t.Fatalf("LoadOrCompile: %s", err)
	}
	if cached {
		t.Error("first call should not be cached")
	}
	if bc1 == nil || len(bc1.Instructions) == 0 {
		t.Fatal("empty bytecode")
	}

	bc2, cached, err := LoadOrCompile(path)
	if err != nil {
		t.Fatalf("LoadOrCompile: %s", err)
	}
	if !cached {
		t.Error("second call should hit the cache")
	}
	if len(bc2.Instructions) != len(bc1.Instructions) {
		t.Error("cached bytecode differs from freshly compiled")
	}
}

func TestLoadOrCompileInvalidatesOnSourceChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.pipe")
	if err := os.WriteFile(path, []byte("42\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadOrCompile(path); err != nil {
		t.Fatal(err)
	}
	if _, cached, _ := LoadOrCompile(path); !cached {
		t.Fatal("expected cache hit before modification")
	}

	if err := os.WriteFile(path, []byte("99\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, cached, err := LoadOrCompile(path); err != nil {
		t.Fatal(err)
	} else if cached {
		t.Error("source change should invalidate the cache")
	}
}

func TestLoadOrCompileInvalidatesOnDependencyChange(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib.pipe")
	main := filepath.Join(dir, "main.pipe")
	if err := os.WriteFile(lib, []byte("fn fortytwo\n    42\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(main, []byte("import \"./lib.pipe\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := LoadOrCompile(main); err != nil {
		t.Fatalf("first compile: %s", err)
	}
	if _, cached, _ := LoadOrCompile(main); !cached {
		t.Fatal("expected cache hit")
	}

	// The top-level file is unchanged, but the compiler bakes the import into
	// the bytecode, so a dependency change must invalidate the cache.
	if err := os.WriteFile(lib, []byte("fn fortytwo\n    43\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, cached, err := LoadOrCompile(main); err != nil {
		t.Fatalf("after dependency change: %s", err)
	} else if cached {
		t.Error("dependency change should invalidate the cache")
	}
}
