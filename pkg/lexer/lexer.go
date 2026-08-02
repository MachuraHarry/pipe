package lexer

import (
	"strconv"
	"strings"
)

type Lexer struct {
	input       string
	pos         int
	readPos     int
	ch          byte
	line        int
	col         int
	indentStack []int
	parenDepth  int
	atLineStart bool
	tokens      []Token // buffered tokens for DEDENT
}

func New(input string) *Lexer {
	l := &Lexer{
		input:       input,
		line:        1,
		col:         0,
		indentStack: []int{0},
		atLineStart: true,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) NextToken() Token {
	if len(l.tokens) > 0 {
		tok := l.tokens[0]
		l.tokens = l.tokens[1:]
		return tok
	}

	if l.atLineStart {
		return l.handleLineStart()
	}

	l.skipWhitespace()

	return l.scanToken()
}

func (l *Lexer) handleLineStart() Token {
	l.atLineStart = false

	indent := 0
	for l.ch == ' ' {
		indent++
		l.readChar()
	}

	// Empty line: skip
	if l.ch == '\n' {
		l.readLine()
		l.atLineStart = true
		return l.NextToken()
	}

	// EOF after whitespace on last line
	if l.ch == 0 {
		return l.emitPendingDedents()
	}

	// Comment line: skip
	if l.ch == '-' && l.peekChar() == '-' {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		if l.ch == '\n' {
			l.readLine()
			l.atLineStart = true
			return l.NextToken()
		}
		return l.emitPendingDedents()
	}

	if l.parenDepth > 0 {
		// Inside brackets: no indent tracking
		return l.scanToken()
	}

	top := l.indentStack[len(l.indentStack)-1]

	if indent > top {
		l.indentStack = append(l.indentStack, indent)
		// Return INDENT; position is already at first non-space char
		return Token{Type: INDENT, Literal: "INDENT", Line: l.line, Col: indent}
	}

	if indent < top {
		// Pop deeper levels, buffer DEDENT tokens
		for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > indent {
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			l.tokens = append(l.tokens, Token{Type: DEDENT, Literal: "DEDENT", Line: l.line, Col: indent})
		}

		// Validate indent matches a previous level
		if len(l.indentStack) == 0 || l.indentStack[len(l.indentStack)-1] != indent {
			l.tokens = append(l.tokens, Token{
				Type:    ILLEGAL,
				Literal: "unexpected indent",
				Line:    l.line,
				Col:     indent,
			})
		}

		// Return first buffered token; position stays at first non-space char
		if len(l.tokens) > 0 {
			tok := l.tokens[0]
			l.tokens = l.tokens[1:]
			return tok
		}
	}

	// Same indent level: scan normally
	return l.scanToken()
}

func (l *Lexer) emitPendingDedents() Token {
	if len(l.indentStack) > 1 {
		l.indentStack = l.indentStack[:len(l.indentStack)-1]
		return Token{Type: DEDENT, Literal: "DEDENT", Line: l.line, Col: l.col}
	}
	return Token{Type: EOF, Literal: "EOF", Line: l.line, Col: l.col}
}

func (l *Lexer) scanToken() Token {
	var tok Token

	switch l.ch {
	case 0:
		tok = l.emitPendingDedents()

	case '\n':
		tok = Token{Type: NEWLINE, Literal: "NEWLINE", Line: l.line, Col: l.col}
		l.readLine()
		return tok

	case '+':
		if l.peekChar() == '+' {
			l.readChar()
			tok = Token{Type: CONCAT, Literal: "++", Line: l.line, Col: l.col - 1}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: PLUSEQ, Literal: "+=", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: PLUS, Literal: "+", Line: l.line, Col: l.col}
		}

	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: FAT_ARROW, Literal: "->", Line: l.line, Col: l.col - 1}
		} else if l.peekChar() == '-' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			if l.ch == '\n' {
				l.readLine()
			}
			if l.ch == 0 {
				return l.emitPendingDedents()
			}
			return l.NextToken()
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: MINUSEQ, Literal: "-=", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: MINUS, Literal: "-", Line: l.line, Col: l.col}
		}

	case '*':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: STAREQ, Literal: "*=", Line: l.line, Col: l.col - 1}
		} else if l.peekChar() == '*' {
			l.readChar()
			tok = Token{Type: POWER, Literal: "**", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: STAR, Literal: "*", Line: l.line, Col: l.col}
		}

	case '/':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: SLASHEQ, Literal: "/=", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: SLASH, Literal: "/", Line: l.line, Col: l.col}
		}

	case '%':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: PERCENTEQ, Literal: "%=", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: PERCENT, Literal: "%", Line: l.line, Col: l.col}
		}

	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: EQ, Literal: "==", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: ASSIGN, Literal: "=", Line: l.line, Col: l.col}
		}

	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: "!=", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: BANG, Literal: "!", Line: l.line, Col: l.col}
		}

	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: LTE, Literal: "<=", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: LT, Literal: "<", Line: l.line, Col: l.col}
		}

	case '>':
		if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: ARROW2, Literal: ">>", Line: l.line, Col: l.col - 1}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: GTE, Literal: ">=", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: ARROW, Literal: ">", Line: l.line, Col: l.col}
		}

	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = Token{Type: AND, Literal: "&&", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: ILLEGAL, Literal: "&", Line: l.line, Col: l.col}
		}

	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = Token{Type: OR, Literal: "||", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: PIPE, Literal: "|", Line: l.line, Col: l.col}
		}

	case '(':
		l.parenDepth++
		tok = Token{Type: LPAREN, Literal: "(", Line: l.line, Col: l.col}

	case ')':
		l.parenDepth--
		if l.parenDepth < 0 {
			l.parenDepth = 0
		}
		tok = Token{Type: RPAREN, Literal: ")", Line: l.line, Col: l.col}

	case '[':
		l.parenDepth++
		tok = Token{Type: LBRACKET, Literal: "[", Line: l.line, Col: l.col}

	case ']':
		l.parenDepth--
		if l.parenDepth < 0 {
			l.parenDepth = 0
		}
		tok = Token{Type: RBRACKET, Literal: "]", Line: l.line, Col: l.col}

	case '{':
		l.parenDepth++
		tok = Token{Type: LBRACE, Literal: "{", Line: l.line, Col: l.col}

	case '}':
		l.parenDepth--
		if l.parenDepth < 0 {
			l.parenDepth = 0
		}
		tok = Token{Type: RBRACE, Literal: "}", Line: l.line, Col: l.col}

	case ',':
		tok = Token{Type: COMMA, Literal: ",", Line: l.line, Col: l.col}

	case ';':
		tok = Token{Type: SEMICOLON, Literal: ";", Line: l.line, Col: l.col}

	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			tok = Token{Type: DOTDOT, Literal: "..", Line: l.line, Col: l.col - 1}
		} else {
			tok = Token{Type: DOT, Literal: ".", Line: l.line, Col: l.col}
		}

	case ':':
		tok = Token{Type: COLON, Literal: ":", Line: l.line, Col: l.col}

	case '"':
		tok = l.readString()

	case '`':
		tok = l.readBacktickString()

	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			tokType := LookupIdent(literal)
			tok = Token{Type: tokType, Literal: literal, Line: l.line, Col: l.col - len(literal)}
			return tok
		} else if isDigit(l.ch) {
			literal, isFloat := l.readNumber()
			col := l.col - len(literal)
			if isFloat {
				tok = Token{Type: FLOAT, Literal: literal, Line: l.line, Col: col}
			} else {
				tok = Token{Type: INT, Literal: literal, Line: l.line, Col: col}
			}
			return tok
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Line: l.line, Col: l.col}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readLine() {
	l.readChar() // consume the newline; l.ch is now the first char of the next line
	l.line++
	l.col = 1 // that char sits in column 1, so following positions are 1-based
	l.atLineStart = true
}

func (l *Lexer) readBacktickString() Token {
	startCol := l.col
	l.readChar() // skip opening backtick

	var buf strings.Builder
	for l.ch != '`' && l.ch != 0 {
		buf.WriteByte(l.ch)
		if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		l.readChar()
	}

	if l.ch == 0 {
		return Token{Type: ILLEGAL, Literal: "unterminated backtick string", Line: l.line, Col: startCol}
	}

	return Token{Type: STRING, Literal: buf.String(), Line: l.line, Col: startCol}
}

func (l *Lexer) readString() Token {
	startCol := l.col
	l.readChar()

	var buf strings.Builder

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case 'r':
				buf.WriteByte('\r')
			case '\\':
				buf.WriteByte('\\')
			case '"':
				buf.WriteByte('"')
			case '0':
				buf.WriteByte(0)
			default:
				buf.WriteByte('\\')
				buf.WriteByte(l.ch)
			}
		} else {
			buf.WriteByte(l.ch)
			if l.ch == '\n' {
				l.line++
				l.col = 0
			}
		}
		l.readChar()
	}

	if l.ch == 0 {
		return Token{Type: ILLEGAL, Literal: "unterminated string", Line: l.line, Col: startCol}
	}

	return Token{Type: STRING, Literal: buf.String(), Line: l.line, Col: startCol}
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readNumber() (string, bool) {
	start := l.pos
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[start:l.pos], isFloat
}

func (l *Lexer) TokenizeAll() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF || tok.Type == ILLEGAL {
			break
		}
	}
	return tokens
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func ParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func ParseInt(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}
