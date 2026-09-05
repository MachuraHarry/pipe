package formatter

import (
	"strings"
	"testing"
)

func TestFormatArithmetic(t *testing.T) {
	input := "1 + 2"
	got := FormatSource(input)
	if !strings.Contains(got, "1 + 2") {
		t.Errorf("got %q", got)
	}
}

func TestFormatVariable(t *testing.T) {
	input := "x:  42"
	got := FormatSource(input)
	if !strings.Contains(got, "x: 42") {
		t.Errorf("got %q", got)
	}
}

func TestFormatFn(t *testing.T) {
	input := "fn add a b\n    a + b"
	got := FormatSource(input)
	if !strings.Contains(got, "fn add") {
		t.Errorf("output should contain 'fn add', got %q", got)
	}
}

func TestFormatIf(t *testing.T) {
	input := "if true\n    42\nelse\n    10"
	got := FormatSource(input)
	if !strings.Contains(got, "if true") && !strings.Contains(got, "else") {
		t.Errorf("got %q", got)
	}
}

func TestFormatList(t *testing.T) {
	input := "[1, 2, 3]"
	got := FormatSource(input)
	if !strings.Contains(got, "[1, 2, 3]") {
		t.Errorf("got %q", got)
	}
}

func TestFormatMatchGuard(t *testing.T) {
	input := "match x\n    | _ if x > 0 -> \"positive\"\n    | _ -> \"other\""
	got := FormatSource(input)
	if !strings.Contains(got, "if x > 0") {
		t.Errorf("expected guard rendered as 'if x > 0', got %q", got)
	}
}

func TestFormatMap(t *testing.T) {
	input := "{a: 1, b: 2}"
	got := FormatSource(input)
	if !strings.Contains(got, "{a: 1, b: 2}") || !strings.Contains(got, "{b: 2, a: 1}") {
		// Map iteration is random, check that both keys appear
		if !strings.Contains(got, "a: 1") || !strings.Contains(got, "b: 2") {
			t.Errorf("got %q", got)
		}
	}
}

func TestFormatFallbackOnError(t *testing.T) {
	// This input has a trailing operator which might parse or error
	input := "  x:   42\n    print  x"
	got := FormatSource(input)
	if got == "" {
		t.Error("got empty output")
	}
	// Should at least not be empty
}

func TestFormatWhitespace(t *testing.T) {
	input := "x  :   42  \n  y  :  x  +  8"
	got := FormatSource(input)
	if !strings.Contains(got, "x: 42") && !strings.Contains(got, "x:  42") {
		t.Errorf("expected normalized whitespace, got %q", got)
	}
}

// TestFormatPreservesCommentsVerbatim covers a regression where
// formatProgram, which rebuilds source purely from the AST, silently
// deleted every "--"/"--!" comment: the AST has no representation of them
// at all (the lexer discards comment lines as it scans). Real, heavily
// documented programs lost their entire documentation on a single `pipe
// -fmt` run with no warning. Until the formatter can actually reproduce
// comments in their original position, it must leave a commented file
// completely untouched rather than destroy them.
func TestFormatPreservesCommentsVerbatim(t *testing.T) {
	input := "--! doc comment above a function\nfn add a b\n    -- comment inside the body\n    a + b\n"
	got := FormatSource(input)
	if got != input {
		t.Errorf("expected commented source to be returned unchanged, got %q, want %q", got, input)
	}
}

// TestFormatPreservesCommentsEvenWithOtherIssues checks that the safety
// bail-out fires even when the rest of the file also has sloppy
// whitespace/indentation that a normal (comment-free) input would have had
// normalized — confirming this is a real "leave everything alone" bail-out,
// not a check that only happens to pass on already-clean input.
func TestFormatPreservesCommentsEvenWithOtherIssues(t *testing.T) {
	input := "x  :   42\n-- a comment\ny:  x  +  8\n"
	got := FormatSource(input)
	if got != input {
		t.Errorf("expected untouched source, got %q, want %q", got, input)
	}
}

// TestFormatPreservesImportAlias covers a regression where the formatter
// printed "import \"x.pipe\"" and silently dropped the "as alias" clause
// (ast.ImportStatement.Alias was always populated correctly by the parser —
// formatStatement's *ast.ImportStatement case just never read it). Losing
// the alias breaks every `alias.thing` reference in the file.
func TestFormatPreservesImportAlias(t *testing.T) {
	input := "import \"modules/foo.pipe\" as f\nx: 1\n"
	got := FormatSource(input)
	if !strings.Contains(got, "import \"modules/foo.pipe\" as f") {
		t.Errorf("expected import alias to survive formatting, got %q", got)
	}
}
