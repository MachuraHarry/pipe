package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/analysis"
	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/build"
	"github.com/MachuraHarry/pipe/pkg/cache"
	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/docgen"
	"github.com/MachuraHarry/pipe/pkg/eval"
	"github.com/MachuraHarry/pipe/pkg/formatter"
	"github.com/MachuraHarry/pipe/pkg/gen"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/module"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
	"github.com/MachuraHarry/pipe/pkg/util"
	"github.com/MachuraHarry/pipe/pkg/vm"
)

var version = "v0.9.4.0"

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
		doGen          bool
		genCount       int
		genRun         bool
		genCheck       bool
		doGet          bool
		doSearch       bool
		doInit         bool
		doCheck        bool
		doGenRegistry  bool
		doInstall      bool
		doPublish      bool
		doDoc          bool
		docBuiltins    bool
		searchTerm     string
		sandbox        bool
		sandboxProfile string
		allowAI        bool
		aiProvider     string
		timeoutSec     int
		buildOut       string
		useUPX         bool
		embedFiles     []string
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
		case "-upx":
			useUPX = true
		case "-gen":
			doGen = true
		case "-run":
			genRun = true
		case "-check":
			genCheck = true
		case "-n":
			genCount = -1
		case "-get":
			doGet = true
		case "-search":
			doSearch = true
		case "-init":
			doInit = true
		case "-validate":
			doCheck = true
		case "-gen-registry":
			doGenRegistry = true
		case "-install":
			doInstall = true
		case "-publish":
			doPublish = true
		case "-doc":
			doDoc = true
		case "--builtins":
			docBuiltins = true
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
		case "--embed-file":
			embedFiles = append(embedFiles, "-") // marker to read next arg
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
			if genCount == -1 {
				if n, err := strconv.Atoi(arg); err == nil && n > 0 {
					genCount = n
					continue
				}
			}
			if len(embedFiles) > 0 && embedFiles[len(embedFiles)-1] == "-" {
				embedFiles[len(embedFiles)-1] = arg
				continue
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
		if len(embedFiles) > 0 {
			efs := make([]build.EmbedFile, len(embedFiles))
			for i, fp := range embedFiles {
				data, err := os.ReadFile(fp)
				if err != nil {
					fmt.Fprintf(os.Stderr, "pipe: embed-file '%s': %s\n", fp, err)
					os.Exit(1)
				}
				efs[i] = build.EmbedFile{Path: fp, Data: data}
			}
			if err := build.BuildWithFiles(filePath, outPath, efs); err != nil {
				fmt.Fprintf(os.Stderr, "pipe build: %s\n", err)
				os.Exit(1)
			}
		} else {
			if err := build.Build(filePath, outPath); err != nil {
				fmt.Fprintf(os.Stderr, "pipe build: %s\n", err)
				os.Exit(1)
			}
		}
		if useUPX {
			if upxPath, err := exec.LookPath("upx"); err == nil {
				cmd := exec.Command(upxPath, "-q", outPath, "-o", outPath)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "UPX failed: %s\n", err)
				}
			} else {
				fmt.Fprintln(os.Stderr, "UPX not found in PATH — install with: apt install upx-ucl")
				fmt.Fprintln(os.Stderr, "Binary built without compression")
			}
		}
		fmt.Printf("Built %s -> %s\n", filePath, outPath)
		return
	}

	if doGen {
		n := genCount
		if n <= 0 {
			n = 1
		}
		allOK := true
		for i := 0; i < n; i++ {
			opts := gen.DefaultOptions()
			opts.Seed = time.Now().UnixNano() + int64(i)

			prog, src, err := gen.GenerateValid(opts)
			if err != nil {
				allOK = false
				fmt.Fprintf(os.Stderr, "gen: %s\n", err)
				continue
			}
			if !strings.HasSuffix(src, "\n") {
				src += "\n"
			}

			fmt.Print(src)

			if genRun {
				if err := runGenProgram(prog); err != nil {
					allOK = false
					fmt.Fprintf(os.Stderr, "run: %s\n", err)
				}
			}

			if i < n-1 {
				fmt.Println()
			}
		}
		if genCheck && !allOK {
			os.Exit(1)
		}
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

	if doInit {
		dir := filePath
		if dir == "" {
			fmt.Fprintln(os.Stderr, "pipe: -init requires a module name")
			fmt.Fprintln(os.Stderr, "  Example: pipe -init my-module")
			os.Exit(1)
		}
		name := filepath.Base(dir)
		if err := module.InitModule(dir, name); err != nil {
			fmt.Fprintf(os.Stderr, "pipe init: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Initialized module %q:\n", name)
		fmt.Printf("  %s/pipe.json\n", dir)
		fmt.Printf("  %s/module.pipe\n", dir)
		fmt.Printf("  %s/README.md\n", dir)
		return
	}

	if doCheck {
		dir := filePath
		if dir == "" {
			dir = "."
		}
		m, err := module.Parse(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe validate: %s\n", err)
			os.Exit(1)
		}
		if err := m.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "pipe validate: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("OK %s v%s\n", m.Name, m.Version)
		if len(m.Exports) > 0 {
			fmt.Printf("  Exports: %s\n", strings.Join(m.Exports, ", "))
		}
		if len(m.Dependencies) > 0 {
			fmt.Println("  Dependencies:")
			for dep, ver := range m.Dependencies {
				fmt.Printf("    %s: %s\n", dep, ver)
			}
		}
		return
	}

	if doGenRegistry {
		dir := filePath
		if dir == "" {
			dir = "."
		}
		baseURL := ""
		if len(scriptArgs) > 0 {
			baseURL = scriptArgs[0]
		}
		if err := module.GenerateRegistry(dir, baseURL); err != nil {
			fmt.Fprintf(os.Stderr, "pipe gen-registry: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated registry.json from pipe.json files in %s\n", dir)
		report, _ := module.GenerateRegistryReport(dir)
		for _, line := range report {
			fmt.Println(line)
		}
		return
	}

	if doInstall {
		dir := filePath
		if dir == "" {
			dir = "."
		}
		if err := module.Install(dir); err != nil {
			fmt.Fprintf(os.Stderr, "pipe install: %s\n", err)
			os.Exit(1)
		}
		return
	}

	if doPublish {
		dir := filePath
		if err := module.Publish(dir); err != nil {
			fmt.Fprintf(os.Stderr, "pipe publish: %s\n", err)
			os.Exit(1)
		}
		return
	}

	if doDoc {
		if docBuiltins {
			fmt.Print(docgen.MarkdownForBuiltins())
			return
		}
		target := filePath
		if target == "" {
			target = "."
		}
		md, err := docgen.MarkdownForPath(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe doc: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(md)
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
			fmt.Fprintln(os.Stderr, "  "+util.FormatErrorWithSnippet(string(data), err))
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
		object.ActiveProfile.Store(prof)
		object.SetSandboxStartLocked()
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
		runVM(program, quietVM, filePath, scriptArgs, string(data))
	} else {
		runEval(program, scriptArgs, filePath, string(data))
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
  -vm           Use bytecode VM instead of tree-walker (measured 0.6x-55x speedup)
  -q            VM mode: suppress bytecode output
  -ast          Only print AST, don't execute
  -fmt          Format file or directory (indentation, whitespace)
  --check       Check formatting without writing (requires -fmt)
  -test         Run all test files in current directory
  -bench        Run benchmarks (tree-walker vs VM)
  -gen          Generate a random valid program
  -run          Execute the generated program (use with -gen)
  -check        Validate generated program, exit non-zero on failure (use with -gen)
  -n N          Generate N programs (use with -gen)
  --sandbox              Restrict dangerous builtins (exec, tcp, http, ai, fs-write)
  --sandbox-profile <name>  Use a predefined sandbox profile (strict, networked, etc.)
  --allow-ai             In sandbox: re-enable AI builtins
  --timeout N            Kill execution after N seconds
  --ai-provider openai|anthropic|deepseek|ollama
  -get <url>    Download a module into ~/.pipe/modules/
  -search [term] Search available modules
  -init <name>   Create a new module scaffold (pipe.json + module.pipe)
  -validate [dir] Check a module's pipe.json validity
  -install [dir] Install dependencies from pipe.json
  -publish [dir] Publish a module via pull request (requires gh CLI)
  -gen-registry [dir]  Generate registry.json from pipe.json files
  -doc [file|dir]  Generate Markdown documentation from --! docstrings
  --builtins       With -doc: generate the builtin reference
  -h, --help    Show this help

Examples:
  pipe examples/hello.pipe
  pipe -vm -q examples/fib.pipe
  pipe -init my-module            # Scaffold new module
  pipe -search                    # List all modules
  pipe -search log                # Search for "log" modules
  pipe -get log-analyzer          # Install a module
  pipe --sandbox script.pipe
  pipe -build my.pipe -o my_prog
  pipe -build my.pipe -o my_prog -upx  # UPX-compressed (~60% smaller)`)
}

// printErrorBlock writes a runtime/VM error to stderr, appending a source
// snippet whenever the error carries a resolvable position.
func printErrorBlock(prefix string, source string, err error) {
	msg := err.Error()
	line, col := 0, 0
	if e, ok := err.(*object.Error); ok {
		msg = e.Message
		line, col = e.Line, e.Col
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", prefix, msg)
	if line > 0 {
		if snip := util.Snippet(source, line, col); snip != "" {
			fmt.Fprintln(os.Stderr, snip)
		}
	}
}

// printWarnings reports LSP-style semantic warnings (unused variables) for a
// parsed program to stderr. Purely informational; never affects exit codes.
func printWarnings(source string, program *ast.Program) {
	if source == "" || program == nil {
		return
	}
	res := analysis.LintProgram(program)
	for _, d := range res.Diagnostics {
		if d.Severity != analysis.SeverityWarning {
			continue
		}
		fmt.Fprintf(os.Stderr, "warning: %s\n", d.Message)
		if snip := util.Snippet(source, d.Range.Start.Line, d.Range.Start.Col); snip != "" {
			fmt.Fprintln(os.Stderr, snip)
		}
	}
}

func runEval(program *ast.Program, scriptArgs []string, filePath, source string) {
	object.ScriptArgs = scriptArgs
	ai.ResetCostMetrics()

	printWarnings(source, program)

	env := object.NewEnvironment()

	ctx := eval.NewEvalContext(filePath)
	result := ctx.Eval(program, env)
	if result != nil && result.Type() == object.ERROR {
		printErrorBlock("Runtime error", source, result.(*object.Error))
		os.Exit(1)
	}

	printCostTrace()
}

func checkProgram(program *ast.Program) error {
	comp := compiler.New()
	return comp.Compile(program)
}

func runGenProgram(program *ast.Program) error {
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		return err
	}
	bc := comp.Bytecode()
	machine := vm.New(bc)
	return machine.Run()
}

func runVM(program *ast.Program, quiet bool, filePath string, scriptArgs []string, source string) {
	object.ScriptArgs = scriptArgs
	ai.ResetCostMetrics()

	printWarnings(source, program)

	var bc *compiler.Bytecode
	fromCache := false
	if filePath != "" {
		var err error
		bc, fromCache, err = cache.LoadOrCompile(filePath)
		if err != nil {
			printErrorBlock("Compiler error", source, err)
			os.Exit(1)
		}
	} else {
		comp := compiler.NewWithFile("")
		if err := comp.Compile(program); err != nil {
			printErrorBlock("Compiler error", source, err)
			os.Exit(1)
		}
		bc = comp.Bytecode()
	}

	if !quiet {
		if fromCache {
			fmt.Fprintln(os.Stderr, "--- Bytecode (cached) ---")
		} else {
			fmt.Fprintln(os.Stderr, "--- Bytecode ---")
		}
		fmt.Fprint(os.Stderr, bc.Instructions.String())
	}

	start := time.Now()
	machine := vm.New(bc)
	if err := machine.Run(); err != nil {
		printErrorBlock("VM error", source, err)
		os.Exit(1)
	}
	checkVMTopResult(machine, source)
	elapsed := time.Since(start)

	if !quiet {
		fmt.Fprintf(os.Stderr, "--- VM: %v ---\n", elapsed)
	}
	printCostTrace()
}

// checkVMTopResult mirrors the tree-walker's top-level error handling
// (runEval): a script whose final value is an Error object is reported as a
// runtime error, not silently accepted.
func checkVMTopResult(machine *vm.VM, source string) {
	result := machine.LastPoppedStackElem()
	if result != nil && result.Type() == object.ERROR {
		printErrorBlock("Runtime error", source, result.(*object.Error))
		os.Exit(1)
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
	if err := machine.Run(); err != nil {
		return err
	}
	if machine.TestFailed {
		return fmt.Errorf("some tests failed")
	}
	result := machine.LastPoppedStackElem()
	if result != nil && result.Type() == object.ERROR {
		return fmt.Errorf("%s", result.Inspect())
	}
	return nil
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
		{"string concat 20000", "s: \"\"\ni: 0\nwhile i < 20000\n    s: s ++ \"ab\"\n    i: i + 1\nlen s"},
		{"list push + sum 20000", "lst: []\ni: 0\nwhile i < 20000\n    lst: push lst i\n    i: i + 1\ntotal: 0\nfor x in lst\n    total: total + x\ntotal"},
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
	object.ScriptArgs = os.Args[1:]
	ai.ResetCostMetrics()

	if dir, err := build.ExtractFiles(os.Args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "pipe: extracting embedded files: %s\n", err)
		os.Exit(1)
	} else if dir != "" {
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintf(os.Stderr, "pipe: changing to embedded files dir: %s\n", err)
			os.Exit(1)
		}
	}

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
	printCostTrace()
}

func printCostTrace() {
	cost, tokens, calls, hits, misses := ai.GetCostMetrics()
	history := ai.GetCostHistory()
	if calls == 0 && hits == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "\n═══ Cost Trace ═══\n")
	fmt.Fprintf(os.Stderr, "Total cost:    $%.6f\n", cost)
	fmt.Fprintf(os.Stderr, "Total tokens:  %d\n", tokens)
	fmt.Fprintf(os.Stderr, "API calls:     %d\n", calls)
	fmt.Fprintf(os.Stderr, "Cache hits:    %d | misses: %d\n", hits, misses)
	for i, h := range history {
		cached := ""
		if h.Cached {
			cached = " [CACHE]"
		}
		fmt.Fprintf(os.Stderr, "  #%d %s/%s | %d tokens | $%.6f%s\n",
			i+1, h.Provider, h.Model, h.TotalTokens, h.CostUSD, cached)
	}
	if cost > 0 {
		fmt.Fprintf(os.Stderr, "══════════════════\n")
	}
}
