package parser

import (
	"testing"

	"github.com/MachuraHarry/pipe/pkg/lexer"
)

func FuzzParser(f *testing.F) {
	f.Add("42")
	f.Add("x: 10\ny: 20")
	f.Add("fn greet name\n    \"Hello, \" ++ name ++ \"!\"\n\ngreet \"World\"")
	f.Add("print (2 + 2)")
	f.Add("")
	f.Add("if true\n    print 1\nelse\n    print 2")
	f.Add("match 3\n    | 1 -> print 1\n    | _ -> print 0")
	f.Add("struct Point\n    x\n    y\n\np: Point 10 20\np.x")
	f.Add("fn fib n\n    if n <= 1\n        n\n    else\n        fib(n-1) + fib(n-2)\n\nfib 5")
	f.Add("\n\n\n")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parser panic on input %q: %v", input, r)
			}
		}()
		l := lexer.New(input)
		p := New(l)
		p.ParseProgram()
		// Even on parse errors, should not panic
	})
}
