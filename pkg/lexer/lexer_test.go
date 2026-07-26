package lexer

import (
	"testing"
)

func TestLiterals(t *testing.T) {
	input := `42 3.14 "hello world" true false nil`
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{INT, FLOAT, STRING, TRUE, FALSE, NIL, EOF})
	if tokens[2].Literal != "hello world" {
		t.Errorf("STRING literal: want 'hello world', got %q", tokens[2].Literal)
	}
}

func TestOperators(t *testing.T) {
	input := `+ - * / % == != < <= >= ++`
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		PLUS, MINUS, STAR, SLASH, PERCENT,
		EQ, NOT_EQ, LT, LTE, GTE, CONCAT,
		EOF,
	})
}

func TestPipelineAndMatch(t *testing.T) {
	input := `x > f > g`
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{IDENT, ARROW, IDENT, ARROW, IDENT, EOF})
}

func TestMatchArrow(t *testing.T) {
	input := "match x\n    | 0 -> \"null\"\n    | _ -> \"other\"\n"
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		MATCHKW, IDENT, NEWLINE,
		INDENT,
		PIPE, INT, MATCH, STRING, NEWLINE,
		PIPE, IDENT, MATCH, STRING, NEWLINE,
		DEDENT, EOF,
	})
}

func TestFunctionDef(t *testing.T) {
	input := "fn greet name\n    \"Hallo \" ++ name"
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		FN, IDENT, IDENT, NEWLINE,
		INDENT,
		STRING, CONCAT, IDENT,
		DEDENT, EOF,
	})
}

func TestVariableDef(t *testing.T) {
	input := "name: \"Welt\"\nzahlen: [1, 2, 3]"
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		IDENT, COLON, STRING, NEWLINE,
		IDENT, COLON, LBRACKET, INT, COMMA, INT, COMMA, INT, RBRACKET,
		EOF,
	})
}

func TestIfElse(t *testing.T) {
	input := "if x > 10\n    print \"groß\"\nelse\n    print \"klein\""
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		IF, IDENT, ARROW, INT, NEWLINE,
		INDENT,
		IDENT, STRING, NEWLINE,
		DEDENT,
		ELSE, NEWLINE,
		INDENT,
		IDENT, STRING,
		DEDENT, EOF,
	})
}

func TestComments(t *testing.T) {
	input := "-- Dies ist ein Kommentar\nx: 42\n-- noch ein Kommentar\ny: 10"
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		IDENT, COLON, INT, NEWLINE,
		IDENT, COLON, INT,
		EOF,
	})
}

func TestMapLiteral(t *testing.T) {
	input := `{a: 1, b: 2, c: 3}`
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		LBRACE, IDENT, COLON, INT, COMMA, IDENT, COLON, INT, COMMA, IDENT, COLON, INT, RBRACE,
		EOF,
	})
}

func TestMultilineBrackets(t *testing.T) {
	input := "[1,\n 2,\n 3]"
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		LBRACKET, INT, COMMA, NEWLINE, INT, COMMA, NEWLINE, INT, RBRACKET,
		EOF,
	})
}

func TestNestedIndent(t *testing.T) {
	input := "fn outer\n    fn inner\n        x: 42\n        print x\n    inner \"Hallo\""
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		FN, IDENT, NEWLINE,
		INDENT,
		FN, IDENT, NEWLINE,
		INDENT,
		IDENT, COLON, INT, NEWLINE,
		IDENT, IDENT, NEWLINE,
		DEDENT,
		IDENT, STRING,
		DEDENT, EOF,
	})
}

func TestPipelineVertical(t *testing.T) {
	input := "users\n    > filter .age >= 18\n    > map .name\n    > sort\n    > print"
	tokens := New(input).TokenizeAll()

	assertTokens(t, tokens, []TokenType{
		IDENT, NEWLINE,
		INDENT,
		ARROW, IDENT, DOT, IDENT, GTE, INT, NEWLINE,
		ARROW, IDENT, DOT, IDENT, NEWLINE,
		ARROW, IDENT, NEWLINE,
		ARROW, IDENT,
		DEDENT, EOF,
	})
}

func TestStringEscapes(t *testing.T) {
	input := `"line1\nline2\ttab"`
	tokens := New(input).TokenizeAll()

	if len(tokens) < 2 || tokens[0].Type != STRING {
		t.Fatalf("expected STRING, got %v", tokens)
	}
	if tokens[0].Literal != "line1\nline2\ttab" {
		t.Errorf("escaped string: got %q", tokens[0].Literal)
	}
}

func TestEmptyInput(t *testing.T) {
	tokens := New("").TokenizeAll()
	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Errorf("empty input: expected single EOF, got %d tokens", len(tokens))
	}
}

func TestOnlyComments(t *testing.T) {
	input := "-- comment 1\n-- comment 2"
	tokens := New(input).TokenizeAll()

	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Errorf("only comments: expected single EOF, got %d tokens", len(tokens))
	}
}

func assertTokens(t *testing.T, tokens []Token, expected []TokenType) {
	t.Helper()

	if len(tokens) != len(expected) {
		var got []string
		for _, tok := range tokens {
			got = append(got, tokenNames[tok.Type])
		}
		t.Fatalf("token count mismatch:\n  want (%d): %v\n  got  (%d): %v",
			len(expected), tokenNamesList(expected), len(tokens), got)
	}

	for i, exp := range expected {
		if got := tokens[i].Type; got != exp {
			t.Errorf("token[%d]: want %s, got %s (%q)",
				i, tokenNames[exp], tokenNames[got], tokens[i].Literal)
		}
	}
}

func tokenNamesList(types []TokenType) []string {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = tokenNames[t]
	}
	return names
}
