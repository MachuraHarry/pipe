package gen

import "fmt"

var varPrefixes = []string{
	"x", "y", "z", "a", "b", "c",
	"val", "sum", "cnt", "idx", "i", "j", "k",
	"result", "tmp", "data", "text", "line", "entry",
	"found", "done", "start", "limit", "total",
}

var fnPrefixes = []string{
	"main", "helper", "calc", "process", "transform",
	"check", "parse", "build", "handle", "run",
	"init", "cleanup", "validate", "lookup", "compute",
	"format", "extract", "filter", "convert", "find",
}

type nameGen struct {
	idx   int
	varI  int
	fnI   int
	seed  int64
}

func newNameGen(seed int64) *nameGen {
	return &nameGen{seed: seed}
}

func (n *nameGen) nextVar() string {
	name := fmt.Sprintf("%s%d", varPrefixes[n.varI%len(varPrefixes)], n.varI)
	n.varI++
	n.idx++
	return name
}

func (n *nameGen) nextFn() string {
	name := fmt.Sprintf("%s%d", fnPrefixes[n.fnI%len(fnPrefixes)], n.fnI)
	n.fnI++
	n.idx++
	return name
}

func (n *nameGen) paramName(idx int) string {
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "x", "y", "n", "s", "val", "input", "data", "text"}
	if idx < len(names) {
		return names[idx]
	}
	return fmt.Sprintf("p%d", idx)
}
