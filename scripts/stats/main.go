package main

// stats reports the authoritative project statistics used throughout the
// README, website, and documentation so that doc claims can never drift from
// the code. Run with: go run ./scripts/stats
//
// It counts every builtin registered at import time (pkg/object exposes the
// full table after all init functions ran), then derives the secondary
// numbers (examples, tests, docs chapters) directly from the filesystem.

import (
	"encoding/json"
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

type Stats struct {
	Builtins    int            `json:"builtins"`
	BuiltinCats map[string]int `json:"builtin_categories"`
	Examples    int            `json:"examples"`
	GoTests     int            `json:"go_tests"`
	TestPackages int           `json:"test_packages"`
	ASTNodeTypes int           `json:"ast_node_types"`
	DocsEn      int            `json:"docs_en_chapters"`
	DocsDe      int            `json:"docs_de_chapters"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	stats := &Stats{BuiltinCats: map[string]int{}}

	total := len(object.Builtins)
	stats.Builtins = total
	for _, b := range object.Builtins {
		switch {
		case strings.HasPrefix(b.Name, "ai_") || aiNames[b.Name]:
			if strings.HasPrefix(b.Name, "ai_") {
				stats.BuiltinCats["ai_"]++
			}
			stats.BuiltinCats["ai_all"]++
		case strings.HasPrefix(b.Name, "mcp_"):
			stats.BuiltinCats["mcp_"]++
		case sandboxNames[b.Name]:
			stats.BuiltinCats["sandbox"]++
		case fileNames[b.Name]:
			stats.BuiltinCats["file"]++
		default:
			stats.BuiltinCats["stdlib"]++
		}
	}

	examples, err := filepath.Glob(filepath.Join(root, "examples", "*.pipe"))
	if err != nil {
		fatal(err)
	}
	stats.Examples = len(examples)

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
	stats.GoTests = testFns
	stats.TestPackages = len(testPkgs)

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
	stats.ASTNodeTypes = astNodes

	for _, lang := range []string{"en", "de"} {
		dir := filepath.Join(root, "docs", lang)
		entries, _ := os.ReadDir(dir)
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "index.md" {
				count++
			}
		}
		if lang == "en" {
			stats.DocsEn = count
		} else {
			stats.DocsDe = count
		}
	}

	writeJSON(root, stats)
	report(stats)
}

// statsJSONPath is the committed canonical statistics snapshot that CI checks
// against. When it drifts from the live numbers, the documentation claims
// (README, website, docs) are stale and CI fails.
const statsJSONPath = "stats.json"

func writeJSON(root string, stats *Stats) {
	raw, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, statsJSONPath), append(raw, '\n'), 0644); err != nil {
		fatal(err)
	}
}

func report(s *Stats) {
	fmt.Println("== Builtins ==")
	fmt.Printf("total: %d\n", s.Builtins)
	for _, key := range []string{"ai_", "ai_all", "mcp_", "sandbox", "file", "stdlib"} {
		fmt.Printf("  %-8s %d\n", key+":", s.BuiltinCats[key])
	}
	fmt.Println()
	fmt.Printf("examples: %d\n", s.Examples)
	fmt.Printf("go tests: %d\n", s.GoTests)
	fmt.Printf("test packages: %d\n", s.TestPackages)
	fmt.Printf("AST node types: %d\n", s.ASTNodeTypes)
	fmt.Printf("docs/en chapters: %d\n", s.DocsEn)
	fmt.Printf("docs/de chapters: %d\n", s.DocsDe)
	fmt.Println()
	fmt.Printf("wrote %s\n", statsJSONPath)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "stats:", err)
	os.Exit(1)
}
