package analysis

// SemanticToken types (indices into SemanticTokenTypes legend).
const (
	SemComment = iota
	SemKeyword
	SemString
	SemNumber
	SemOperator
	SemFunction
	SemVariable
	SemParameter
	SemEnum
	SemEnumMember
	SemNamespace
)

// SemanticTokenTypes is the LSP token type legend, in index order.
var SemanticTokenTypes = []string{
	"comment", "keyword", "string", "number", "operator",
	"function", "variable", "parameter", "enum", "enumMember", "namespace",
}

// SemanticToken is one classified source range (1-based line/col).
type SemanticToken struct {
	Line   int
	Col    int
	Length int
	Type   int
}

// SemanticTokens classifies every character of source into semantic tokens.
// analysis is optional; without it identifiers fall back to "variable".
// Multi-line backtick strings are split into one token per line so the result
// can be encoded with the LSP (per-line) semantic token format.
func SemanticTokens(source string, a *Analysis) []SemanticToken {
	var out []SemanticToken
	n := len(source)
	line, col := 1, 1
	i := 0

	for i < n {
		c := source[i]
		switch {
		case c == '\n':
			line++
			col = 1
			i++

		case c == ' ' || c == '\t' || c == '\r':
			i++
			col++

		case c == '-' && i+1 < n && source[i+1] == '-':
			startIdx, startCol := i, col
			for i < n && source[i] != '\n' {
				i++
			}
			out = append(out, SemanticToken{Line: line, Col: startCol, Length: i - startIdx, Type: SemComment})
			col = col + (i - startIdx)

		case c == '"' || c == '`':
			quote := c
			startIdx, startLine, startCol := i, line, col
			i++ // skip opening quote
			if quote == '"' {
				for i < n && source[i] != '"' {
					if source[i] == '\\' && i+1 < n {
						i += 2
					} else {
						i++
					}
				}
				if i < n {
					i++ // closing quote
				}
			} else {
				for i < n && source[i] != '`' {
					i++
				}
				if i < n {
					i++
				}
			}
			emitSegment(&out, source, startIdx, i, startLine, startCol, SemString)
			line, col = afterSegment(source, startIdx, i, startLine, startCol)

		case c >= '0' && c <= '9':
			startIdx, startCol := i, col
			for i < n && source[i] >= '0' && source[i] <= '9' {
				i++
			}
			if i < n && source[i] == '.' && i+1 < n && source[i+1] >= '0' && source[i+1] <= '9' {
				i++
				for i < n && source[i] >= '0' && source[i] <= '9' {
					i++
				}
			}
			out = append(out, SemanticToken{Line: line, Col: startCol, Length: i - startIdx, Type: SemNumber})
			col += i - startIdx

		case isIdentStart(c):
			startIdx, startCol := i, col
			for i < n && isIdentByte(source[i]) {
				i++
			}
			word := source[startIdx:i]
			out = append(out, SemanticToken{
				Line:   line,
				Col:    startCol,
				Length: i - startIdx,
				Type:   classifyIdent(a, word, line, startCol),
			})
			col += i - startIdx

		default:
			// Operator or punctuation: group consecutive operator chars.
			startIdx, startCol := i, col
			for i < n && !isSpaceByte(source[i]) && !isIdentByte(source[i]) &&
				source[i] != '\n' && source[i] != '"' && source[i] != '`' &&
				!(source[i] == '-' && i+1 < n && source[i+1] == '-') {
				i++
			}
			out = append(out, SemanticToken{Line: line, Col: startCol, Length: i - startIdx, Type: SemOperator})
			col += i - startIdx
		}
	}

	return out
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r'
}

func classifyIdent(a *Analysis, word string, line, col int) int {
	for _, kw := range Keywords {
		if word == kw {
			return SemKeyword
		}
	}
	if a != nil {
		if sym := a.SymbolAt(line, col); sym != nil {
			switch sym.Kind {
			case KindFunction:
				return SemFunction
			case KindVariable:
				return SemVariable
			case KindParameter:
				return SemParameter
			case KindEnum:
				return SemEnum
			case KindEnumMember:
				return SemEnumMember
			case KindModule:
				return SemNamespace
			case KindBuiltin:
				return SemFunction
			}
		}
	}
	return SemVariable
}

// emitSegment appends a token for the byte range [startIdx,endIdx), splitting
// it into one token per line when it contains newlines.
func emitSegment(out *[]SemanticToken, source string, startIdx, endIdx, startLine, startCol, typ int) {
	line, col := startLine, startCol
	segStart, segCol := startIdx, startCol
	for i := startIdx; i < endIdx; i++ {
		if source[i] == '\n' {
			*out = append(*out, SemanticToken{Line: line, Col: segCol, Length: i - segStart, Type: typ})
			line++
			segStart = i + 1
			segCol = 1
		}
	}
	if segStart < endIdx {
		*out = append(*out, SemanticToken{Line: line, Col: segCol, Length: endIdx - segStart, Type: typ})
	}
	_ = col
}

// afterSegment returns the (line, col) position just after the byte range
// [startIdx,endIdx), starting from (startLine, startCol).
func afterSegment(source string, startIdx, endIdx, startLine, startCol int) (int, int) {
	line, col := startLine, startCol
	for i := startIdx; i < endIdx; i++ {
		if source[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}
