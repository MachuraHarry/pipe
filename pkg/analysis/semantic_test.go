package analysis

import "testing"

func tokenTypes(toks []SemanticToken) []int {
	out := make([]int, len(toks))
	for i, t := range toks {
		out[i] = t.Type
	}
	return out
}

func containsType(toks []SemanticToken, typ int) bool {
	for _, t := range toks {
		if t.Type == typ {
			return true
		}
	}
	return false
}

func TestSemanticTokensBasics(t *testing.T) {
	src := "fn greet name\n    -- a comment\n    print name\n    print \"hi\"\n"
	a := analyzeSource(t, src)
	toks := SemanticTokens(src, a)

	if !containsType(toks, SemComment) {
		t.Error("missing comment token")
	}
	if !containsType(toks, SemKeyword) {
		t.Error("missing keyword token")
	}
	if !containsType(toks, SemFunction) {
		t.Error("missing function token for greet")
	}
	if !containsType(toks, SemParameter) {
		t.Error("missing parameter token for name")
	}
	if !containsType(toks, SemString) {
		t.Error("missing string token")
	}
}

func TestSemanticTokensStringAndNumber(t *testing.T) {
	src := "x: 42\nprint \"hello\"\n"
	a := analyzeSource(t, src)
	toks := SemanticTokens(src, a)
	if !containsType(toks, SemNumber) {
		t.Error("missing number token")
	}
	if !containsType(toks, SemString) {
		t.Error("missing string token")
	}
	if !containsType(toks, SemOperator) {
		t.Error("missing operator token for :")
	}
}

func TestSemanticTokensCommentNotInsideString(t *testing.T) {
	src := "print \"a -- b\"\n-- real comment\n"
	toks := SemanticTokens(src, nil)
	comments := 0
	for _, tk := range toks {
		if tk.Type == SemComment {
			comments++
			if tk.Line != 2 {
				t.Errorf("comment on line %d, want line 2", tk.Line)
			}
		}
	}
	if comments != 1 {
		t.Errorf("expected exactly 1 comment, got %d", comments)
	}
}

func TestSemanticTokensPositionsAreDisjoint(t *testing.T) {
	src := "fn foo x\n    if x > 3\n        print x\n"
	toks := SemanticTokens(src, nil)
	// Tokens must not overlap on the same line.
	for i := 0; i < len(toks); i++ {
		a := toks[i]
		for j := i + 1; j < len(toks); j++ {
			b := toks[j]
			if a.Line != b.Line {
				continue
			}
			if a.Col < b.Col && a.Col+a.Length > b.Col {
				t.Errorf("tokens overlap on line %d: %+v vs %+v", a.Line, a, b)
			}
			if b.Col < a.Col && b.Col+b.Length > a.Col {
				t.Errorf("tokens overlap on line %d: %+v vs %+v", a.Line, a, b)
			}
		}
	}
}

func TestSemanticTokensMultiLineString(t *testing.T) {
	src := "x: `line1\nline2`\n"
	toks := SemanticTokens(src, nil)
	strings := 0
	for _, tk := range toks {
		if tk.Type == SemString {
			strings++
			if tk.Line != 1 && tk.Line != 2 {
				t.Errorf("string token on unexpected line %d", tk.Line)
			}
		}
	}
	if strings < 2 {
		t.Errorf("expected multi-line string split into 2 tokens, got %d", strings)
	}
}

func TestSemanticTokensBuiltinIsFunction(t *testing.T) {
	src := "print 1\n"
	a := analyzeSource(t, src)
	toks := SemanticTokens(src, a)
	for _, tk := range toks {
		if tk.Line == 1 && tk.Col == 1 {
			if tk.Type != SemFunction {
				t.Errorf("builtin print type = %d, want SemFunction", tk.Type)
			}
		}
	}
}
