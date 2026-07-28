package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/harry/pipe/pkg/ast"
	"github.com/harry/pipe/pkg/cache"
	"github.com/harry/pipe/pkg/compiler"
	"github.com/harry/pipe/pkg/eval"
	"github.com/harry/pipe/pkg/formatter"
	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/object"
	"github.com/harry/pipe/pkg/parser"
	"github.com/harry/pipe/pkg/vm"
)

const version = "v0.5.0"

func main() {
	var (
		useVM     bool
		quietVM   bool
		showAST   bool
		doFmt     bool
		doBench   bool
		doTest    bool
		filePath  string
		scriptArgs []string
	)

	foundFile := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-vm":
			useVM = true
		case "-q":
			quietVM = true
		case "-ast":
			showAST = true
		case "-fmt":
			doFmt = true
		case "-bench":
			doBench = true
		case "-test":
			doTest = true
		case "-h", "--help":
			printHelp()
			return
		default:
			if !strings.HasPrefix(arg, "-") && !foundFile {
				filePath = arg
				foundFile = true
			} else if foundFile {
				scriptArgs = append(scriptArgs, arg)
			}
		}
	}

	if doBench {
		runBenchmark()
		return
	}

	if doTest {
		runTests(useVM)
		return
	}

	if filePath == "" {
		startREPL(useVM)
		return
	}

	if doFmt {
		if err := formatter.Format(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "pipe fmt: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Formatted %s\n", filePath)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipe: %s: %s\n", filePath, err)
		os.Exit(1)
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Fprintln(os.Stderr, "Parse-Fehler:")
		for _, err := range p.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
		os.Exit(1)
	}

	if showAST {
		fmt.Println(ASTString(program))
		return
	}

	if useVM {
		runVM(program, quietVM, filePath)
	} else {
		runEval(program, scriptArgs, filePath)
	}
}

func printHelp() {
	fmt.Println(`Pipe ` + version + ` — Minimalistische Pipeline-Skriptsprache

Verwendung:
  pipe [flags] <datei.pipe>    Datei ausführen
  pipe                          REPL starten

Flags:
  -vm           Bytecode-VM statt Tree-Walker verwenden
  -q            Im VM-Modus: Bytecode-Ausgabe unterdrücken
  -ast          Nur AST ausgeben, nicht ausführen
  -fmt          Datei formatieren (Einrückung, Whitespace)
  -test         Alle *.pipe und *_test.pipe im aktuellen Verzeichnis testen
  -bench        Benchmarks ausführen (Tree-Walker vs VM)
  -h, --help    Diese Hilfe anzeigen

Beispiele:
  pipe examples/hello.pipe
  pipe -vm examples/fib.pipe
  pipe -vm -q examples/fizzbuzz.pipe
  pipe -ast examples/pipeline.pipe
  pipe -test
  pipe -bench`)
}

func runEval(program *ast.Program, scriptArgs []string, filePath string) {
	env := object.NewEnvironment()

	argObjs := make([]object.Object, len(scriptArgs))
	for i, a := range scriptArgs {
		argObjs[i] = &object.String{Value: a}
	}
	env.Set("args", &object.List{Elements: argObjs})

	ctx := eval.NewEvalContext(filePath)
	result := ctx.Eval(program, env)
	if result != nil && result.Type() == object.ERROR {
		fmt.Fprintf(os.Stderr, "Laufzeit-Fehler: %s\n", result.Inspect())
		os.Exit(1)
	}
}

func runVM(program *ast.Program, quiet bool, filePath string) {
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		fmt.Fprintf(os.Stderr, "Compiler-Fehler: %s\n", err)
		os.Exit(1)
	}

	bc := comp.Bytecode()

	// Write cache if file path known
	if filePath != "" {
		if err := cache.WriteCache(filePath+"c", bc); err == nil {
			_ = err
		}
	}

	if !quiet {
		fmt.Fprintln(os.Stderr, "--- Bytecode ---")
		fmt.Fprint(os.Stderr, bc.Instructions.String())
	}

	start := time.Now()
	machine := vm.New(bc)
	if err := machine.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "VM-Fehler: %s\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	if !quiet {
		fmt.Fprintf(os.Stderr, "--- VM: %v ---\n", elapsed)
	}
}

func runVMWithCache(filePath string, quiet bool) {
	bc, fromCache, err := cache.LoadOrCompile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipe: %s\n", err)
		os.Exit(1)
	}

	if fromCache {
		fmt.Fprintln(os.Stderr, "  [cached]")
	}

	if !quiet {
		fmt.Fprintln(os.Stderr, "--- Bytecode ---")
		fmt.Fprint(os.Stderr, bc.Instructions.String())
	}

	start := time.Now()
	machine := vm.New(bc)
	if err := machine.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "VM-Fehler: %s\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	if !quiet {
		fmt.Fprintf(os.Stderr, "--- VM: %v ---\n", elapsed)
	}
}

func parseFile(path string) (*ast.Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("%s: %v", path, p.Errors())
	}
	return program, nil
}

func runTests(useVM bool) {
	matches, _ := filepath.Glob("*_test.pipe")
	if len(matches) == 0 {
		matches, _ = filepath.Glob("*.test.pipe")
	}
	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, "Keine Test-Dateien gefunden (*_test.pipe oder *.test.pipe)")
		os.Exit(1)
	}

	passed := 0
	failed := 0
	for _, path := range matches {
		program, err := parseFile(path)
		if err != nil {
			fmt.Printf("FAIL %s (parse error: %s)\n", path, err)
			failed++
			continue
		}

		if useVM {
			err = runTestVM(program)
		} else {
			err = runTestEval(program, path)
		}

		if err != nil {
			fmt.Printf("FAIL %s (%s)\n", path, err)
			failed++
		} else {
			fmt.Printf("PASS %s\n", path)
			passed++
		}
	}

	fmt.Printf("\n%d passed, %d failed, %d total\n", passed, failed, passed+failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func runTestEval(program *ast.Program, path string) error {
	ctx := eval.NewEvalContext(path)
	env := object.NewEnvironment()
	result := ctx.Eval(program, env)
	if result != nil && result.Type() == object.ERROR {
		return fmt.Errorf("%s", result.Inspect())
	}
	return nil
}

func runTestVM(program *ast.Program) error {
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		return err
	}
	bc := comp.Bytecode()
	machine := vm.New(bc)
	return machine.Run()
}

func runBenchmark() {
	fmt.Println("Pipe Benchmark: Tree-Walker vs Bytecode-VM")
	fmt.Println(strings.Repeat("-", 50))

	benchmarks := []struct {
		name string
		code string
	}{
		{"fib(20)", "fn fib n\n    match n\n        | 0 -> 0\n        | 1 -> 1\n        | _ -> fib(n - 1) + fib(n - 2)\n\nfib 20"},
		{"fizzbuzz 1-100", "fn fizz n\n    if n % 15 == 0\n        \"FizzBuzz\"\n    else if n % 3 == 0\n        \"Fizz\"\n    else if n % 5 == 0\n        \"Buzz\"\n    else\n        n\n\nx: 0\nwhile x < 100\n    x: x + 1\n    fizz x"},
		{"list sum 10000", "s: 0\ni: 0\nwhile i < 10000\n    s: s + i\n    i: i + 1\ns"},
	}

	for _, bm := range benchmarks {
		fmt.Printf("\n%s:\n", bm.name)

		// Parse once
		l := lexer.New(bm.code)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			fmt.Printf("  Parse error: %v\n", p.Errors())
			continue
		}

		// Tree-Walker
		evalStart := time.Now()
		for i := 0; i < 5; i++ {
			ctx := eval.NewEvalContext("<bench>")
			env := object.NewEnvironment()
			ctx.Eval(program, env)
		}
		evalTime := time.Since(evalStart) / 5
		fmt.Printf("  Tree-Walker: %v\n", evalTime)

		// VM
		comp := compiler.New()
		if err := comp.Compile(program); err != nil {
			fmt.Printf("  Compile error: %s\n", err)
			continue
		}
		bc := comp.Bytecode()
		vmStart := time.Now()
		for i := 0; i < 5; i++ {
			vm.New(bc).Run()
		}
		vmTime := time.Since(vmStart) / 5
		fmt.Printf("  VM:          %v\n", vmTime)

		if evalTime > 0 {
			fmt.Printf("  Speedup:     %.1fx\n", float64(evalTime)/float64(vmTime))
		}
	}
}

// ---- REPL ----

func startREPL(useVM bool) {
	fmt.Printf("Pipe %s — REPL\n", version)
	fmt.Println("Gib Pipe-Code ein. :quit oder Strg+D zum Beenden.")
	fmt.Println("Leerzeile zum Abschließen von mehrzeiligen Blöcken.")
	fmt.Println(":history — letzte Befehle | :!N — Befehl N wiederholen")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	env := object.NewEnvironment()
	history := make([]string, 0, 100)

	var lines []string
	needBlank := false

	for {
		if len(lines) > 0 {
			fmt.Print("...   ")
		} else {
			fmt.Print(">>> ")
		}

		if !scanner.Scan() {
			fmt.Println()
			return
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if len(lines) == 0 {
			switch trimmed {
			case ":quit", ":q":
				return
			case ":help", ":h":
				fmt.Println("  :quit, :q   — Beenden")
				fmt.Println("  :help, :h   — Hilfe")
				fmt.Println("  :clear, :c  — Eingabe zurücksetzen")
				fmt.Println("  :vm          — VM-Modus umschalten")
				fmt.Println("  :history     — Letzte Befehle anzeigen")
				fmt.Println("  :!N          — Befehl N wiederholen (z.B. :!3)")
				fmt.Println("  Strg+D       — Beenden")
				continue
			case ":clear", ":c":
				lines = nil
				needBlank = false
				continue
			case ":vm":
				useVM = !useVM
				if useVM {
					fmt.Println("  VM-Modus: ein")
				} else {
					fmt.Println("  VM-Modus: aus")
				}
				continue
			case ":history":
				if len(history) == 0 {
					fmt.Println("  (keine History)")
				} else {
					for i, cmd := range history {
						fmt.Printf("  %d: %s\n", i+1, cmd)
					}
				}
				continue
			}
			if strings.HasPrefix(trimmed, ":!") {
				numStr := strings.TrimPrefix(trimmed, ":!")
				num, err := strconv.Atoi(numStr)
				if err != nil || num < 1 || num > len(history) {
					fmt.Fprintf(os.Stderr, "  Ungültige Nummer. 1-%d\n", len(history))
				} else {
					replayCmd := history[num-1]
					fmt.Printf("  → %s\n", replayCmd)
					lines = append(lines, replayCmd)
					needBlank = false
					if !isMultiLineStart(replayCmd) && tryParse(replayCmd) {
						executeREPL(lines, env, useVM)
						lines = nil
					} else {
						needBlank = true
					}
				}
				continue
			}
		}

		if trimmed == "" {
			if len(lines) > 0 {
				cmd := strings.Join(lines, "; ")
				history = append(history, cmd)
				if len(history) > 100 {
					history = history[1:]
				}
				executeREPL(lines, env, useVM)
				lines = nil
				needBlank = false
			}
			continue
		}

		lines = append(lines, line)

		if !needBlank && len(lines) == 1 {
			if isMultiLineStart(trimmed) {
				needBlank = true
				continue
			}
			// Try to execute as single-line — parse only once
			executeREPL(lines, env, useVM)
			history = append(history, trimmed)
			if len(history) > 100 {
				history = history[1:]
			}
			lines = nil
			// If execution failed due to parse error, don't multi-line
			// If it was valid, we're done
		}
	}
}

func isMultiLineStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "fn ") {
		return true
	}
	if strings.HasPrefix(trimmed, "if ") {
		return true
	}
	if strings.HasPrefix(trimmed, "match ") {
		return true
	}
	if strings.HasPrefix(trimmed, "while ") {
		return true
	}
	if strings.HasPrefix(trimmed, "for ") {
		return true
	}
	if strings.HasPrefix(trimmed, "try") && len(trimmed) <= 3 {
		return true
	}
	if strings.HasPrefix(trimmed, "defer ") {
		return true
	}
	return false
}

func tryParse(input string) bool {
	l := lexer.New(input)
	p := parser.New(l)
	p.ParseProgram()
	return len(p.Errors()) == 0
}

func executeREPL(lines []string, env *object.Environment, useVM bool) {
	input := strings.Join(lines, "\n")
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Fprintln(os.Stderr, "Parse-Fehler:")
		for _, err := range p.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
		return
	}

	if useVM {
		replRunVM(program)
	} else {
		replRunEval(program, env)
	}
}

func replRunEval(program *ast.Program, env *object.Environment) {
	ctx := eval.NewEvalContext("<repl>")
	result := ctx.Eval(program, env)
	if result != nil {
		if result.Type() == object.ERROR {
			fmt.Fprintf(os.Stderr, "Fehler: %s\n", result.Inspect())
		} else if result.Type() != object.NIL {
			if result.Type() != object.FUNCTION && result.Type() != object.COMPILED_FUNCTION {
				fmt.Println(result.Inspect())
			}
		}
	}
}

func replRunVM(program *ast.Program) {
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		fmt.Fprintf(os.Stderr, "Compiler-Fehler: %s\n", err)
		return
	}
	bc := comp.Bytecode()
	machine := vm.New(bc)
	if err := machine.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "VM-Fehler: %s\n", err)
		return
	}
	result := machine.LastPoppedStackElem()
	if result != nil && result.Type() != object.NIL {
		if result.Type() != object.FUNCTION && result.Type() != object.COMPILED_FUNCTION {
			fmt.Println(result.Inspect())
		}
	}
}

func ASTString(node ast.Node) string {
	var out strings.Builder
	astToString(&out, node, 0)
	return out.String()
}

func astToString(out *strings.Builder, node ast.Node, depth int) {
	indent := strings.Repeat("  ", depth)

	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			astToString(out, stmt, depth)
		}

	case *ast.ExpressionStatement:
		out.WriteString(indent + "Expr:\n")
		astToString(out, n.Expression, depth+1)

	case *ast.VarStatement:
		out.WriteString(fmt.Sprintf("%sVar: %s =\n", indent, n.Name.Value))
		astToString(out, n.Value, depth+1)

	case *ast.FnLiteral:
		out.WriteString(fmt.Sprintf("%sFnLiteral:\n", indent))
		out.WriteString(indent + "  Params:\n")
		for _, p := range n.Parameters {
			out.WriteString(fmt.Sprintf("%s    %s\n", indent, p.Value))
		}
		out.WriteString(indent + "  Body:\n")
		astToString(out, n.Body, depth+2)

	case *ast.FnStatement:
		params := []string{}
		for _, p := range n.Parameters {
			params = append(params, p.Value)
		}
		out.WriteString(fmt.Sprintf("%sFn: %s(%s)\n", indent, n.Name.Value, strings.Join(params, ", ")))
		out.WriteString(indent + "Body:\n")
		astToString(out, n.Body, depth+1)

	case *ast.BlockStatement:
		for _, stmt := range n.Statements {
			astToString(out, stmt, depth)
		}

	case *ast.IfExpression:
		out.WriteString(fmt.Sprintf("%sIf:\n", indent))
		out.WriteString(indent + "  Cond:\n")
		astToString(out, n.Condition, depth+2)
		out.WriteString(indent + "  Then:\n")
		astToString(out, n.Consequence, depth+2)
		if n.Alternative != nil {
			out.WriteString(indent + "  Else:\n")
			astToString(out, n.Alternative, depth+2)
		}

	case *ast.MatchExpression:
		out.WriteString(fmt.Sprintf("%sMatch:\n", indent))
		out.WriteString(indent + "  Value:\n")
		astToString(out, n.Value, depth+2)
		for _, c := range n.Cases {
			out.WriteString(fmt.Sprintf("%s  | Pattern:\n", indent))
			astToString(out, c.Pattern, depth+3)
			out.WriteString(fmt.Sprintf("%s    -> Body:\n", indent))
			astToString(out, c.Body, depth+3)
		}

	case *ast.PipelineExpression:
		out.WriteString(fmt.Sprintf("%sPipeline:\n", indent))
		astToString(out, n.Left, depth+1)
		out.WriteString(indent + "  >\n")
		astToString(out, n.Right, depth+1)

	case *ast.InfixExpression:
		out.WriteString(fmt.Sprintf("%sInfix: %s\n", indent, n.Operator))
		astToString(out, n.Left, depth+1)
		astToString(out, n.Right, depth+1)

	case *ast.PrefixExpression:
		out.WriteString(fmt.Sprintf("%sPrefix: %s\n", indent, n.Operator))
		astToString(out, n.Right, depth+1)

	case *ast.CallExpression:
		out.WriteString(fmt.Sprintf("%sCall:\n", indent))
		astToString(out, n.Function, depth+1)
		for _, arg := range n.Arguments {
			out.WriteString(indent + "  Arg:\n")
			astToString(out, arg, depth+2)
		}

	case *ast.DotExpression:
		out.WriteString(fmt.Sprintf("%sDot: .%s\n", indent, n.Field))
		astToString(out, n.Left, depth+1)

	case *ast.IntegerLiteral:
		out.WriteString(fmt.Sprintf("%sInt: %d\n", indent, n.Value))
	case *ast.FloatLiteral:
		out.WriteString(fmt.Sprintf("%sFloat: %g\n", indent, n.Value))
	case *ast.StringLiteral:
		out.WriteString(fmt.Sprintf("%sString: %q\n", indent, n.Value))
	case *ast.BooleanLiteral:
		out.WriteString(fmt.Sprintf("%sBool: %t\n", indent, n.Value))
	case *ast.NilLiteral:
		out.WriteString(fmt.Sprintf("%sNil\n", indent))
	case *ast.Identifier:
		out.WriteString(fmt.Sprintf("%sIdent: %s\n", indent, n.Value))
	case *ast.ListLiteral:
		out.WriteString(fmt.Sprintf("%sList [%d elements]\n", indent, len(n.Elements)))
	case *ast.MapLiteral:
		out.WriteString(fmt.Sprintf("%sMap {%d pairs}\n", indent, len(n.Pairs)))
	}
}
