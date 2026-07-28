package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/harry/pulse/pkg/ast"
	"github.com/harry/pulse/pkg/compiler"
	"github.com/harry/pulse/pkg/eval"
	"github.com/harry/pulse/pkg/lexer"
	"github.com/harry/pulse/pkg/object"
	"github.com/harry/pulse/pkg/parser"
	"github.com/harry/pulse/pkg/vm"
)

const version = "v0.3.0"

func main() {
	var (
		useVM    bool
		quietVM  bool
		showAST  bool
		filePath string
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
			showAST = false
			// -fmt handled below
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

	if filePath == "" {
		startREPL(useVM)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pulse: %s\n", err)
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
		runVM(program, quietVM)
	} else {
		runEval(program, scriptArgs, filePath)
	}
}

func printHelp() {
	fmt.Println(`Pulse ` + version + ` — Minimalistische Pipeline-Skriptsprache

Verwendung:
  pulse [flags] <datei.pulse>    Datei ausführen
  pulse                          REPL starten

Flags:
  -vm           Bytecode-VM statt Tree-Walker verwenden
  -q            Im VM-Modus: Bytecode-Ausgabe unterdrücken
  -ast          Nur AST ausgeben, nicht ausführen
  -h, --help    Diese Hilfe anzeigen

Beispiele:
  pulse examples/hello.pulse
  pulse -vm examples/fib.pulse
  pulse -vm -q examples/fizzbuzz.pulse
  pulse -ast examples/pipeline.pulse`)
}

func runEval(program *ast.Program, scriptArgs []string, filePath string) {
	env := object.NewEnvironment()

	// Set global args variable
	argObjs := make([]object.Object, len(scriptArgs))
	for i, a := range scriptArgs {
		argObjs[i] = &object.String{Value: a}
	}
	env.Set("args", &object.List{Elements: argObjs})

	eval.SourceFile = filePath
	result := eval.Eval(program, env)
	if result != nil && result.Type() == object.ERROR {
		fmt.Fprintf(os.Stderr, "Laufzeit-Fehler: %s\n", result.Inspect())
		os.Exit(1)
	}
}

func runVM(program *ast.Program, quiet bool) {
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		fmt.Fprintf(os.Stderr, "Compiler-Fehler: %s\n", err)
		os.Exit(1)
	}

	bc := comp.Bytecode()

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

// ---- REPL ----

func startREPL(useVM bool) {
	fmt.Printf("Pulse %s — REPL\n", version)
	fmt.Println("Gib Pulse-Code ein. :quit oder Strg+D zum Beenden.")
	fmt.Println("Leerzeile zum Abschließen von mehrzeiligen Blöcken.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	env := object.NewEnvironment()

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
				fmt.Println("  :help, :h   — Diese Hilfe")
				fmt.Println("  :clear, :c  — Eingabe zurücksetzen")
				fmt.Println("  :vm          — VM-Modus umschalten")
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
			}
		}

		if trimmed == "" {
			if len(lines) > 0 {
				executeREPL(lines, env, useVM)
				lines = nil
				needBlank = false
			}
			continue
		}

		lines = append(lines, line)

		// Auto-execute single-line inputs that parse successfully and aren't multi-line starters
		if !needBlank && len(lines) == 1 {
			if isMultiLineStart(trimmed) {
				needBlank = true
				continue
			}
			// Try to parse and execute if successful
			if tryParse(strings.Join(lines, "\n")) {
				executeREPL(lines, env, useVM)
				lines = nil
				needBlank = false
			} else {
				needBlank = true
			}
		}
	}
}

func isMultiLineStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "fn ") {
		// fn at the start of a line is always multi-line
		return true
	}
	if strings.HasPrefix(trimmed, "if ") {
		return true
	}
	if strings.HasPrefix(trimmed, "match ") {
		return true
	}
	// Variable def with trailing colon: name:
	if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
		// e.g., "name:" at end of line = multi-line value
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
	result := eval.Eval(program, env)
	if result != nil {
		if result.Type() == object.ERROR {
			fmt.Fprintf(os.Stderr, "Fehler: %s\n", result.Inspect())
		} else if result.Type() != object.NIL {
			// Don't show function definitions as values
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

// ---- AST Printer (für -ast Flag) ----

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
