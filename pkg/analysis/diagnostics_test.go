package analysis

import "testing"

func diags(src string) []Diagnostic {
	return Lint(src).Diagnostics
}

func hasDiag(t *testing.T, ds []Diagnostic, code, contains string) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code {
			if contains != "" && !containsStr(d.Message, contains) {
				continue
			}
			return
		}
	}
	t.Fatalf("diagnostic with code %q (containing %q) not found in %+v", code, contains, ds)
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(sub) <= len(s) && indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestParseErrorDiagnostic(t *testing.T) {
	ds := diags("fn foo a\n    a +\n")
	hasDiag(t, ds, "parse", "")
	if ds[0].Severity != SeverityError {
		t.Errorf("parse severity = %v, want Error", ds[0].Severity)
	}
	if ds[0].Range.Start.Line != 2 {
		t.Errorf("parse error line = %d, want 2", ds[0].Range.Start.Line)
	}
}

func TestUndefinedVariableDiag(t *testing.T) {
	ds := diags("print missing_var\n")
	hasDiag(t, ds, "E001", "missing_var")
}

func TestUnusedVariableWarning(t *testing.T) {
	ds := diags("unused: 42\nprint \"hi\"\n")
	hasDiag(t, ds, "E007", "unused")
}

func TestUnusedIsNotFlaggedWhenUsed(t *testing.T) {
	ds := diags("used: 42\nprint used\n")
	for _, d := range ds {
		if d.Code == "E007" {
			t.Fatalf("unexpected unused warning: %+v", d)
		}
	}
}

func TestReassignmentDoesNotTriggerUnused(t *testing.T) {
	ds := diags("x: 1\nx: x + 1\nprint x\n")
	for _, d := range ds {
		if d.Code == "E007" {
			t.Fatalf("unexpected unused warning: %+v", d)
		}
		if d.Code == "E001" {
			t.Fatalf("unexpected undefined var: %+v", d)
		}
	}
}

func TestCompoundAssignResolves(t *testing.T) {
	ds := diags("x: 1\nx += 5\nprint x\n")
	for _, d := range ds {
		if d.Code == "E001" {
			t.Fatalf("unexpected undefined var: %+v", d)
		}
	}
}

func TestCleanFileHasNoDiagnostics(t *testing.T) {
	ds := diags(`fn add a b
    a + b

print add 1 2
`)
	if len(ds) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", ds)
	}
}
