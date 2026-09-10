package object

import (
	"testing"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

// fakeSpawner is a minimal UserFunctionSpawner/UserFunctionExecutor stand-in
// for *eval.EvalContext, avoiding an import of the eval package (which
// itself imports object, so an internal `package object` test cannot import
// it without a cycle). It only needs to prove that runToolBatch actually
// dispatches parallel_safe Pipe fn tools through SpawnUserFunction and runs
// them concurrently, not that eval's real spawning machinery is correct
// (that is covered separately by eval's own tests).
type fakeSpawner struct{}

func (fakeSpawner) CallUserFunction(fn Object, args ...Object) Object { return NILOBJ }

func (fakeSpawner) SpawnUserFunction(fn Object, args ...Object) *Future {
	f := NewFuture()
	go func() {
		time.Sleep(300 * time.Millisecond)
		f.Val = &Integer{Value: 1}
		close(f.Done)
	}()
	return f
}

func TestRunToolBatchParallelSafePipeFnRunsConcurrently(t *testing.T) {
	fn := &Function{Name: "slow", EvalCtx: fakeSpawner{}}
	names := []string{"test_slow_a", "test_slow_b", "test_slow_c"}
	for _, n := range names {
		toolRegistry[n] = ToolEntry{Def: ai.ToolDef{Name: n, ParallelSafe: true}, Fn: fn}
	}
	defer func() {
		for _, n := range names {
			delete(toolRegistry, n)
		}
	}()

	calls := make([]ai.ToolCallRequest, len(names))
	for i, n := range names {
		calls[i] = ai.ToolCallRequest{Name: n}
	}

	start := time.Now()
	results := runToolBatch(nil, calls)
	elapsed := time.Since(start)

	if elapsed > 600*time.Millisecond {
		t.Fatalf("expected the 3 calls to overlap (~300ms total), took %v — looks sequential (~900ms)", elapsed)
	}
	for i, r := range results {
		if r.Err != nil {
			t.Fatalf("call %d (%s): unexpected error: %v", i, names[i], r.Err)
		}
		if r.Content != "1" {
			t.Fatalf("call %d (%s): expected content %q, got %q", i, names[i], "1", r.Content)
		}
	}
}

func TestRunToolBatchNonParallelSafePipeFnRunsSynchronously(t *testing.T) {
	fn := &Function{Name: "slow", EvalCtx: fakeSpawner{}}
	toolRegistry["test_not_safe"] = ToolEntry{Def: ai.ToolDef{Name: "test_not_safe", ParallelSafe: false}, Fn: fn}
	defer delete(toolRegistry, "test_not_safe")

	// CallUserFunction on our fakeSpawner returns NILOBJ synchronously, so a
	// non-parallel_safe entry must go through that path (Inspect() == "nil"),
	// never through SpawnUserFunction's goroutine (which would yield "1").
	results := runToolBatch(nil, []ai.ToolCallRequest{{Name: "test_not_safe"}})
	if results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}
	if results[0].Content != NILOBJ.Inspect() {
		t.Fatalf("expected synchronous CallUserFunction dispatch (content %q), got %q", NILOBJ.Inspect(), results[0].Content)
	}
}
