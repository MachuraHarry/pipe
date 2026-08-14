package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/object"
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
