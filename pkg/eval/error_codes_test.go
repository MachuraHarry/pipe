package eval

import (
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/object"
)

func assertErrorContains(t *testing.T, input string, needle string) {
	t.Helper()
	result := parseAndEval(t, input)
	if result == nil {
		t.Fatalf("%q: got nil, expected error", input)
	}
	if result.Type() != object.ERROR {
		t.Fatalf("%q: expected error, got %s (%q)", input, result.Type(), result.Inspect())
	}
	msg := result.Inspect()
	if !strings.Contains(msg, needle) {
		t.Errorf("%q: error %q does not contain %q", input, msg, needle)
	}
}

func TestEvalE001UndefinedVar(t *testing.T) {
	expectError(t, "no_such_var")
	assertErrorContains(t, "no_such_var", "E001")
}

func TestEvalE002TypeMismatch(t *testing.T) {
	expectError(t, `"a" + 1`)
	assertErrorContains(t, `"a" + 1`, "E002")

	expectError(t, "true - 1")
	assertErrorContains(t, "true - 1", "E002")

	expectError(t, "nil + 1")
	assertErrorContains(t, "nil + 1", "E002")
}

func TestEvalE003DivisionByZero(t *testing.T) {
	expectError(t, "1 / 0")
	assertErrorContains(t, "1 / 0", "E003")

	expectError(t, "1 % 0")
	assertErrorContains(t, "1 % 0", "E003")
}

func TestEvalE004NotAFunction(t *testing.T) {
	expectError(t, "42 1")
	assertErrorContains(t, "42 1", "E004")

	expectError(t, "true 1")
	assertErrorContains(t, "true 1", "E004")

	expectError(t, "nil 1")
	assertErrorContains(t, "nil 1", "E004")
}

func TestEvalE005UnsupportedOperator(t *testing.T) {
	expectError(t, `"a" * "b"`)
	assertErrorContains(t, `"a" * "b"`, "E005")

	expectError(t, `"x" - "y"`)
	assertErrorContains(t, `"x" - "y"`, "E005")
}

func TestEvalE006CannotIndex(t *testing.T) {
	expectError(t, "42[0]")
	assertErrorContains(t, "42[0]", "E006")

	expectError(t, "true[0]")
	assertErrorContains(t, "true[0]", "E006")

	expectError(t, "nil[0]")
	assertErrorContains(t, "nil[0]", "E006")
}

func TestEvalTryCatchWithErrorCode(t *testing.T) {
	expectValue(t, "try\n    1 / 0\ncatch e\n    \"caught: \" ++ e", "caught: <test>: E003: division by zero")
}
