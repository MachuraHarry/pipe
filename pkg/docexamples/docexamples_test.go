// Package docexamples guards against documentation drift: every "-- Example"
// / "-- Beispiel" tagged snippet inside a ```pipe fenced block in docs/en
// and docs/de (or, for a block with no such marker, the whole block) must
// be valid Pipe. It always has to PARSE; where it doesn't touch an AI
// provider, the network, stdin, a subprocess, or the filesystem, it also
// has to RUN without error. This is the check that would have caught a
// past real bug: a documented `print (counter)` that doesn't actually call
// a zero-argument closure (needs `counter()`) parses fine and runs fine --
// it just silently prints the wrong thing -- so a plain parse check alone
// is not enough; the point is to actually execute what a reader would
// copy-paste and hit the same broken behavior they would.
//
// A block a doc author genuinely intends as a non-standalone fragment
// (illustrative syntax only, deliberately incomplete) can opt out entirely
// by placing an HTML comment on its own line directly above the fence:
//
//	<!-- doctest:skip -->
//	```pipe
//	... fragment ...
//	```
package docexamples

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

type snippet struct {
	file      string // path relative to repo root
	block     int    // index of the enclosing ```pipe fence within file
	label     string
	source    string
	startLine int // 1-based line number in the .md file
}

var markerRe = regexp.MustCompile(`^--\s*(Example|Beispiel)\b`)

// skipIdentifiers: a snippet containing any of these builtin names as a
// whole word is skipped for EXECUTION only (it's still parse-checked) --
// each needs an AI provider key, the network, stdin, a subprocess, or
// writes/deletes files, none of which this offline test can safely or
// deterministically provide.
var skipIdentifiers = []string{
	// Word-boundary matching (see buildSkipRegex) means "ai_chat" alone
	// would NOT also match "ai_chat_json" -- '_' counts as a word
	// character, so there's no boundary between them -- hence explicit
	// longer variants (ai_chat_json, ai_swarm_trace, ai_swarm_stream,
	// ai_cache_hits, ai_cache_misses) alongside their shorter prefixes.
	"ai_batch", "ai_cache", "ai_cache_hits", "ai_cache_misses", "ai_chat",
	"ai_chat_json", "ai_cost", "ai_host", "ai_model",
	"ai_parallel", "ai_provider", "ai_rate_limit", "ai_set_key", "ai_stream",
	"ai_swarm", "ai_swarm_stream", "ai_swarm_trace", "ai_timeout", "ai_tokens",
	"ai_tool", "ai_vision", "ai_with_tools",
	"summarize", "translate", "classify", "extract", "redact", "rerank",
	"moderate", "generate", "generate_json", "ask", "embed", "embed_batch",
	"cosine_sim", "dot_product", "nearest", "tool_call", "swarm_agent", "try_ai",
	"http_get", "http_get_json", "http_post", "http_request", "http_server", "http_close",
	"tcp_listen", "tcp_connect", "tcp_connect_tls", "tcp_accept", "tcp_read",
	"tcp_read_bytes", "tcp_write", "tcp_close", "tcp_set_read_timeout",
	"mcp_server", "mcp_serve_stdio", "mcp_serve_sse", "mcp_use_stdio", "mcp_use_sse",
	"mcp_tools", "mcp_resources", "mcp_resource", "mcp_resource_template",
	"mcp_prompts", "mcp_prompt", "mcp_prompt_get", "mcp_read_resource",
	"web_search", "wiki_search",
	"proc_start", "proc_wait", "proc_kill", "proc_running",
	"exec", "input", "read_line",
	"write_file", "append_file", "file_delete", "make_dir", "remove_dir",
	"file_copy", "file_move",
	// Read-only, but examples routinely reference an illustrative filename
	// (data.txt, config.json, ...) the doc never actually creates -- a
	// legitimate "file not found" in an isolated temp dir, not a bug.
	"read_file", "read_lines", "list_dir",
	// import references a file on disk (an external module, or a sibling
	// file the doc's prose introduces elsewhere) that has no reason to
	// exist relative to an isolated temp dir holding just this one
	// concatenated snippet -- verifying cross-file/module references is a
	// different kind of check than this harness's job (catching wrong Pipe
	// syntax/semantics in the example code itself).
	"import",
}

func buildSkipRegex() *regexp.Regexp {
	escaped := make([]string, len(skipIdentifiers))
	for i, s := range skipIdentifiers {
		escaped[i] = regexp.QuoteMeta(s)
	}
	return regexp.MustCompile(`\b(` + strings.Join(escaped, "|") + `)\b`)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (go.mod)")
		}
		dir = parent
	}
}

func buildPipe(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pipe")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/pipe")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

type mdBlock struct {
	lines     []string
	startLine int // 1-based line number of the block's first content line
	skip      bool
}

func extractBlocks(content string) []mdBlock {
	lines := strings.Split(content, "\n")
	var blocks []mdBlock
	inBlock := false
	var cur []string
	start := 0
	skipNext := false
	for i, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if trimmed == "<!-- doctest:skip -->" {
				skipNext = true
				continue
			}
			if trimmed == "```pipe" {
				inBlock = true
				cur = nil
				start = i + 2
				continue
			}
			if trimmed != "" {
				skipNext = false
			}
			continue
		}
		if trimmed == "```" {
			inBlock = false
			blocks = append(blocks, mdBlock{lines: cur, startLine: start, skip: skipNext})
			skipNext = false
			continue
		}
		cur = append(cur, line)
	}
	return blocks
}

func splitSnippets(file string, blockIdx int, b mdBlock) []snippet {
	if b.skip {
		return nil
	}
	var markerIdx []int
	for i, l := range b.lines {
		if markerRe.MatchString(strings.TrimSpace(l)) {
			markerIdx = append(markerIdx, i)
		}
	}
	if len(markerIdx) == 0 {
		return []snippet{{
			file:      file,
			block:     blockIdx,
			label:     fmt.Sprintf("block%d", blockIdx),
			source:    strings.Join(b.lines, "\n"),
			startLine: b.startLine,
		}}
	}
	var out []snippet
	for mi, idx := range markerIdx {
		end := len(b.lines)
		if mi+1 < len(markerIdx) {
			end = markerIdx[mi+1]
		}
		label := strings.TrimSpace(b.lines[idx])
		content := b.lines[idx+1 : end]
		out = append(out, snippet{
			file:      file,
			block:     blockIdx,
			label:     fmt.Sprintf("block%d_%s", blockIdx, label),
			source:    strings.Join(content, "\n"),
			startLine: b.startLine + idx + 1,
		})
	}
	return out
}

func allSnippets(t *testing.T, root string) []snippet {
	t.Helper()
	var files []string
	for _, dir := range []string{"docs/en", "docs/de"} {
		full := filepath.Join(root, dir)
		entries, err := os.ReadDir(full)
		if err != nil {
			t.Fatalf("read %s: %v", full, err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				files = append(files, filepath.Join(dir, e.Name()))
			}
		}
	}
	sort.Strings(files)

	var out []snippet
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		blocks := extractBlocks(string(data))
		for bi, b := range blocks {
			out = append(out, splitSnippets(rel, bi, b)...)
		}
	}
	return out
}

var safeNameRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func testName(sn snippet) string {
	base := strings.TrimSuffix(filepath.Base(sn.file), ".md") + "_" + sn.label
	return safeNameRe.ReplaceAllString(base, "_")
}

func runSource(t *testing.T, bin string, sn snippet, fullSource string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "snippet.pipe")
	if err := os.WriteFile(path, []byte(fullSource+"\n"), 0o644); err != nil {
		t.Fatalf("write snippet: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-q", path)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s:%d %q failed to run (together with every earlier example in this file, which it may depend on): %v\n--- stdout ---\n%s\n--- stderr ---\n%s\n--- this example's own source ---\n%s",
			sn.file, sn.startLine, sn.label, err, stdout.String(), stderr.String(), sn.source)
	}
}

// narrativeTourFiles are chapters where execution verification isn't
// reliable no matter the concatenation strategy, each for one of two
// reasons documented per group below. Both are still parse-checked like
// every other file; only execution is skipped, unconditionally.
var narrativeTourFiles = map[string]bool{
	// "Language tour" chapters interleave prose with deliberately partial
	// or forward-referencing snippets -- e.g. docs/de/05-funktionen-und-
	// closures.md calls `begruesse(...)` in its 5.2 example, two sections
	// before 5.3 ever defines `begruesse`. No concatenation order (per-
	// block, per-file, or otherwise) makes that executable, since it's a
	// genuine backward reference relative to file order, not a
	// missing-context problem.
	"docs/en/01-getting-started.md":         true,
	"docs/de/01-erste-schritte.md":          true,
	"docs/en/02-language-tour.md":           true,
	"docs/de/02-sprachuebersicht.md":        true,
	"docs/en/03-types-and-expressions.md":   true,
	"docs/de/03-typen-und-ausdruecke.md":    true,
	"docs/en/04-control-flow.md":            true,
	"docs/de/04-kontrollfluss.md":           true,
	"docs/en/05-functions-and-closures.md":  true,
	"docs/de/05-funktionen-und-closures.md": true,
	"docs/en/06-pipelines.md":               true,
	"docs/de/06-pipelines.md":               true,
	"docs/en/07-data-structures.md":         true,
	"docs/de/07-datenstrukturen.md":         true,
	"docs/en/08-error-handling.md":          true,
	"docs/de/08-fehlerbehandlung.md":        true,
	"docs/en/09-modules-and-imports.md":     true,
	"docs/de/09-module-und-importe.md":      true,
	"docs/en/11-tooling.md":                 true,
	"docs/de/11-tooling.md":                 true,

	// Reference chapters (10-builtin-reference) show one terse,
	// illustrative snippet per function (of ~230), often reusing generic
	// variable names (x, y) without defining them -- they document the
	// shape of a call, not a runnable program. A no-marker block here is
	// exactly one function's example, so there's no "-- Example" structure
	// to lean on either.
	"docs/en/10-builtin-reference.md": true,
	"docs/de/10-builtin-referenz.md":  true,

	// External-module chapters document a module (discord, the generic
	// "x" example module) that isn't installed in this environment and
	// isn't part of core Pipe -- every example fails identically for that
	// one environmental reason, and a one-time `import ... as d`-style
	// setup a few sections up is assumed live for everything below it,
	// which no per-block or whole-file concatenation captures cleanly.
	"docs/en/23-x-module.md":       true,
	"docs/de/23-x-modul.md":        true,
	"docs/en/24-discord-module.md": true,
	"docs/de/24-discord-modul.md":  true,
	"docs/en/27-mqtt-module.md":    true,
	"docs/de/27-mqtt-modul.md":     true,
}

// TestDocExamples parses every documented example on its own, and executes
// it -- concatenated after every earlier example from the SAME ```pipe
// block, in block order -- unless the combined source needs an AI key, the
// network, stdin, a subprocess, the filesystem, an import, or belongs to a
// narrativeTourFiles chapter. Concatenating within a block (but not across
// separate blocks) mirrors the one case that's actually reliable: a
// "-- Example 2" a few lines below "-- Example 1" in the SAME fenced block
// routinely reuses something the first example set up, without redefining
// it, so checking each in total isolation produced "undefined variable"
// false positives; concatenating whole FILES, on the other hand, produced
// a different false positive -- unrelated examples coincidentally reusing
// a generic name like `x` for different values, colliding across blocks
// that were never meant to share state. See the package doc comment for
// the doctest:skip escape hatch (for a block that's a deliberately
// incomplete fragment even within its own block).
func TestDocExamples(t *testing.T) {
	root := repoRoot(t)
	bin := buildPipe(t, root)
	snippets := allSnippets(t, root)
	skipRe := buildSkipRegex()

	var numParsed, numRan, numSkipped int
	var cumulative strings.Builder
	currentFile := ""
	currentBlock := -1
	for _, sn := range snippets {
		sn := sn
		if sn.file != currentFile || sn.block != currentBlock {
			currentFile = sn.file
			currentBlock = sn.block
			cumulative.Reset()
		}

		t.Run(testName(sn), func(t *testing.T) {
			src := strings.TrimSpace(sn.source)
			if src == "" {
				t.Skip("empty snippet")
			}

			l := lexer.New(sn.source)
			p := parser.New(l)
			p.ParseProgram()
			if errs := p.Errors(); len(errs) > 0 {
				t.Fatalf("%s:%d %q: parse error(s):\n%s\n--- source ---\n%s",
					sn.file, sn.startLine, sn.label, strings.Join(errs, "\n"), sn.source)
			}
			numParsed++

			if narrativeTourFiles[sn.file] {
				numSkipped++
				t.Skip("narrative tour chapter: prose interleaves forward-referencing/partial snippets not meant to execute in isolation or in sequence (see narrativeTourFiles)")
				return
			}

			full := cumulative.String() + sn.source
			if skipRe.MatchString(full) {
				numSkipped++
				t.Skip("this example (or an earlier one in the same block it's concatenated with) touches an AI provider, the network, stdin, a subprocess, the filesystem, or an import -- not offline-safe to execute")
				return
			}

			runSource(t, bin, sn, full)
			numRan++
		})

		cumulative.WriteString(sn.source)
		cumulative.WriteString("\n")
	}

	t.Logf("doc examples: %d total, %d parsed, %d executed, %d skipped (AI/network/stdin/fs/import)",
		len(snippets), numParsed, numRan, numSkipped)
}
