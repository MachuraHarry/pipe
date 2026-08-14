package object_test

import (
	"testing"

	"github.com/MachuraHarry/pipe/pkg/analysis"
	"github.com/MachuraHarry/pipe/pkg/object"
)

// TestAllBuiltinsDocumented keeps the IntelliSense/doc database
// (analysis.builtinDocs) in sync with the registered builtins. Every builtin
// surfaced by the interpreter must be documented for `pipe -doc --builtins`
// and completion.
func TestAllBuiltinsDocumented(t *testing.T) {
	docd := make(map[string]bool, len(analysis.AllBuiltins()))
	for _, d := range analysis.AllBuiltins() {
		docd[d.Name] = true
	}
	var missing []string
	for _, b := range object.Builtins {
		if !docd[b.Name] {
			missing = append(missing, b.Name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("undocumented builtins (%d): %v", len(missing), missing)
	}
}
