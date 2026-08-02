package gen

import "math/rand"

type PipeType int

const (
	TypeInt PipeType = iota
	TypeFloat
	TypeString
	TypeBool
	TypeNil
	TypeList
	TypeMap
	TypeFn
	TypeError
	TypeResult
	TypeAny
)

func (t PipeType) String() string {
	switch t {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeString:
		return "string"
	case TypeBool:
		return "bool"
	case TypeNil:
		return "nil"
	case TypeList:
		return "list"
	case TypeMap:
		return "map"
	case TypeFn:
		return "fn"
	case TypeError:
		return "error"
	case TypeResult:
		return "result"
	default:
		return "any"
	}
}

type SymbolKind int

const (
	KindVariable SymbolKind = iota
	KindFunction
	KindParam
	KindBuiltin
	KindEnumValue
)

type ScopeEntry struct {
	Kind  SymbolKind
	Type  PipeType
	Arity int
}

type GenOptions struct {
	Seed       int64
	MaxStmts   int
	MaxDepth   int
	Pipelines  bool
	TryExprs   bool
	Tests      bool
}

func DefaultOptions() GenOptions {
	return GenOptions{
		Seed:      1,
		MaxStmts:  8,
		MaxDepth:  4,
		Pipelines: true,
		TryExprs:  false,
		Tests:     false,
	}
}

type weights struct {
	fnDef    int
	varDef   int
	enumDef  int
	exprStmt int
}

func defaultWeights() weights {
	return weights{
		fnDef:    4,
		varDef:   6,
		enumDef:  1,
		exprStmt: 6,
	}
}

func randInt(rng *rand.Rand, lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + rng.Intn(hi-lo)
}
