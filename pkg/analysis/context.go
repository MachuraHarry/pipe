package analysis

import (
	"unicode"

	"github.com/MachuraHarry/pipe/pkg/lexer"
)

// ctxToken is a lexed token with its source position.
type ctxToken struct {
	Type    lexer.TokenType
	Literal string
	Line    int
	Col     int // 1-based column of first character
}

// tokenizeAll lexes the whole source into positioned tokens.
func tokenizeAll(source string) []ctxToken {
	l := lexer.New(source)
	var out []ctxToken
	for {
		t := l.NextToken()
		out = append(out, ctxToken{Type: t.Type, Literal: t.Literal, Line: t.Line, Col: t.Col})
		if t.Type == lexer.EOF || t.Type == lexer.ILLEGAL {
			break
		}
	}
	return out
}

// wordAt returns the identifier-ish word being typed at the cursor, plus its
// start column. col is 1-based.
func wordAt(source string, line, col int) (string, int) {
	lines := splitLines(source)
	if line < 1 || line > len(lines) {
		return "", col
	}
	ln := lines[line-1]
	// cursor col is 1-based; byte index = col-1 (ASCII source for identifiers)
	idx := col - 1
	if idx < 0 {
		idx = 0
	}
	if idx > len(ln) {
		idx = len(ln)
	}
	start := idx
	for start > 0 && isIdentByte(ln[start-1]) {
		start--
	}
	end := idx
	for end < len(ln) && isIdentByte(ln[end]) {
		end++
	}
	return ln[start:end], start + 1 // return word and its 1-based start col
}

// isIdentByte matches Pipe identifier characters.
func isIdentByte(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r)
}

// prevSignificantToken returns the token before the given position that is not
// NEWLINE/INDENT/DEDENT.
func prevSignificantToken(toks []ctxToken, line, col int) *ctxToken {
	for i := len(toks) - 1; i >= 0; i-- {
		t := toks[i]
		if t.Line > line || (t.Line == line && t.Col >= col) {
			continue
		}
		switch t.Type {
		case lexer.NEWLINE, lexer.INDENT, lexer.DEDENT, lexer.EOF:
			continue
		}
		return &toks[i]
	}
	return nil
}

// tokenAt returns the token whose range contains the position (1-based).
// Layout tokens (INDENT/DEDENT/NEWLINE/EOF) are ignored.
func tokenAt(toks []ctxToken, line, col int) *ctxToken {
	for i := range toks {
		t := &toks[i]
		switch t.Type {
		case lexer.INDENT, lexer.DEDENT, lexer.NEWLINE, lexer.EOF, lexer.ILLEGAL:
			continue
		}
		if t.Line != line {
			continue
		}
		start := t.Col
		end := t.Col + len(t.Literal) - 1
		if col >= start && col <= end {
			return t
		}
	}
	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
