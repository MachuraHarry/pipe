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
	ASSIGN    // =
	PLUS      // +
	MINUS     // -
	STAR      // *
	SLASH     // /
	PERCENT   // %
	EQ        // ==
	NOT_EQ    // !=
	LT        // <
	GT        // >
	LTE       // <=
	GTE       // >=
	CONCAT    // ++ (string concat)
	BANG      // !
	AND       // &&
	OR        // ||
	PLUSEQ    // +=
	MINUSEQ   // -=
	STAREQ    // *=
	SLASHEQ   // /=
	PERCENTEQ // %=
	POWER     // **
	DOTDOT    // .. (slice range)

	// Pipeline
	PIPE      // |
	ARROW     // >
	ARROW2    // >>
	FAT_ARROW // ->

	// Punctuation
	LPAREN    // (
	RPAREN    // )
	LBRACKET  // [
	RBRACKET  // ]
	LBRACE    // {
	RBRACE    // }
	COMMA     // ,
	SEMICOLON // ;
	DOT       // .
	COLON     // :

	// Structure
	NEWLINE
	INDENT
	DEDENT

	// Keywords
	FN       // fn
	MATCH_KW // match
	IF       // if
	ELSE     // else
	WHILE    // while
	FOR      // for
	BREAK    // break
	CONTINUE // continue
	IMPORT   // import
	EXPORT   // export
	ENUM     // enum
	DEFER    // defer
	RETURN   // return
	TRY      // try
	CATCH    // catch
	TRUE     // true
	FALSE    // false
	NIL      // nil
	TRYAI    // try_ai
	TEST     // test
)

var keywords = map[string]TokenType{
	"fn":       FN,
	"match":    MATCH_KW,
	"if":       IF,
	"else":     ELSE,
	"while":    WHILE,
	"for":      FOR,
	"break":    BREAK,
	"continue": CONTINUE,
	"import":   IMPORT,
	"export":   EXPORT,
	"enum":     ENUM,
	"defer":    DEFER,
	"return":   RETURN,
	"try":      TRY,
	"catch":    CATCH,
	"try_ai":   TRYAI,
	"true":     TRUE,
	"false":    FALSE,
	"nil":      NIL,
	"test":     TEST,
	"not":      BANG,
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
	BANG:      "!",
	AND:       "&&",
	OR:        "||",
	PLUSEQ:    "+=",
	MINUSEQ:   "-=",
	STAREQ:    "*=",
	SLASHEQ:   "/=",
	PERCENTEQ: "%=",
	POWER:     "**",
	DOTDOT:    "..",
	PIPE:      "|",
	ARROW:     ">",
	ARROW2:    ">>",
	FAT_ARROW: "->",
	LPAREN:    "(",
	RPAREN:    ")",
	LBRACKET:  "[",
	RBRACKET:  "]",
	LBRACE:    "{",
	RBRACE:    "}",
	COMMA:     ",",
	SEMICOLON: ";",
	DOT:       ".",
	COLON:     ":",
	NEWLINE:   "NEWLINE",
	INDENT:    "INDENT",
	DEDENT:    "DEDENT",
	FN:        "fn",
	MATCH_KW:  "match",
	IF:        "if",
	ELSE:      "else",
	WHILE:     "while",
	FOR:       "for",
	BREAK:     "break",
	CONTINUE:  "continue",
	IMPORT:    "import",
	EXPORT:    "export",
	ENUM:      "enum",
	DEFER:     "defer",
	RETURN:    "return",
	TRY:       "try",
	CATCH:     "catch",
	TRYAI:     "try_ai",
	TRUE:      "true",
	FALSE:     "false",
	NIL:       "nil",
	TEST:      "test",
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
