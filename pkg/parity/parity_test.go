// Package parity guards the TV/VM execution parity of the example suite.
//
// The tree-walker and the bytecode VM must produce byte-identical stdout and
// the same exit code for every deterministic, offline example. Differences are
// reported as test failures so that regressions like the sqlite VM bug (nested
// returns unwinding the caller frame) are caught before they are released.
package parity

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// deterministicOffline lists examples whose output is deterministic and needs
// no network, AI keys, stdin, sockets, or filesystem access.
//
// Excluded examples and the reason:
//   - ai_*, blog_*, demo[0-9]*, docs_pipe, generate_json, ollama, parallel_ai,
//     rag_*, redteam, server_greeting, tool_*, web_search, incident_analyzer,
//     commit_summary, fluencyloop: need AI provider keys and/or network.
//   - http_*, github, weather, web_scraper, mcp_*, pipe_web_demo: network/ports.
//   - new_features: writes a module file at runtime and imports it. The TV
//     resolves imports lazily while the VM does so at compile time, so this
//     pattern is inherently TV-only.
//   - hashing_demo: prints a raw map; map iteration order is non-deterministic.
//   - futures_demo: prints inside concurrently resolving futures, so the
//     interleaving order of the prints is non-deterministic.
//   - forin_demo, hash_chain, password_hash, token_gen: random/now output.
//   - filesystem, todo: read/write files on disk.
//   - lambda_basics: reads from stdin.
//   - tcp_echo, tcp_chat, http_server: sockets or sleep.
var deterministicOffline = []string{
	"caesar",
	"calculator",
	"concurrency_channels",
	"concurrency_mutex",
	"concurrency_semaphore",
	"concurrency_spawn_await",
	"fib",
	"fizzbuzz",
	"hello",
	"lambda_pipeline",
	"minitest",
	"palindrome",
	"parallel_pipeline_demo",
	"pipeline",
	"prime",
	"sign_verify",
	"temperature",
	"textstats",
	"xor_cipher",
}

// moduleExamples exercise the sqlite module from ~/.pipe/modules. They are
// skipped when the module is not installed (e.g. a fresh CI runner).
var moduleExamples = []string{
	"sqlite_basic",
	"sqlite_pipeline",
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

func runPipe(t *testing.T, bin, example string, vm bool) (string, int, string) {
	t.Helper()
	root := repoRoot(t)
	args := []string{"-q"}
	if vm {
		args = append(args, "-vm")
	}
	args = append(args, filepath.Join("examples", example+".pipe"))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %s %s: %v", bin, example, err)
		}
	}
	return stdout.String(), code, stderr.String()
}

func compare(t *testing.T, bin, example string) {
	t.Helper()
	tvOut, tvCode, tvErr := runPipe(t, bin, example, false)
	vmOut, vmCode, vmErr := runPipe(t, bin, example, true)

	if tvCode != vmCode || tvOut != vmOut {
		t.Errorf("%s.pipe parity mismatch (TV exit=%d, VM exit=%d):\n"+
			"--- TV stdout ---\n%s\n--- VM stdout ---\n%s\n"+
			"--- TV stderr ---\n%s\n--- VM stderr ---\n%s",
			example, tvCode, vmCode, tvOut, vmOut, tvErr, vmErr)
	}
}

func TestExamplesParity(t *testing.T) {
	root := repoRoot(t)
	bin := buildPipe(t, root)

	for _, name := range deterministicOffline {
		name := name
		t.Run(name, func(t *testing.T) {
			compare(t, bin, name)
		})
	}
}

func TestModuleExamplesParity(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	cache := filepath.Join(home, ".pipe", "modules", "sqlite.pipe")
	if _, err := os.Stat(cache); err != nil {
		t.Skip("sqlite module not installed in ~/.pipe/modules; skipping")
	}

	root := repoRoot(t)
	bin := buildPipe(t, root)

	for _, name := range moduleExamples {
		name := name
		t.Run(name, func(t *testing.T) {
			compare(t, bin, name)
		})
	}
}

// TestIntegrationSuiteParity guards the whole integration test suite: the
// compiled test blocks (`test` statements) must be executed by the VM exactly
// like the tree-walker, so `pipe -test` and `pipe -vm -test` must produce
// byte-identical output and the same exit code.
func TestIntegrationSuiteParity(t *testing.T) {
	root := repoRoot(t)
	bin := buildPipe(t, root)
	dir := filepath.Join(root, "test", "integration")

	run := func(vm bool) (string, int, string) {
		t.Helper()
		args := []string{"-test"}
		if vm {
			args = append(args, "-vm")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Dir = dir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				t.Fatalf("run integration suite: %v", err)
			}
		}
		return stdout.String(), code, stderr.String()
	}

	tvOut, tvCode, tvErr := run(false)
	vmOut, vmCode, vmErr := run(true)

	if tvCode != vmCode || tvOut != vmOut {
		t.Errorf("integration suite parity mismatch (TV exit=%d, VM exit=%d):\n"+
			"--- TV stdout ---\n%s\n--- VM stdout ---\n%s\n"+
			"--- TV stderr ---\n%s\n--- VM stderr ---\n%s",
			tvCode, vmCode, tvOut, vmOut, tvErr, vmErr)
	}
}
