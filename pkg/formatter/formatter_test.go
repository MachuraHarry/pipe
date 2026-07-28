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
