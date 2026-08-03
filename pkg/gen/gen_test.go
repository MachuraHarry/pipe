package gen

import (
	"testing"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/formatter"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

func TestGenerateComprehensive(t *testing.T) {
	seeds := []int64{1, 2, 3, 7, 13, 42, 100, 999, 1234, 4242}
	for _, seed := range seeds {
		opts := DefaultOptions()
		opts.Seed = seed
		prog, src, err := GenerateValid(opts)
		if err != nil {
			t.Errorf("seed %d: %v", seed, err)
			continue
		}
		_ = prog
		_ = src
	}
}

func TestGenerateRoundtrip(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		opts := DefaultOptions()
		opts.Seed = seed
		prog := Generate(opts)
		src := formatter.FormatProgram(prog)

		l := lexer.New(src)
		p := parser.New(l)
		_ = p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Errorf("seed %d: formatted output does not reparse: %v\n---\n%s\n---", seed, p.Errors(), src)
			continue
		}
	}
}

func TestGenerateCompiles(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		opts := DefaultOptions()
		opts.Seed = seed
		prog := Generate(opts)

		comp := compiler.New()
		if err := comp.Compile(prog); err != nil {
			t.Errorf("seed %d: compile error: %v\nsource:\n%s\n---", seed, err, formatter.FormatProgram(prog))
			continue
		}
	}
}

func TestGenerateQuick(t *testing.T) {
	opts := DefaultOptions()
	opts.Seed = 42
	prog := Generate(opts)
	src := formatter.FormatProgram(prog)
	if len(src) < 10 {
		t.Errorf("expected non-trivial program, got %q", src)
	}
	t.Logf("seed 42:\n%s", src)
}
