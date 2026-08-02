package analysis

import (
	"testing"
)

func analyzeSrc(t *testing.T, src string) *Analysis {
	t.Helper()
	a, errs := Analyze(src)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	return a
}

func findSymbol(t *testing.T, a *Analysis, name string) *Symbol {
	t.Helper()
	sym := a.SymbolAtName(name)
	if sym == nil {
		t.Fatalf("symbol %q not found", name)
	}
	return sym
}

func TestFunctionDefinitionAndReferences(t *testing.T) {
	a := analyzeSrc(t, `fn foo a b
    a + b

foo 1 2
`)
	foo := findSymbol(t, a, "foo")
	if foo.Kind != KindFunction {
		t.Fatalf("foo kind = %v, want Function", foo.Kind)
	}
	if foo.Pos.Line != 1 {
		t.Errorf("foo defined on line %d, want 1", foo.Pos.Line)
	}
	if len(foo.Usages) != 1 {
		t.Fatalf("foo usages = %d, want 1", len(foo.Usages))
	}
	if got := foo.Usages[0].Pos.Line; got != 4 {
		t.Errorf("foo reference on line %d, want 4", got)
	}
}

func TestParametersAndBodyReferences(t *testing.T) {
	a := analyzeSrc(t, `fn add a b
    a + b
`)
	add := findSymbol(t, a, "add")
	// a and b are parameters; the body `a + b` references them.
	var aParam, bParam *Symbol
	for _, s := range a.Symbols {
		if s.Name == "a" {
			aParam = s
		}
		if s.Name == "b" {
			bParam = s
		}
	}
	if aParam == nil || bParam == nil {
		t.Fatalf("params not found: a=%v b=%v", aParam, bParam)
	}
	if len(aParam.Usages) != 1 {
		t.Errorf("param a usages = %d, want 1", len(aParam.Usages))
	}
	if len(add.Usages) != 0 {
		t.Errorf("add should have no usages, got %d", len(add.Usages))
	}
}

func TestVariableDefinitionAndReference(t *testing.T) {
	a := analyzeSrc(t, `x: 5
print x
`)
	x := findSymbol(t, a, "x")
	if x.Kind != KindVariable {
		t.Fatalf("x kind = %v, want Variable", x.Kind)
	}
	if len(x.Usages) != 1 {
		t.Fatalf("x usages = %d, want 1", len(x.Usages))
	}
}

func TestUndefinedVariableDiagnostic(t *testing.T) {
	a := analyzeSrc(t, `print y
`)
	if len(a.Unresolved) != 1 {
		t.Fatalf("unresolved = %d, want 1 (%+v)", len(a.Unresolved), a.Unresolved)
	}
	if a.Unresolved[0].Name != "y" {
		t.Errorf("unresolved name = %q, want y", a.Unresolved[0].Name)
	}
}

func TestBuiltinReferences(t *testing.T) {
	a := analyzeSrc(t, `print upper "hi"
`)
	// builtins resolve to synthetic symbols
	for _, name := range []string{"print", "upper"} {
		if !IsBuiltin(name) {
			t.Fatalf("%s should be a builtin", name)
		}
	}
	if len(a.Unresolved) != 0 {
		t.Fatalf("unresolved = %+v, want none", a.Unresolved)
	}
}

func TestPipelineBuiltinsResolve(t *testing.T) {
	a := analyzeSrc(t, `read_file "f"
    > upper
    > print
`)
	if len(a.Unresolved) != 0 {
		t.Fatalf("unresolved = %+v, want none", a.Unresolved)
	}
}

func TestFunctionParamShadowsGlobal(t *testing.T) {
	a := analyzeSrc(t, `x: 1
fn f x
    print x
`)
	x := findSymbol(t, a, "x")
	// two symbols named x: global and param
	param := a.SymbolAtName("x")
	_ = param
	// The body reference should resolve to the parameter (innermost scope),
	// so the global symbol should have NO usages.
	if len(x.Usages) != 0 {
		t.Errorf("global x usages = %d, want 0 (shadowed)", len(x.Usages))
	}
}

func TestEnumSymbol(t *testing.T) {
	a := analyzeSrc(t, `enum Color: Red, Green, Blue
print Red
`)
	c := findSymbol(t, a, "Color")
	if c.Kind != KindEnum {
		t.Fatalf("Color kind = %v, want Enum", c.Kind)
	}
	if len(a.Unresolved) != 0 {
		t.Fatalf("unresolved = %+v, want none (enum member)", a.Unresolved)
	}
}

func TestImportModuleSymbol(t *testing.T) {
	a := analyzeSrc(t, `import "log-analyzer@1.0.0"
`)
	m := findSymbol(t, a, "log-analyzer")
	if m.Kind != KindModule {
		t.Fatalf("module kind = %v, want Module", m.Kind)
	}
}

func TestImportAlias(t *testing.T) {
	a := analyzeSrc(t, `import "log-analyzer@1.0.0" as la
`)
	findSymbol(t, a, "la")
}

func TestForInIterator(t *testing.T) {
	a := analyzeSrc(t, `for item in [1, 2, 3]
    print item
`)
	item := findSymbol(t, a, "item")
	if len(item.Usages) != 1 {
		t.Fatalf("item usages = %d, want 1", len(item.Usages))
	}
}

func TestTryCatchParam(t *testing.T) {
	a := analyzeSrc(t, `try
    read_file "x"
catch e
    print e
`)
	e := findSymbol(t, a, "e")
	if len(e.Usages) != 1 {
		t.Fatalf("catch param e usages = %d, want 1", len(e.Usages))
	}
}

func TestSymbolAtPosition(t *testing.T) {
	a := analyzeSrc(t, `fn foo a
    a

foo 5
`)
	// position of `foo` in the call (line 4, col 1)
	sym := a.SymbolAt(4, 1)
	if sym == nil || sym.Name != "foo" {
		t.Fatalf("SymbolAt(4,1) = %+v, want foo", sym)
	}
	// inside the definition (line 1, col 4)
	sym = a.SymbolAt(1, 4)
	if sym == nil || sym.Name != "foo" {
		t.Fatalf("SymbolAt(1,4) = %+v, want foo", sym)
	}
	// miss
	if a.SymbolAt(5, 1) != nil {
		t.Fatal("SymbolAt should be nil on miss")
	}
}

func TestReferencesTo(t *testing.T) {
	a := analyzeSrc(t, `fn foo
    print "hi"

foo
foo
`)
	foo := findSymbol(t, a, "foo")
	refs := a.ReferencesTo(foo)
	if len(refs) != 2 {
		t.Fatalf("references = %d, want 2", len(refs))
	}
}

func TestNoScopeForBlocks(t *testing.T) {
	a := analyzeSrc(t, `if true
    z: 1
print z
`)
	z := findSymbol(t, a, "z")
	if z == nil {
		t.Fatal("z should be declared in the enclosing scope (blocks share scope)")
	}
	if len(z.Usages) != 1 {
		t.Fatalf("z usages = %d, want 1", len(z.Usages))
	}
}

func TestParseErrorsReturned(t *testing.T) {
	_, errs := Analyze("fn foo a\n    a +\n")
	if len(errs) == 0 {
		t.Fatal("expected parse errors")
	}
}

func TestAllBuiltinsDocumented(t *testing.T) {
	docs := AllBuiltins()
	if len(docs) < 110 {
		t.Fatalf("only %d builtins documented, want >= 110", len(docs))
	}
	for _, d := range docs {
		if d.Name == "" || d.Signature == "" || d.Description == "" {
			t.Errorf("builtin %+v missing metadata", d)
		}
	}
}
