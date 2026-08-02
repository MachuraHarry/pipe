package formatter_test

import (
	"os"
	"strings"
	"testing"

	"github.com/harry/pipe/pkg/formatter"
	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/parser"
)

// TestExampleRoundtrip verifies that formatting any example program produces
// source that re-parses without errors. This guards the CallExpression and
// pipeline rendering against output the parser cannot consume.
func TestExampleRoundtrip(t *testing.T) {
	matches, _ := os.ReadDir("../../examples")
	for _, m := range matches {
		if !strings.HasSuffix(m.Name(), ".pipe") {
			continue
		}
		data, err := os.ReadFile("../../examples/" + m.Name())
		if err != nil {
			t.Fatal(err)
		}
		l := lexer.New(string(data))
		p := parser.New(l)
		prog := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Logf("%s: skipped (parse errors: %v)", m.Name(), p.Errors())
			continue
		}
		formatted := formatter.FormatSource(string(data))
		l2 := lexer.New(formatted)
		p2 := parser.New(l2)
		p2.ParseProgram()
		if len(p2.Errors()) > 0 {
			t.Errorf("%s: formatted output does not reparse: %v", m.Name(), p2.Errors())
		}
		_ = prog
	}
}

// TestCallSyntax documents which call forms Pipe accepts. Multi-argument calls
// use implicit space-separated arguments; f(a, b) and f() do not parse.
func TestCallSyntax(t *testing.T) {
	valid := []string{
		"f a",
		"f a b",
		"f (a)",
		"f(a + b)",
		"f (a + b)",
		"g (f a)",
		"print (fizzbuzz 1)",
		"print(\"hi\")",
		"x: f a b",
	}
	invalid := []string{
		"f()",
		"f(a, b)",
		"f (a, b)",
		"f a (b, c)",
	}
	for _, c := range valid {
		l := lexer.New(c)
		p := parser.New(l)
		p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Errorf("expected %q to parse, got: %v", c, p.Errors())
		}
	}
	for _, c := range invalid {
		l := lexer.New(c)
		p := parser.New(l)
		p.ParseProgram()
		if len(p.Errors()) == 0 {
			t.Errorf("expected %q to fail to parse", c)
		}
	}
}
