package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/harry/pipe/pkg/ai"
	"github.com/harry/pipe/pkg/ast"
	"github.com/harry/pipe/pkg/build"
	"github.com/harry/pipe/pkg/cache"
	"github.com/harry/pipe/pkg/compiler"
	"github.com/harry/pipe/pkg/eval"
	"github.com/harry/pipe/pkg/formatter"
	"github.com/harry/pipe/pkg/lexer"
	"github.com/harry/pipe/pkg/object"
	"github.com/harry/pipe/pkg/parser"
	"github.com/harry/pipe/pkg/vm"
)

const version = "v0.7.0"

func main() {
	// Self-extracting binary detection
	if embedded, ok := build.LoadEmbedded(os.Args[0]); ok {
		runEmbedded(embedded)
		return
	}

	var (
		useVM          bool
		quietVM        bool
		showAST        bool
		doFmt          bool
		fmtCheck       bool
		doBench        bool
		doTest         bool
		doBuild        bool
		doGet          bool
		doSearch       bool
		searchTerm     string
		sandbox        bool
		sandboxProfile string
		allowAI        bool
		aiProvider     string
		timeoutSec     int
		buildOut       string
		filePath       string
		scriptArgs     []string
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
		case "--check":
			doFmt = true
			fmtCheck = true
		case "-bench":
			doBench = true
		case "-test":
			doTest = true
		case "-build":
			doBuild = true
		case "-get":
			doGet = true
		case "-search":
			doSearch = true
		case "-h", "--help":
			printHelp()
			return
		case "--sandbox":
			sandbox = true
		case "--sandbox-profile":
			sandboxProfile = "-" // marker to read next arg
		case "--allow-ai":
			allowAI = true
		case "--ai-provider":
			aiProvider = "-" // marker to read next arg
		case "--timeout":
			timeoutSec = -1 // marker to read next arg
		default:
			if aiProvider == "-" {
				aiProvider = arg
				continue
			}
			if sandboxProfile == "-" {
				sandboxProfile = arg
				continue
			}
			if timeoutSec == -1 {
				// --timeout <seconds>
				if n, err := strconv.Atoi(arg); err == nil && n > 0 {
					timeoutSec = n
					continue
				}
			}
			if doBuild && !strings.HasPrefix(arg, "-") {
				if filePath == "" {
					filePath = arg
					foundFile = true
				} else if buildOut == "" {
					buildOut = arg
				}
			} else if doSearch && !strings.HasPrefix(arg, "-") {
				searchTerm = arg
			} else if !strings.HasPrefix(arg, "-") && !foundFile {
				filePath = arg
				foundFile = true
			} else if foundFile {
				scriptArgs = append(scriptArgs, arg)
			}
		}
	}

	if doBuild {
		if filePath == "" {
			fmt.Fprintln(os.Stderr, "pipe: -build requires a .pipe file")
			os.Exit(1)
		}
		outPath := buildOut
		if outPath == "" {
			outPath = strings.TrimSuffix(filePath, ".pipe")
		}
		if err := build.Build(filePath, outPath); err != nil {
			fmt.Fprintf(os.Stderr, "pipe build: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Built %s -> %s\n", filePath, outPath)
		return
	}

	if doGet {
		if filePath == "" {
			fmt.Fprintln(os.Stderr, "pipe: -get requires a URL or module name")
			fmt.Fprintln(os.Stderr, "  Example: pipe -get https://raw.githubusercontent.com/.../module.pipe")
			fmt.Fprintln(os.Stderr, "  Example: pipe -get log-analyzer")
			fmt.Fprintln(os.Stderr, "  Example: pipe -get log-analyzer@1.0.0")
			os.Exit(1)
		}

		target := filePath
		isName := false
		if !strings.HasPrefix(filePath, "http://") && !strings.HasPrefix(filePath, "https://") {
			url, err := resolveModuleURL(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pipe get: %s\n", err)
				fmt.Fprintln(os.Stderr, "  Use pipe -search to find available modules.")
				os.Exit(1)
			}
			target = url
			isName = true
		}

		modDir := object.ModuleCacheDir()
		_, content, err := object.ResolveImport(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe get: %s\n", err)
			os.Exit(1)
		}

		// Also save under the short name so import "name" works
		if isName {
			shortPath := filepath.Join(modDir, filePath+".pipe")
			os.WriteFile(shortPath, []byte(content), 0644)
		}

		fmt.Printf("✓ Installed %s → %s\n", filePath, modDir)
		return
	}

	if doSearch {
		runSearch(searchTerm)
		return
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
		if filePath == "" {
			fmt.Fprintln(os.Stderr, "pipe fmt: requires a .pipe file or directory")
			os.Exit(1)
		}
		changed, err := formatPath(filePath, fmtCheck)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe fmt: %s\n", err)
			os.Exit(1)
		}
		if fmtCheck {
			if changed > 0 {
				fmt.Printf("%d file(s) need formatting\n", changed)
				os.Exit(1)
			}
			fmt.Println("All files formatted")
		} else {
			if changed > 0 {
				fmt.Printf("Formatted %d file(s)\n", changed)
			}
		}
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
		fmt.Fprintln(os.Stderr, "Parse errors:")
		for _, err := range p.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
		os.Exit(1)
	}

	if showAST {
		fmt.Println(ASTString(program))
		return
	}

	if sandbox {
		object.SetSandbox(true)
	}
	if sandboxProfile != "" && sandboxProfile != "-" {
		prof, err := object.GetProfile(sandboxProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe: %s\n", err)
			os.Exit(1)
		}
		object.ActiveProfile = prof
	}
	if allowAI {
		object.SetSandboxAllowAI(true)
	}
	if aiProvider != "" {
		ai.SetProvider(aiProvider)
	}
	if timeoutSec > 0 {
		go func() {
			time.Sleep(time.Duration(timeoutSec) * time.Second)
			fmt.Fprintf(os.Stderr, "\nTIMEOUT: execution exceeded %ds\n", timeoutSec)
			os.Exit(124)
		}()
	}

	if useVM {
		runVM(program, quietVM, filePath)
	} else {
		runEval(program, scriptArgs, filePath)
	}
}

func runSearch(term string) {
	reg, err := object.FetchRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipe search: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nPipe Modules%s\n\n", map[bool]string{true: " (filter: \"" + term + "\")"}[term != ""])

	found := 0
	for name, mod := range reg.Modules {
		if term != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(term)) && !strings.Contains(strings.ToLower(mod.Description), strings.ToLower(term)) {
			continue
		}
		found++
		fmt.Printf("  %s", name)
		if mod.Latest != "" {
			fmt.Printf("  (latest: %s)", mod.Latest)
		}
		fmt.Printf("\n    %s\n", mod.Description)
		fmt.Printf("    Functions: %s\n", strings.Join(mod.Functions, ", "))
		if len(mod.Versions) > 1 {
			vers := []string{}
			for v := range mod.Versions {
				vers = append(vers, v)
			}
			fmt.Printf("    Versions: %s\n", strings.Join(vers, ", "))
		}
		fmt.Println()
	}

	if found == 0 {
		fmt.Printf("  No modules found matching \"%s\"\n", term)
	} else {
		fmt.Printf("  %d module(s) found. Install with: pipe -get <name>\n\n", found)
	}
}

func resolveModuleURL(name string) (string, error) {
	modName, version := object.ParseModuleSpec(name)
	return object.ResolveModuleURL(modName, version)
}

func formatPath(path string, checkOnly bool) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return formatDir(path, checkOnly)
	}
	if checkOnly {
		same, err := formatter.FormatCheck(path)
		if err != nil {
			return 0, err
		}
		if same {
			return 0, nil
		}
		fmt.Println(path)
		return 1, nil
	}
	if err := formatter.Format(path); err != nil {
		return 0, err
	}
	return 1, nil
}

func formatDir(dir string, checkOnly bool) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".pipe") {
			return nil
		}
		if checkOnly {
			same, err := formatter.FormatCheck(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "pipe fmt: %s: %s\n", path, err)
				return nil
			}
			if !same {
				fmt.Println(path)
				count++
			}
			return nil
		}
		if err := formatter.Format(path); err != nil {
			fmt.Fprintf(os.Stderr, "pipe fmt: %s: %s\n", path, err)
			return nil
		}
		count++
		return nil
	})
	return count, err
}

func printHelp() {
	fmt.Println(`Pipe (SPR) ` + version + ` — Semantic Pipeline Runtime

Usage:
  pipe [flags] <file.pipe>    Execute file
  pipe                         Start REPL

Flags:
  -vm           Use bytecode VM instead of tree-walker (~7x faster)
  -q            VM mode: suppress bytecode output
  -ast          Only print AST, don't execute
  -fmt          Format file or directory (indentation, whitespace)
  --check       Check formatting without writing (requires -fmt)
  -test         Run all test files in current directory
  -bench        Run benchmarks (tree-walker vs VM)
  --sandbox              Restrict dangerous builtins (exec, tcp, http, ai, fs-write)
  --sandbox-profile <name>  Use a predefined sandbox profile (strict, networked, etc.)
  --allow-ai             In sandbox: re-enable AI builtins
  --timeout N            Kill execution after N seconds
  --ai-provider openai|anthropic|deepseek|ollama
  -get <url>    Download a module into ~/.pipe/modules/
  -search [term] Search available modules
  -h, --help    Show this help

Examples:
  pipe examples/hello.pipe
  pipe -vm -q examples/fib.pipe
  pipe -search                  # List all modules
  pipe -search log              # Search for "log" modules
  pipe -get log-analyzer        # Install a module
  pipe --sandbox script.pipe
  pipe -build my.pipe -o my_prog`)
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
		fmt.Fprintf(os.Stderr, "Runtime error: %s\n", result.Inspect())
		os.Exit(1)
	}
}

func runVM(program *ast.Program, quiet bool, filePath string) {
	comp := compiler.NewWithFile(filePath)
	if err := comp.Compile(program); err != nil {
		fmt.Fprintf(os.Stderr, "Compiler error: %s\n", err)
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
		fmt.Fprintf(os.Stderr, "VM error: %s\n", err)
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
		fmt.Fprintf(os.Stderr, "VM error: %s\n", err)
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
		fmt.Fprintln(os.Stderr, "No test files found (*_test.pipe or *.test.pipe)")
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
	fmt.Println("Enter Pipe code. :quit or Ctrl+D to exit.")
	fmt.Println("Blank line to complete multi-line blocks.")
	fmt.Println(":history — last commands | :!N — repeat command N")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	env := object.NewEnvironment()
	history := loadHistory()

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
			saveHistory(history)
			return
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if len(lines) == 0 {
			switch trimmed {
			case ":quit", ":q":
				saveHistory(history)
				return
			case ":help", ":h":
				fmt.Println("  :quit, :q   — Exit")
				fmt.Println("  :help, :h   — Help")
				fmt.Println("  :clear, :c  — Reset input")
				fmt.Println("  :vm          — Toggle VM mode")
				fmt.Println("  :history     — Show last commands")
				fmt.Println("  :!N          — Repeat command N (e.g. :!3)")
				fmt.Println("  Ctrl+D       — Exit")
				continue
			case ":clear", ":c":
				lines = nil
				needBlank = false
				continue
			case ":vm":
				useVM = !useVM
				if useVM {
					fmt.Println("  VM mode: on")
				} else {
					fmt.Println("  VM mode: off")
				}
				continue
			case ":history":
				if len(history) == 0 {
					fmt.Println("  (no history)")
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
					fmt.Fprintf(os.Stderr, "  Invalid number. 1-%d\n", len(history))
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
	if trimmed == "try" || strings.HasPrefix(trimmed, "try\n") {
		return true
	}
	if strings.HasPrefix(trimmed, "defer ") {
		return true
	}
	if strings.HasPrefix(trimmed, "export ") {
		return true
	}
	if strings.HasPrefix(trimmed, "enum ") {
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

func histFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pipe_history")
}

func loadHistory() []string {
	path := histFile()
	if path == "" {
		return make([]string, 0, 100)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return make([]string, 0, 100)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	history := make([]string, 0, max(100, len(lines)+20))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			history = append(history, trimmed)
		}
	}
	return history
}

func saveHistory(history []string) {
	path := histFile()
	if path == "" || len(history) == 0 {
		return
	}
	data := strings.Join(history, "\n") + "\n"
	os.WriteFile(path, []byte(data), 0644)
}

func executeREPL(lines []string, env *object.Environment, useVM bool) {
	input := strings.Join(lines, "\n")
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Fprintln(os.Stderr, "Parse errors:")
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
			fmt.Fprintf(os.Stderr, "Error: %s\n", result.Inspect())
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
		fmt.Fprintf(os.Stderr, "Compiler error: %s\n", err)
		return
	}
	bc := comp.Bytecode()
	machine := vm.New(bc)
	if err := machine.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "VM error: %s\n", err)
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
		if n.Parallel {
			out.WriteString(indent + "  >>\n")
		} else {
			out.WriteString(indent + "  >\n")
		}
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

func runEmbedded(src []byte) {
	l := lexer.New(string(src))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		fmt.Fprintln(os.Stderr, "Parse errors in embedded binary:")
		for _, err := range p.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
		os.Exit(1)
	}

	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		fmt.Fprintf(os.Stderr, "Compiler error: %s\n", err)
		os.Exit(1)
	}

	bc := comp.Bytecode()
	machine := vm.New(bc)
	if err := machine.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
