package analysis

import (
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/lexer"
)

// AttachDocstrings associates `--!` docstring lines with the definition
// symbols they precede. A contiguous run of docstring lines ending exactly
// one line above a symbol's start position becomes that symbol's Doc.
// Only top-level-style definitions (functions, variables, enums) receive docs;
// parameters and enum members are skipped.
func AttachDocstrings(source string, a *Analysis) {
	if a == nil || len(a.Symbols) == 0 {
		return
	}
	byLine := docLinesByStart(lexer.CollectDocstrings(source))
	if len(byLine) == 0 {
		return
	}
	for _, sym := range a.Symbols {
		if !docKind(sym.Kind) {
			continue
		}
		doc, ok := docAbove(byLine, sym.Pos.Line)
		if ok && sym.Doc == "" {
			sym.Doc = doc
		}
	}
}

func docKind(k SymbolKind) bool {
	switch k {
	case KindFunction, KindVariable, KindEnum:
		return true
	}
	return false
}

// docLinesByStart indexes docstring lines by their line number and maps the
// last line of each contiguous run to the run text.
func docLinesByStart(lines []lexer.DocLine) map[int]string {
	byLine := make(map[int]string)
	for i := 0; i < len(lines); i++ {
		start := lines[i].Line
		var texts []string
		expect := start
		for i < len(lines) && lines[i].Line == expect {
			if lines[i].Text != "" {
				texts = append(texts, lines[i].Text)
			}
			expect++
			i++
		}
		// The run occupies [start, expect-1]; record it keyed by its end line.
		byLine[expect-1] = strings.Join(texts, " ")
		i--
	}
	return byLine
}

// docAbove returns the docstring run that ends exactly on line-1, if any.
func docAbove(byLine map[int]string, line int) (string, bool) {
	doc, ok := byLine[line-1]
	return doc, ok
}

// PosRange returns a symbol's source range for documentation rendering.
func PosRange(s *Symbol) (start, end ast.Position) {
	return s.Pos, s.End
}
