package lexer

import "fmt"

type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF

	// Literals
	IDENT  // x, y, name
	INT    // 42
	FLOAT  // 3.14
	STRING // "hello"

	// Operators
	ASSIGN   // =
	PLUS     // +
	MINUS    // -
	STAR     // *
	SLASH    // /
	PERCENT  // %
	EQ       // ==
	NOT_EQ   // !=
	LT       // <
	GT       // >
	LTE      // <=
	GTE      // >=
	CONCAT   // ++ (string concat)
	DOTDOT   // .. (slice range)

	// Pipeline
	PIPE   // |
	ARROW  // >
	MATCH  // ->

	// Punctuation
	LPAREN   // (
	RPAREN   // )
	LBRACKET // [
	RBRACKET // ]
	LBRACE   // {
	RBRACE   // }
	COMMA    // ,
	DOT      // .
	COLON    // :

	// Structure
	NEWLINE
	INDENT
	DEDENT

	// Keywords
	FN       // fn
	MATCHKW  // match
	IF       // if
	ELSE     // else
	WHILE    // while
	FOR      // for
	BREAK    // break
	CONTINUE // continue
	IMPORT   // import
	TRY      // try
	CATCH    // catch
	TRUE     // true
	FALSE    // false
	NIL      // nil
)

var keywords = map[string]TokenType{
	"fn":       FN,
	"match":    MATCHKW,
	"if":       IF,
	"else":     ELSE,
	"while":    WHILE,
	"for":      FOR,
	"break":    BREAK,
	"continue": CONTINUE,
	"import":   IMPORT,
	"try":      TRY,
	"catch":    CATCH,
	"true":     TRUE,
	"false":    FALSE,
	"nil":      NIL,
}

var tokenNames = map[TokenType]string{
	ILLEGAL:   "ILLEGAL",
	EOF:       "EOF",
	IDENT:     "IDENT",
	INT:       "INT",
	FLOAT:     "FLOAT",
	STRING:    "STRING",
	ASSIGN:    "=",
	PLUS:      "+",
	MINUS:     "-",
	STAR:      "*",
	SLASH:     "/",
	PERCENT:   "%",
	EQ:        "==",
	NOT_EQ:    "!=",
	LT:        "<",
	GT:        ">",
	LTE:       "<=",
	GTE:       ">=",
	CONCAT:    "++",
	DOTDOT:    "..",
	PIPE:      "|",
	ARROW:     ">",
	MATCH:     "->",
	LPAREN:    "(",
	RPAREN:    ")",
	LBRACKET:  "[",
	RBRACKET:  "]",
	LBRACE:    "{",
	RBRACE:    "}",
	COMMA:     ",",
	DOT:       ".",
	COLON:     ":",
	NEWLINE:   "NEWLINE",
	INDENT:    "INDENT",
	DEDENT:    "DEDENT",
	FN:        "fn",
	MATCHKW:   "match",
	IF:        "if",
	ELSE:      "else",
	WHILE:     "while",
	FOR:       "for",
	BREAK:     "break",
	CONTINUE:  "continue",
	IMPORT:    "import",
	TRY:       "try",
	CATCH:     "catch",
	TRUE:      "true",
	FALSE:     "false",
	NIL:       "nil",
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

func (t Token) String() string {
	return fmt.Sprintf("Token{%s, %q, line=%d, col=%d}", tokenNames[t.Type], t.Literal, t.Line, t.Col)
}

func TokenName(t TokenType) string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
