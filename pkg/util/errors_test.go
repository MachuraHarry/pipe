package util

import "testing"

func TestSnippet(t *testing.T) {
	src := "a: 1\nx ++ 42\nprint x\n"
	got := Snippet(src, 2, 3)
	want := "2 | x ++ 42\n" + "    " + "  ^"
	if got != want {
		t.Errorf("Snippet = %q, want %q", got, want)
	}
}

func TestSnippetOutOfRange(t *testing.T) {
	if got := Snippet("one\n", 5, 1); got != "" {
		t.Errorf("out-of-range line should return empty, got %q", got)
	}
}

func TestSnippetClampsColumn(t *testing.T) {
	if got := Snippet("hi\n", 1, 0); got == "" {
		t.Error("col 0 should clamp to 1, not be empty")
	}
}

func TestParseParserError(t *testing.T) {
	line, col, msg, ok := ParseParserError("line 3 col 5: boom")
	if !ok || line != 3 || col != 5 || msg != "boom" {
		t.Errorf("got %d/%d/%q/%v", line, col, msg, ok)
	}
	if _, _, _, ok := ParseParserError("not a parser error"); ok {
		t.Error("non-parser string should not parse")
	}
}

func TestFormatErrorWithSnippet(t *testing.T) {
	out := FormatErrorWithSnippet("a: 1\nx +\n", "line 2 col 3: unexpected EOF")
	if out != "line 2 col 3: unexpected EOF\n2 | x +\n"+"    "+"  ^" {
		t.Errorf("got %q", out)
	}

	plain := FormatErrorWithSnippet("", "some other error")
	if plain != "some other error" {
		t.Errorf("non-parser errors pass through, got %q", plain)
	}
}
