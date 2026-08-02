package gen

var varPrefixes = []string{
	"name", "age", "score", "total", "count", "index",
	"result", "data", "text", "value", "list", "nums",
	"sum", "avg", "maxVal", "minVal", "found", "msg",
}

var fnPrefixes = []string{
	"greet", "double", "isEven", "add", "multiply",
	"capitalize", "length", "check", "transform", "process",
	"calculate", "format", "extract", "validate", "lookup",
}

type nameGen struct {
	varI int
	fnI  int
	seed int64
}

func newNameGen(seed int64) *nameGen {
	return &nameGen{seed: seed}
}

func (n *nameGen) nextVar() string {
	name := varPrefixes[n.varI%len(varPrefixes)]
	if n.varI >= len(varPrefixes) {
		name += string(rune('0' + n.varI/len(varPrefixes)))
	}
	n.varI++
	return name
}

func (n *nameGen) nextFn() string {
	name := fnPrefixes[n.fnI%len(fnPrefixes)]
	if n.fnI >= len(fnPrefixes) {
		name += string(rune('0' + n.fnI/len(fnPrefixes)))
	}
	n.fnI++
	return name
}

func (n *nameGen) paramName(idx int) string {
	names := []string{"x", "y", "n", "s", "val", "input", "list", "text"}
	if idx < len(names) {
		return names[idx]
	}
	return string(rune('a' + idx))
}
