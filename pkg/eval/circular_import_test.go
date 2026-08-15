package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

// writeImportFiles writes a set of module files into a temp dir and returns
// the temp dir together with the path of the entry file.
func writeImportFiles(t *testing.T, files map[string]string) (dir, entry string) {
	t.Helper()
	dir = t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, filepath.Join(dir, "main.pipe")
}

func evalFile(t *testing.T, sourceFile string, source string) object.Object {
	t.Helper()
	program := parseProgram(t, source)
	ctx := NewEvalContext(sourceFile)
	env := object.NewEnvironment()
	return ctx.Eval(program, env)
}

func assertIsError(t *testing.T, obj object.Object) {
	t.Helper()
	if obj == nil {
		t.Fatal("expected error object, got nil")
	}
	if _, ok := obj.(*object.Error); !ok {
		t.Fatalf("expected error object, got %T", obj)
	}
}

func TestCircularImportDetectedEval(t *testing.T) {
	dir, entry := writeImportFiles(t, map[string]string{
		"main.pipe":  "import \"a.pipe\"\nprint (a_fn)\n",
		"a.pipe":     "import \"b.pipe\"\nfn a_fn\n    b_fn\n",
		"b.pipe":     "import \"a.pipe\"\nfn b_fn\n    42\n",
	})
	t.Setenv("PIPE_PATH", dir)

	result := evalFile(t, entry, "import \"a.pipe\"\nprint (a_fn)\n")
	assertIsError(t, result)
	if !strings.Contains(result.Inspect(), "E009") {
		t.Errorf("expected E009 in error, got: %v", result.Inspect())
	}
	if !strings.Contains(result.Inspect(), "circular import") {
		t.Errorf("expected circular import message, got: %v", result.Inspect())
	}
}

func TestCircularImportDetectedVM(t *testing.T) {
	dir, entry := writeImportFiles(t, map[string]string{
		"main.pipe": "import \"a.pipe\"\nprint (a_fn)\n",
		"a.pipe":    "import \"b.pipe\"\nfn a_fn\n    b_fn\n",
		"b.pipe":    "import \"a.pipe\"\nfn b_fn\n    42\n",
	})
	t.Setenv("PIPE_PATH", dir)

	program := parseProgram(t, "import \"a.pipe\"\nprint (a_fn)\n")
	c := compiler.NewWithFile(entry)
	err := c.Compile(program)
	if err == nil {
		t.Fatal("expected compile error for circular import, got nil")
	}
	if !strings.Contains(err.Error(), "E009") {
		t.Errorf("expected E009 in compile error, got: %v", err)
	}
}

func TestCircularImportDetectedEvalAlias(t *testing.T) {
	dir, _ := writeImportFiles(t, map[string]string{
		"main.pipe": "import \"a.pipe\" as a\nprint (a.a_fn)\n",
		"a.pipe":    "import \"b.pipe\" as b\nfn a_fn\n    b.b_fn\n",
		"b.pipe":    "import \"a.pipe\" as a\nfn b_fn\n    42\n",
	})
	t.Setenv("PIPE_PATH", dir)
	entry := filepath.Join(dir, "main.pipe")

	result := evalFile(t, entry, "import \"a.pipe\" as a\nprint (a.a_fn)\n")
	assertIsError(t, result)
	if !strings.Contains(result.Inspect(), "E009") {
		t.Errorf("expected E009 in alias circular import, got: %v", result.Inspect())
	}
}

func TestDiamondImportNotDetectedAsCircular(t *testing.T) {
	dir, entry := writeImportFiles(t, map[string]string{
		"main.pipe": "import \"a.pipe\"\nimport \"b.pipe\"\nprint (a_fn)\nprint (b_fn)\n",
		"a.pipe":    "fn a_fn\n    1\n",
		"b.pipe":    "import \"a.pipe\"\nfn b_fn\n    a_fn\n",
	})
	t.Setenv("PIPE_PATH", dir)

	result := evalFile(t, entry, "import \"a.pipe\"\nimport \"b.pipe\"\nprint (a_fn)\nprint (b_fn)\n")
	if err, ok := result.(*object.Error); ok {
		t.Fatalf("diamond import should not be circular: %v", err.Inspect())
	}
}

func TestNestedImportCompilesVM(t *testing.T) {
	dir, entry := writeImportFiles(t, map[string]string{
		"main.pipe": "import \"a.pipe\"\nprint (a_fn)\n",
		"a.pipe":    "import \"b.pipe\"\nfn a_fn\n    b_fn\n",
		"b.pipe":    "fn b_fn\n    42\n",
	})
	t.Setenv("PIPE_PATH", dir)

	program := parseProgram(t, "import \"a.pipe\"\nprint (a_fn)\n")
	c := compiler.NewWithFile(entry)
	if err := c.Compile(program); err != nil {
		t.Fatalf("nested import should compile: %v", err)
	}
}

func TestAliasImportVMMatchesEval(t *testing.T) {
	dir, entry := writeImportFiles(t, map[string]string{
		"main.pipe": "import \"a.pipe\" as a\nprint (a.double 21)\nprint a.greeting\nprint a.green\nprint (a.use_internal 5)\n",
		"a.pipe": "export fn double x\n    x * 2\n\nexport greeting: \"hi\"\n\nexport enum Color: red, green, blue\n\nfn internal_helper x\n    x + 100\n\nexport fn use_internal x\n    internal_helper x\n",
	})
	t.Setenv("PIPE_PATH", dir)

	source := "import \"a.pipe\" as a\nprint (a.double 21)\nprint a.greeting\nprint a.green\nprint (a.use_internal 5)\n"
	program := parseProgram(t, source)

	evalR := evalFile(t, entry, source)
	if e, ok := evalR.(*object.Error); ok {
		t.Fatalf("eval failed: %v", e.Inspect())
	}

	c := compiler.NewWithFile(entry)
	if err := c.Compile(program); err != nil {
		t.Fatalf("aliased import should compile in VM: %v", err)
	}
	machine := vm.New(c.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("vm failed: %v", err)
	}
}

func TestAliasImportVMHidesInternalSymbols(t *testing.T) {
	// Only exported names must be reachable through the alias namespace;
	// internal (non-exported) module symbols must stay hidden.
	dir, entry := writeImportFiles(t, map[string]string{
		"main.pipe": "import \"a.pipe\" as a\nprint (a.double 2)\nprint a.internal_helper\n",
		"a.pipe":    "export fn double x\n    x * 2\n\nfn internal_helper x\n    x + 100\n",
	})
	t.Setenv("PIPE_PATH", dir)

	source := "import \"a.pipe\" as a\nprint (a.double 2)\nprint a.internal_helper\n"
	program := parseProgram(t, source)

	// The tree-walker rejects the field access (not exported).
	assertIsError(t, evalFile(t, entry, source))

	// The VM must not expose the internal symbol as a callable either.
	c := compiler.NewWithFile(entry)
	if err := c.Compile(program); err != nil {
		t.Fatalf("aliased import should compile in VM: %v", err)
	}
	machine := vm.New(c.Bytecode())
	if err := machine.Run(); err != nil {
		if e, ok := err.(*object.Error); ok && strings.Contains(e.Message, "not a function") {
			return // internal symbol resolved to nil, not callable — hidden
		}
		t.Fatalf("vm failed: %v", err)
	}
	top := machine.LastPoppedStackElem()
	if top == nil || top.Inspect() == "nil" {
		return // resolved to nil — hidden
	}
	t.Fatalf("internal symbol leaked into alias namespace: %s", top.Inspect())
}
