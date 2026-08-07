package lexer

import (
	"testing"
)

func FuzzLexer(f *testing.F) {
	f.Add("42")
	f.Add("x: 10\ny: 20")
	f.Add(`fn greet name\n    "Hello, " ++ name ++ "!"`)
	f.Add("print (2 + 2)")
	f.Add("")
	f.Add("\n\n\n")
	f.Add("äöü日本語")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("lexer panic on input %q: %v", input, r)
			}
		}()
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF || tok.Type == ILLEGAL {
				break
			}
		}
	})
}
