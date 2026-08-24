package analysis

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ast"
)

// Severity mirrors LSP DiagnosticSeverity.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInformation
	SeverityHint
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "Error"
	case SeverityWarning:
		return "Warning"
	case SeverityInformation:
		return "Information"
	case SeverityHint:
		return "Hint"
	}
	return "Unknown"
}

// Diagnostic is a single issue reported on a source range.
type Diagnostic struct {
	Range    Range
	Severity Severity
	Message  string
	Code     string
	Source   string
}

// Range is a source span (inclusive end).
type Range struct {
	Start ast.Position
	End   ast.Position
}

// LintResult collects all diagnostics for a document.
type LintResult struct {
	Diagnostics []Diagnostic
}

// Lint analyzes source and returns parse errors plus semantic diagnostics.
func Lint(source string) *LintResult {
	res := &LintResult{}
	a, errs := Analyze(source)
	res.addParseErrors(errs)
	res.addSemantic(a)
	return res
}

// LintProgram analyzes an already-parsed program.
func LintProgram(program *ast.Program) *LintResult {
	res := &LintResult{}
	res.addSemantic(AnalyzeProgram(program))
	return res
}

func (r *LintResult) add(d Diagnostic) {
	r.Diagnostics = append(r.Diagnostics, d)
}

func (r *LintResult) addParseErrors(errs []string) {
	for _, e := range errs {
		line, col, msg, ok := parseError(e)
		if !ok {
			continue
		}
		r.add(Diagnostic{
			Range: Range{
				Start: ast.Position{Line: line, Col: col},
				End:   ast.Position{Line: line, Col: col},
			},
			Severity: SeverityError,
			Message:  msg,
			Code:     "parse",
			Source:   "pipe",
		})
	}
}

// addSemantic reports undefined variables and unused definitions.
func (r *LintResult) addSemantic(a *Analysis) {
	if a == nil {
		return
	}
	for _, u := range a.Unresolved {
		r.add(Diagnostic{
			Range: Range{
				Start: u.Pos,
				End:   u.End,
			},
			Severity: SeverityError,
			Message:  "undefined variable: " + u.Name,
			Code:     "E001",
			Source:   "pipe",
		})
	}
	r.addUnusedWarnings(a)
}

// addUnusedWarnings flags defined variables/params that are never used.
// Globals referenced across functions are intentionally left alone: Pipe has
// a single global namespace and functions may reference module-level state.
func (r *LintResult) addUnusedWarnings(a *Analysis) {
	for _, s := range a.Symbols {
		switch s.Kind {
		case KindVariable, KindParameter:
		default:
			continue
		}
		if len(s.Usages) == 0 {
			if s.Name == "_" {
				continue
			}
			r.add(Diagnostic{
				Range: Range{
					Start: s.Pos,
					End:   s.End,
				},
				Severity: SeverityWarning,
				Message:  "unused variable: " + s.Name,
				Code:     "E007",
				Source:   "pipe",
			})
		}
	}
}

var parseErrRe = regexp.MustCompile(`^line (\d+) col (\d+): (.*)$`)

// parseError splits a parser error string like "line 3 col 5: message".
func parseError(e string) (int, int, string, bool) {
	m := parseErrRe.FindStringSubmatch(strings.TrimSpace(e))
	if m == nil {
		return 0, 0, "", false
	}
	line, err1 := strconv.Atoi(m[1])
	col, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, "", false
	}
	return line, col, m[3], true
}
