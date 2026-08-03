package analysis

import (
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ast"
)

// HoverInfo is the LSP hover payload. Contents are markdown strings.
type HoverInfo struct {
	Start    ast.Position // range of the hovered token
	End      ast.Position
	Contents []string
}

// keywordDocs describes reserved words for hover and completion detail.
var keywordDocs = map[string]string{
	"fn":       "Defines a function: `fn name params`.",
	"if":       "Conditional statement: `if condition` followed by an indented block.",
	"else":     "Alternative branch attached to `if`.",
	"while":    "Loop while the condition is true.",
	"for":      "C-style loop: `for i: 0; i < n; i: i + 1`.",
	"in":       "Loop keyword: `for x in list`.",
	"as":       "Type assertion or import alias.",
	"break":    "Exits the innermost loop.",
	"continue": "Skips to the next loop iteration.",
	"return":   "Returns a value from the current function.",
	"import":   "Imports a module: `import \"path\"`.",
	"export":   "Exports a function or value so other files can import it.",
	"enum":     "Defines a set of named constants: `enum Name: A, B, C`.",
	"defer":    "Schedules a statement to run when the enclosing function exits.",
	"try":      "Runs a block, catching errors into `catch e`.",
	"catch":    "Handles an error from a preceding `try` or `try_ai` block.",
	"try_ai":   "Like `try`, but asks an AI to repair failures before re-running.",
	"match":    "Pattern match: `match value` with `| pattern -> body` arms.",
	"test":     "Defines a test block: `test \"description\"`.",
	"not":      "Logical negation: `not x`.",
	"true":     "Boolean literal.",
	"false":    "Boolean literal.",
	"nil":      "The null value.",
}

// Hover returns markdown documentation for the symbol, builtin or keyword at
// the position. ok is false when nothing is hoverable.
func (a *Analysis) Hover(source string, line, col int) (HoverInfo, bool) {
	toks := tokenizeAll(source)
	t := tokenAt(toks, line, col)
	if t == nil {
		return HoverInfo{}, false
	}
	start := ast.Position{Line: t.Line, Col: t.Col}
	end := ast.Position{Line: t.Line, Col: t.Col + len(t.Literal) - 1}
	info := HoverInfo{Start: start, End: end}

	// User symbol at this position (definition or reference).
	if a != nil {
		if sym := a.SymbolAt(line, col); sym != nil {
			if sym.Builtin != nil {
				return builtinHover(sym.Builtin), true
			}
			info.Contents = append(info.Contents, symbolHoverMarkdown(sym))
			return info, true
		}
	}

	// Builtin by name.
	if b, ok := Builtin(t.Literal); ok {
		return builtinHover(&b), true
	}

	// Keyword.
	if doc, ok := keywordDocs[t.Literal]; ok {
		info.Contents = append(info.Contents, "**"+t.Literal+"** — "+doc)
		return info, true
	}

	return HoverInfo{}, false
}

func builtinHover(b *BuiltinDoc) HoverInfo {
	detail := "**" + b.Name + "**  \n`" + b.Signature
	if b.ReturnType != "" {
		detail += "` → " + b.ReturnType
	} else {
		detail += "`"
	}
	lines := []string{detail}
	if b.Description != "" {
		lines = append(lines, "", b.Description)
	}
	lines = append(lines, "", "_Category: "+b.Category+"_")
	return HoverInfo{Contents: lines}
}

func symbolHoverMarkdown(sym *Symbol) string {
	var b strings.Builder
	header := "**" + sym.Name + "**"
	switch sym.Kind {
	case KindFunction:
		header += "  \n`fn " + sym.Name + "`"
	case KindEnumMember:
		header += "  \n`enum " + sym.Name + "`"
	case KindModule:
		header += "  \n`module`"
	default:
		header += "  \n`" + strings.ToLower(sym.Kind.String()) + "`"
	}
	b.WriteString(header)
	if sym.Doc != "" {
		b.WriteString("\n\n" + sym.Doc)
	}
	return b.String()
}
