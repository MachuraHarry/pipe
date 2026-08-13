package main

// stats reports the authoritative project statistics used throughout the
// README, website, and documentation so that doc claims can never drift from
// the code. Run with: go run ./scripts/stats
//
// It counts every builtin registered at import time (pkg/object exposes the
// full table after all init functions ran), then derives the secondary
// numbers (examples, tests, docs chapters) directly from the filesystem.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/object"
)

var sandboxNames = map[string]bool{
	"sandbox_profile": true, "set_sandbox": true,
	"with_sandbox": true, "sandbox_lock": true,
}

var fileNames = map[string]bool{
	"read_file": true, "write_file": true, "append_file": true,
	"file_open": true, "file_close": true, "file_read": true,
	"file_write": true, "file_truncate": true, "file_sync": true,
}

var aiNames = map[string]bool{
	"ask": true, "agent": true, "agent_ask": true, "agent_clear": true,
	"classify": true, "extract": true, "generate": true, "generate_json": true,
	"summarize": true, "translate": true, "embed": true, "embed_batch": true,
	"web_search": true, "wiki_search": true,
	"cosine_sim": true, "dot_product": true, "nearest": true, "try_ai_log": true,
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	total := len(object.Builtins)
	cat := map[string]int{"ai_": 0, "ai_all": 0, "mcp_": 0, "sandbox": 0, "file": 0, "stdlib": 0}
	for _, b := range object.Builtins {
		switch {
		case strings.HasPrefix(b.Name, "ai_") || aiNames[b.Name]:
			if strings.HasPrefix(b.Name, "ai_") {
				cat["ai_"]++
			}
			cat["ai_all"]++
		case strings.HasPrefix(b.Name, "mcp_"):
			cat["mcp_"]++
		case sandboxNames[b.Name]:
			cat["sandbox"]++
		case fileNames[b.Name]:
			cat["file"]++
		default:
			cat["stdlib"]++
		}
	}

	fmt.Println("== Builtins ==")
	fmt.Printf("total: %d\n", total)
	for _, key := range []string{"ai_", "ai_all", "mcp_", "sandbox", "file", "stdlib"} {
		fmt.Printf("  %-8s %d\n", key+":", cat[key])
	}
	fmt.Println()

	examples, err := filepath.Glob(filepath.Join(root, "examples", "*.pipe"))
	if err != nil {
		fatal(err)
	}
	fmt.Printf("examples: %d\n", len(examples))

	testFns := 0
	testPkgs := map[string]bool{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/website/") {
			return nil
		}
		testPkgs[filepath.Dir(path)] = true
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "func Test") {
				testFns++
			}
		}
		return nil
	})
	fmt.Printf("go tests: %d\n", testFns)
	fmt.Printf("test packages: %d\n", len(testPkgs))

	astNodes := 0
	data, err := os.ReadFile(filepath.Join(root, "pkg", "ast", "ast.go"))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "type ") && strings.HasSuffix(line, " struct {") ||
				strings.HasPrefix(line, "type ") && strings.HasSuffix(line, " struct{}") {
				if strings.HasPrefix(line, "type Position") {
					continue
				}
				astNodes++
			}
		}
	}
	fmt.Printf("AST node types: %d\n", astNodes)

	for _, lang := range []string{"en", "de"} {
		dir := filepath.Join(root, "docs", lang)
		entries, _ := os.ReadDir(dir)
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "index.md" {
				count++
			}
		}
		fmt.Printf("docs/%s chapters: %d\n", lang, count)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "stats:", err)
	os.Exit(1)
}
