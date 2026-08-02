package analysis

import "testing"

func analyzeSource(t *testing.T, src string) *Analysis {
	t.Helper()
	a, errs := Analyze(src)
	for _, e := range errs {
		t.Fatalf("unexpected parse error: %s", e)
	}
	return a
}

// analyzeLenient analyzes source even when it is mid-edit (unfinished calls),
// which is the normal situation during signature-help requests.
func analyzeLenient(src string) *Analysis {
	a, _ := Analyze(src)
	return a
}

func TestCompleteOffersBuiltinsAndSymbols(t *testing.T) {
	src := "fn greet name\n    print name\n\nprint gr"
	a := analyzeSource(t, src)
	// Cursor right after "gr": only matching items are returned.
	items := a.Complete(src, 4, 9)

	names := make(map[string]CompletionItem)
	for _, it := range items {
		names[it.Label] = it
	}

	if _, ok := names["greet"]; !ok {
		t.Errorf("completion missing user function greet; got %d items", len(items))
	}
	if _, ok := names["group_by"]; ok {
		t.Errorf("completion should not include group_by (does not match prefix gr)")
	}
	for _, it := range items {
		if len(it.Label) < 2 || it.Label[:2] != "gr" {
			t.Errorf("item %q does not match prefix gr", it.Label)
		}
	}
}

func TestCompleteFullListOnEmptyContext(t *testing.T) {
	src := "x: 1\nprint x\n"
	a := analyzeSource(t, src)
	// Cursor on the empty line after the program: no prefix -> full list.
	items := a.Complete(src, 3, 1)
	if len(items) < 100 {
		t.Fatalf("expected a large completion list, got %d", len(items))
	}
	found := false
	for _, it := range items {
		if it.Label == "x" {
			found = true
			break
		}
	}
	if !found {
		t.Error("completion missing user variable x")
	}
	if _, ok := itemByName(items, "print"); !ok {
		t.Error("completion missing builtin print")
	}
}

func itemByName(items []CompletionItem, name string) (CompletionItem, bool) {
	for _, it := range items {
		if it.Label == name {
			return it, true
		}
	}
	return CompletionItem{}, false
}

func TestCompleteWordRange(t *testing.T) {
	src := "print hello\n"
	word, col := CompletionWord(src, 1, 12)
	if word != "hello" || col != 7 {
		t.Errorf("word=%q col=%d, want hello col=7", word, col)
	}
}

func TestHoverOnBuiltin(t *testing.T) {
	src := "print \"hi\"\n"
	a := analyzeSource(t, src)
	info, ok := a.Hover(src, 1, 3)
	if !ok {
		t.Fatal("expected hover on print")
	}
	if info.Contents[0] == "" {
		t.Error("hover contents empty")
	}
}

func TestHoverOnUserFunction(t *testing.T) {
	src := "fn greet name\n    print name\n\ngreet \"x\"\n"
	a := analyzeSource(t, src)
	info, ok := a.Hover(src, 4, 1)
	if !ok {
		t.Fatal("expected hover on greet reference")
	}
	if info.Start.Line != 4 || info.Start.Col != 1 {
		t.Errorf("hover range = %d:%d, want 4:1", info.Start.Line, info.Start.Col)
	}
}

func TestHoverOnKeyword(t *testing.T) {
	src := "if true\n    print \"x\"\n"
	a := analyzeSource(t, src)
	_, ok := a.Hover(src, 1, 1)
	if !ok {
		t.Fatal("expected hover on if keyword")
	}
}

func TestSignatureHelpForBuiltin(t *testing.T) {
	src := "print join(\n"
	a := analyzeLenient(src)
	sig, ok := a.SignatureHelp(src, 1, 12)
	if !ok {
		t.Fatal("expected signature help for join")
	}
	if sig.Label != "join(list, delimiter)" {
		t.Errorf("label = %q, want join(list, delimiter)", sig.Label)
	}
	if sig.ActiveParam != 0 {
		t.Errorf("active param = %d, want 0", sig.ActiveParam)
	}
}

func TestSignatureHelpActiveParamAfterComma(t *testing.T) {
	src := "print join(xs, \n"
	a := analyzeLenient(src)
	sig, ok := a.SignatureHelp(src, 1, 17)
	if !ok {
		t.Fatal("expected signature help")
	}
	if sig.ActiveParam != 1 {
		t.Errorf("active param = %d, want 1", sig.ActiveParam)
	}
}

func TestSignatureHelpForUserFunction(t *testing.T) {
	src := "fn add a b\n    a + b\n\nadd(\n"
	a := analyzeLenient(src)
	sig, ok := a.SignatureHelp(src, 4, 6)
	if !ok {
		t.Fatal("expected signature help for user fn add")
	}
	if sig.Label != "add(…)" {
		t.Errorf("label = %q, want add(…)", sig.Label)
	}
}

func TestSignatureHelpNotForGroupingParens(t *testing.T) {
	src := "(1 + 2) * 3\n"
	a := analyzeSource(t, src)
	if _, ok := a.SignatureHelp(src, 1, 3); ok {
		t.Fatal("did not expect signature help for grouping parens")
	}
}

func TestDefinitionAtReference(t *testing.T) {
	src := "fn greet name\n    print name\n\ngreet \"x\"\n"
	a := analyzeSource(t, src)
	loc, ok := a.DefinitionAt(4, 1)
	if !ok {
		t.Fatal("expected definition for greet reference")
	}
	if loc.Start.Line != 1 || loc.Start.Col != 4 {
		t.Errorf("definition start = %d:%d, want 1:4", loc.Start.Line, loc.Start.Col)
	}
}

func TestDefinitionAtParameter(t *testing.T) {
	src := "fn greet name\n    print name\n"
	a := analyzeSource(t, src)
	loc, ok := a.DefinitionAt(2, 11)
	if !ok {
		t.Fatal("expected definition for parameter usage")
	}
	if loc.Start.Line != 1 || loc.Start.Col != 10 {
		t.Errorf("definition start = %d:%d, want 1:10", loc.Start.Line, loc.Start.Col)
	}
}

func TestDefinitionAtBuiltinFails(t *testing.T) {
	src := "print \"hi\"\n"
	a := analyzeSource(t, src)
	if _, ok := a.DefinitionAt(1, 1); ok {
		t.Fatal("builtins must not have a source definition")
	}
}

func TestReferencesCollectDefinitionAndUsages(t *testing.T) {
	src := "fn greet name\n    print name\n    greet name\ngreet \"x\"\n"
	a := analyzeSource(t, src)
	// Position on the greet reference in the body.
	refs := a.ReferencesAt(3, 5)
	if len(refs) != 3 {
		t.Fatalf("expected 3 references, got %d: %+v", len(refs), refs)
	}
	// Definition is the first entry.
	if refs[0].Start.Line != 1 || refs[0].Start.Col != 4 {
		t.Errorf("first reference should be the definition at 1:4, got %d:%d", refs[0].Start.Line, refs[0].Start.Col)
	}
}

func TestValidateIdentifier(t *testing.T) {
	if err := ValidateIdentifier("myVar2"); err != nil {
		t.Errorf("myVar2 should be valid, got %v", err)
	}
	if err := ValidateIdentifier("fn"); err == nil {
		t.Error("keyword fn must be rejected")
	}
	if err := ValidateIdentifier(""); err == nil {
		t.Error("empty name must be rejected")
	}
	if err := ValidateIdentifier("2fast"); err == nil {
		t.Error("leading digit must be rejected")
	}
}

func TestRenameReturnsLocations(t *testing.T) {
	src := "fn greet name\n    print name\n    greet name\n"
	a := analyzeSource(t, src)
	locs, err := a.RenameAt(3, 5, "hello")
	if err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	if len(locs) != 2 {
		t.Fatalf("expected 2 locations (definition + usage), got %d", len(locs))
	}
	if _, err := a.RenameAt(3, 5, "if"); err == nil {
		t.Error("renaming to a keyword must fail")
	}
}
