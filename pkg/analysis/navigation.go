package analysis

import (
	"errors"

	"github.com/MachuraHarry/pipe/pkg/ast"
)

// Location is a source range, used for navigation results.
type Location struct {
	Start ast.Position
	End   ast.Position
}

// DefinitionAt returns the definition range of the symbol under the cursor.
// ok is false when the position has no resolvable symbol or it is a builtin
// (builtins have no source location).
func (a *Analysis) DefinitionAt(line, col int) (Location, bool) {
	sym := a.SymbolAt(line, col)
	if sym == nil || sym.Kind == KindBuiltin {
		return Location{}, false
	}
	return Location{Start: sym.Pos, End: sym.End}, true
}

// ReferencesAt returns every occurrence of the symbol under the cursor,
// including its definition. Builtins yield an empty result.
func (a *Analysis) ReferencesAt(line, col int) []Location {
	sym := a.SymbolAt(line, col)
	if sym == nil || sym.Kind == KindBuiltin {
		return nil
	}
	out := make([]Location, 0, len(sym.Usages)+1)
	out = append(out, Location{Start: sym.Pos, End: sym.End})
	for _, u := range sym.Usages {
		out = append(out, Location{Start: u.Pos, End: u.End})
	}
	return out
}

// ValidateIdentifier checks that newName can be used as an identifier and is
// not a reserved keyword.
func ValidateIdentifier(newName string) error {
	if newName == "" {
		return errors.New("name must not be empty")
	}
	if isIdentByte(newName[0]) && !isDigitByte(newName[0]) {
		ok := true
		for i := 0; i < len(newName); i++ {
			if !isIdentByte(newName[i]) {
				ok = false
				break
			}
		}
		if ok {
			for _, kw := range Keywords {
				if newName == kw {
					return errors.New("name is a reserved keyword")
				}
			}
			return nil
		}
	}
	return errors.New("name must be a valid identifier")
}

func isDigitByte(c byte) bool {
	return c >= '0' && c <= '9'
}

// RenameAt validates newName and returns the ranges that must be changed.
// The first error is a validation error; the second (if non-nil) is an
// internal problem resolving the symbol.
func (a *Analysis) RenameAt(line, col int, newName string) ([]Location, error) {
	if err := ValidateIdentifier(newName); err != nil {
		return nil, err
	}
	return a.ReferencesAt(line, col), nil
}
