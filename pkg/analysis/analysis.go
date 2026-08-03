package analysis

import (
	"github.com/MachuraHarry/pipe/pkg/ast"
)

// SymbolKind classifies a defined name.
type SymbolKind int

const (
	KindFunction SymbolKind = iota
	KindVariable
	KindParameter
	KindEnum
	KindEnumMember
	KindModule
	KindBuiltin
)

func (k SymbolKind) String() string {
	switch k {
	case KindFunction:
		return "Function"
	case KindVariable:
		return "Variable"
	case KindParameter:
		return "Parameter"
	case KindEnum:
		return "Enum"
	case KindEnumMember:
		return "EnumMember"
	case KindModule:
		return "Module"
	case KindBuiltin:
		return "Builtin"
	}
	return "Unknown"
}

// Symbol is a named definition (function, variable, parameter, enum, ...).
type Symbol struct {
	Name    string
	Kind    SymbolKind
	Pos     ast.Position // start of the name token (1-based line/col)
	End     ast.Position // end of the name token (inclusive)
	Doc     string
	Builtin *BuiltinDoc
	Usages  []*Reference // resolved references pointing at this symbol
}

// Range returns the [start,end] position range of the symbol name.
func (s *Symbol) Range() (ast.Position, ast.Position) {
	return s.Pos, s.End
}

// Reference is an occurrence of a name that is not its definition.
type Reference struct {
	Name   string
	Pos    ast.Position
	End    ast.Position
	Symbol *Symbol // resolved definition; nil if unresolved
}

// Range returns the [start,end] position range of the reference.
func (r *Reference) Range() (ast.Position, ast.Position) {
	return r.Pos, r.End
}

// ScopeType distinguishes global from function scopes. Pipe blocks do not
// create their own scopes at runtime, so only these two are needed.
type ScopeType int

const (
	ScopeGlobal ScopeType = iota
	ScopeFunction
)

func (t ScopeType) String() string {
	if t == ScopeGlobal {
		return "global"
	}
	return "function"
}

// Scope is a lexical namespace.
type Scope struct {
	Parent  *Scope
	Type    ScopeType
	Symbols map[string]*Symbol
	Start   ast.Position // where the scope becomes active (fn name token)
	End     ast.Position // last statement position in the scope (inclusive)
}

// Contains reports whether position falls inside the scope's source range.
// The global scope spans everything (Start is zero).
func (s *Scope) Contains(line, col int) bool {
	if s.Type == ScopeGlobal {
		return true
	}
	if line < s.Start.Line {
		return false
	}
	if line == s.Start.Line && col < s.Start.Col {
		return false
	}
	if s.End.Line < 0 {
		return true
	}
	if line > s.End.Line {
		return false
	}
	if line == s.End.Line && col > s.End.Col {
		return false
	}
	return true
}

// Lookup resolves name in this scope and its ancestors.
func (s *Scope) Lookup(name string) *Symbol {
	for cur := s; cur != nil; cur = cur.Parent {
		if sym, ok := cur.Symbols[name]; ok {
			return sym
		}
	}
	return nil
}

// Analysis is the result of analyzing one source file.
type Analysis struct {
	Root       *Scope
	Scopes     []*Scope
	Symbols    []*Symbol
	References []*Reference
	Unresolved []*Reference
}

// SymbolAt returns the symbol whose definition range contains the position,
// or the symbol resolved at a reference at that position. Returns nil if the
// position does not hit a symbol.
func (a *Analysis) SymbolAt(line, col int) *Symbol {
	for _, s := range a.Symbols {
		if within(s.Pos, s.End, line, col) {
			return s
		}
	}
	for _, r := range a.References {
		if within(r.Pos, r.End, line, col) {
			return r.Symbol
		}
	}
	return nil
}

// SymbolAtName returns the first symbol with the given name.
func (a *Analysis) SymbolAtName(name string) *Symbol {
	for _, s := range a.Symbols {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// ReferencesTo returns all references plus the definition of the given symbol.
func (a *Analysis) ReferencesTo(s *Symbol) []*Reference {
	out := make([]*Reference, 0, len(s.Usages))
	out = append(out, s.Usages...)
	return out
}

// ScopeAt returns the innermost scope active at a position, or the global scope.
func (a *Analysis) ScopeAt(line, col int) *Scope {
	var best *Scope
	for _, s := range a.Scopes {
		if !s.Contains(line, col) {
			continue
		}
		if best == nil || depth(s) > depth(best) {
			best = s
		}
	}
	if best == nil {
		return a.Root
	}
	return best
}

func depth(s *Scope) int {
	d := 0
	for cur := s.Parent; cur != nil; cur = cur.Parent {
		d++
	}
	return d
}

// VisibleSymbolsAt returns symbols that a completion list at the position
// should offer: the definitions in the active scope chain plus all builtins.
func (a *Analysis) VisibleSymbolsAt(line, col int) []*Symbol {
	scope := a.ScopeAt(line, col)
	seen := make(map[string]bool)
	var out []*Symbol
	for cur := scope; cur != nil; cur = cur.Parent {
		for name, sym := range cur.Symbols {
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, sym)
		}
	}
	return out
}

func within(start, end ast.Position, line, col int) bool {
	if line < start.Line || line > end.Line {
		return false
	}
	if line == start.Line && col < start.Col {
		return false
	}
	if line == end.Line && col > end.Col {
		return false
	}
	return true
}

func identEnd(id *ast.Identifier) ast.Position {
	return ast.Position{Line: id.Line, Col: id.Col + len(id.Value) - 1}
}

func endFromPos(pos ast.Position, length int) ast.Position {
	return ast.Position{Line: pos.Line, Col: pos.Col + length - 1}
}

// builtinSymbol returns a synthetic Symbol for a builtin name.
func builtinSymbol(name string) *Symbol {
	doc, ok := builtinIndex[name]
	if !ok {
		return nil
	}
	return &Symbol{
		Name:    name,
		Kind:    KindBuiltin,
		Doc:     doc.Description,
		Builtin: &doc,
	}
}

// Keywords are reserved words offered for completion.
var Keywords = []string{
	"fn", "if", "else", "while", "for", "in", "as",
	"break", "continue", "return", "import", "export", "enum",
	"defer", "try", "catch", "try_ai", "match", "test", "not",
	"true", "false", "nil",
}

// Snippets provides expandable code templates for completion.
type Snippet struct {
	Label      string
	InsertText string
	Detail     string
}

var Snippets = []Snippet{
	{Label: "fn", InsertText: "fn ${1:name} ${2:params}\n    $0", Detail: "Function definition"},
	{Label: "if", InsertText: "if ${1:condition}\n    $0", Detail: "Conditional"},
	{Label: "if-else", InsertText: "if ${1:condition}\n    $0\nelse\n    ", Detail: "Conditional with else"},
	{Label: "for-in", InsertText: "for ${1:item} in ${2:list}\n    $0", Detail: "Loop over list"},
	{Label: "for", InsertText: "for ${1:i}: 0; ${1:i} < ${2:n}; ${1:i}: ${1:i} + 1\n    $0", Detail: "C-style loop"},
	{Label: "match", InsertText: "match ${1:value}\n    | $2 -> $0", Detail: "Pattern match"},
	{Label: "fn-lambda", InsertText: "fn(${1:x}) { $0 }", Detail: "Anonymous function"},
	{Label: "try", InsertText: "try\n    $0\ncatch e\n    ", Detail: "Try / catch"},
	{Label: "try_ai", InsertText: "try_ai\n    $0\ncatch e\n    ", Detail: "Self-healing try"},
	{Label: "test", InsertText: "test \"${1:description}\"\n    assert_eq $0", Detail: "Test block"},
	{Label: "import", InsertText: "import \"${1:module}\"", Detail: "Import a module"},
	{Label: "export", InsertText: "export fn ${1:name} ${2:params}\n    $0", Detail: "Export a function"},
	{Label: "enum", InsertText: "enum ${1:Name}: ${2:A}, ${3:B}", Detail: "Enum definition"},
}
