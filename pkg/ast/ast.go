package ast

import (
	"fmt"
	"strings"
)

type Node interface {
	TokenLiteral() string
	// Only one of these will be non-nil:
	// Expression nodes implement String()
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
	String() string
}

// Position is a source location: 1-based line, 1-based column.
type Position struct {
	Line int
	Col  int
}

// Program is the root node of every Pipe AST.
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out string
	for _, s := range p.Statements {
		out += fmt.Sprintf("%s\n", s.TokenLiteral())
	}
	return out
}

// ---- Statements ----

type ExpressionStatement struct {
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Expression.TokenLiteral() }
func (es *ExpressionStatement) String() string       { return es.Expression.String() }

type FnStatement struct {
	Name       *Identifier
	Parameters []*Identifier
	ParamTypes []*TypeAnnotation
	ReturnType *TypeAnnotation
	Body       *BlockStatement
}

func (fs *FnStatement) statementNode()       {}
func (fs *FnStatement) TokenLiteral() string { return "fn" }

type VarStatement struct {
	Name           *Identifier
	TypeAnnotation *TypeAnnotation
	Value          Expression
}

func (vs *VarStatement) statementNode()       {}
func (vs *VarStatement) TokenLiteral() string { return vs.Name.TokenLiteral() }

type TypeAnnotation struct {
	Name string // "int", "string", "list", "map", "fn", etc.
	Line int
	Col  int
}

func (ta *TypeAnnotation) expressionNode()      {}
func (ta *TypeAnnotation) TokenLiteral() string { return ta.Name }
func (ta *TypeAnnotation) String() string       { return ta.Name }
func (ta *TypeAnnotation) Pos() Position        { return Position{Line: ta.Line, Col: ta.Col} }

type BlockStatement struct {
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return "block" }

type IfExpression struct {
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return "if" }
func (ie *IfExpression) String() string {
	s := fmt.Sprintf("if %s {\n%s}", ie.Condition.String(), ie.Consequence.TokenLiteral())
	if ie.Alternative != nil {
		s += fmt.Sprintf(" else {\n%s}", ie.Alternative.TokenLiteral())
	}
	return s
}

type MatchExpression struct {
	Value Expression
	Cases []MatchCase
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return "match" }
func (me *MatchExpression) String() string {
	s := fmt.Sprintf("match %s", me.Value.String())
	for _, c := range me.Cases {
		s += fmt.Sprintf("\n  | %s", c.Pattern.String())
		if c.Guard != nil {
			s += fmt.Sprintf(" if %s", c.Guard.String())
		}
		s += fmt.Sprintf(" -> %s", c.Body.String())
	}
	return s
}

type MatchCase struct {
	Pattern Expression
	Body    Expression
	Guard   Expression // guard condition; nil when there is none
	Bind    string     // variable binding name for the matched value (e.g. "x" in `| x: Some(x) -> ...`)
}

// BindingPattern wraps a pattern and binds the matched value to a variable.
// Used in match cases: `| pattern as name -> body` or `| name: pattern -> body`
type BindingPattern struct {
	Name    string
	Pattern Expression
	Line    int
	Col     int
}

func (bp *BindingPattern) expressionNode()      {}
func (bp *BindingPattern) TokenLiteral() string { return bp.Name }
func (bp *BindingPattern) String() string       { return bp.Name + ": " + bp.Pattern.String() }
func (bp *BindingPattern) Pos() Position        { return Position{Line: bp.Line, Col: bp.Col} }

// ListDestructurePattern destructures a list into named variables.
// e.g. `[a, b, ...rest]` or `[first, _, third]`
type ListDestructurePattern struct {
	Elements []Expression // each element is an Identifier or a nested pattern
	Rest     string       // rest variable name (e.g. "rest" in `[a, ...rest]`), empty if none
	Line     int
	Col      int
}

func (ld *ListDestructurePattern) expressionNode()      {}
func (ld *ListDestructurePattern) TokenLiteral() string { return "list_destructure" }
func (ld *ListDestructurePattern) String() string       { return "[list destructure]" }
func (ld *ListDestructurePattern) Pos() Position        { return Position{Line: ld.Line, Col: ld.Col} }

// MapDestructurePattern destructures a map into named variables.
// e.g. `{name: n, age: a}` binds `n` to key "name" and `a` to key "age"
type MapDestructurePattern struct {
	Keys   []*Identifier // map keys to extract
	Values []*Identifier // variable names to bind to (same length as Keys)
	Line   int
	Col    int
}

func (md *MapDestructurePattern) expressionNode()      {}
func (md *MapDestructurePattern) TokenLiteral() string { return "map_destructure" }
func (md *MapDestructurePattern) String() string       { return "{map destructure}" }
func (md *MapDestructurePattern) Pos() Position        { return Position{Line: md.Line, Col: md.Col} }

// ---- Expressions ----

type Identifier struct {
	Value string
	Line  int
	Col   int
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Value }
func (i *Identifier) String() string       { return i.Value }
func (i *Identifier) Pos() Position        { return Position{Line: i.Line, Col: i.Col} }

type IntegerLiteral struct {
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return fmt.Sprintf("%d", il.Value) }
func (il *IntegerLiteral) String() string       { return fmt.Sprintf("%d", il.Value) }

type FloatLiteral struct {
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fmt.Sprintf("%g", fl.Value) }
func (fl *FloatLiteral) String() string       { return fmt.Sprintf("%g", fl.Value) }

type StringLiteral struct {
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Value }
func (sl *StringLiteral) String() string       { return fmt.Sprintf("%q", sl.Value) }

type BooleanLiteral struct {
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return fmt.Sprintf("%t", bl.Value) }
func (bl *BooleanLiteral) String() string       { return fmt.Sprintf("%t", bl.Value) }

type NilLiteral struct{}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return "nil" }
func (nl *NilLiteral) String() string       { return "nil" }

type PrefixExpression struct {
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Operator }
func (pe *PrefixExpression) String() string {
	return fmt.Sprintf("(%s%s)", pe.Operator, pe.Right.String())
}

type InfixExpression struct {
	Operator string
	Left     Expression
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Operator }
func (ie *InfixExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", ie.Left.String(), ie.Operator, ie.Right.String())
}

type PipelineExpression struct {
	Left     Expression
	Right    Expression // the function or match to thread through
	Parallel bool       // true if >> was used instead of >
}

func (pe *PipelineExpression) expressionNode() {}
func (pe *PipelineExpression) TokenLiteral() string {
	if pe.Parallel {
		return ">>"
	}
	return ">"
}
func (pe *PipelineExpression) String() string {
	op := ">"
	if pe.Parallel {
		op = ">>"
	}
	return fmt.Sprintf("(%s %s %s)", pe.Left.String(), op, pe.Right.String())
}

type CallExpression struct {
	Function  Expression
	Arguments []Expression
	PipedArg  bool // true if piped value was already inserted as arg
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return "call" }
func (ce *CallExpression) String() string {
	var args string
	for i, a := range ce.Arguments {
		if i > 0 {
			args += " "
		}
		args += a.String()
	}
	return fmt.Sprintf("%s(%s)", ce.Function.String(), args)
}

type ListLiteral struct {
	Elements []Expression
}

func (ll *ListLiteral) expressionNode()      {}
func (ll *ListLiteral) TokenLiteral() string { return "[" }
func (ll *ListLiteral) String() string {
	var elems string
	for i, e := range ll.Elements {
		if i > 0 {
			elems += ", "
		}
		elems += e.String()
	}
	return fmt.Sprintf("[%s]", elems)
}

// MapEntry is a single key/value pair in a map literal, preserving the
// source declaration order.
type MapEntry struct {
	Key   string
	Value Expression
}

type MapLiteral struct {
	Pairs []MapEntry
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return "{" }
func (ml *MapLiteral) Get(key string) (Expression, bool) {
	for _, p := range ml.Pairs {
		if p.Key == key {
			return p.Value, true
		}
	}
	return nil, false
}
func (ml *MapLiteral) String() string {
	parts := make([]string, 0, len(ml.Pairs))
	for _, p := range ml.Pairs {
		parts = append(parts, fmt.Sprintf("%s: %s", p.Key, p.Value.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

type DotExpression struct {
	Left  Expression
	Field string
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return "." }
func (de *DotExpression) String() string       { return fmt.Sprintf("(%s.%s)", de.Left.String(), de.Field) }

// ---- New: while/for/break/continue/import ----

type WhileExpression struct {
	Condition Expression
	Body      *BlockStatement
}

func (we *WhileExpression) expressionNode()      {}
func (we *WhileExpression) TokenLiteral() string { return "while" }
func (we *WhileExpression) String() string {
	return fmt.Sprintf("while %s { ... }", we.Condition.String())
}

type ForExpression struct {
	Init      Statement
	Condition Expression
	Update    Statement
	Body      *BlockStatement
	// For-in variant:
	Iterator *Identifier // loop variable name
	Iterable Expression  // list/map to iterate over
	IsForIn  bool
}

func (fe *ForExpression) expressionNode()      {}
func (fe *ForExpression) TokenLiteral() string { return "for" }
func (fe *ForExpression) String() string       { return "for ... { ... }" }

type BreakStatement struct{}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return "break" }

type ContinueStatement struct{}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return "continue" }

type ReturnStatement struct {
	Value Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return "return" }

type ImportStatement struct {
	Path  string
	Alias string // optional namespace alias
	Line  int
	Col   int
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return "import" }
func (is *ImportStatement) Pos() Position        { return Position{Line: is.Line, Col: is.Col} }

type DeferStatement struct {
	Expression Expression
}

func (ds *DeferStatement) statementNode()       {}
func (ds *DeferStatement) TokenLiteral() string { return "defer" }

type ExportStatement struct {
	FnName   string
	Fn       *FnStatement
	VarName  string
	Var      *VarStatement
	EnumName string
	Enum     *EnumStatement
}

func (es *ExportStatement) statementNode()       {}
func (es *ExportStatement) TokenLiteral() string { return "export" }
func (es *ExportStatement) ExportName() string {
	if es.FnName != "" {
		return es.FnName
	}
	if es.VarName != "" {
		return es.VarName
	}
	return es.EnumName
}

type EnumStatement struct {
	Name     string
	Values   []string
	ValuePos []Position
	Line     int
	Col      int
}

func (es *EnumStatement) statementNode()       {}
func (es *EnumStatement) TokenLiteral() string { return "enum" }
func (es *EnumStatement) Pos() Position        { return Position{Line: es.Line, Col: es.Col} }

type TestStatement struct {
	// Name is the string description of a regular test (test "name").
	Name *StringLiteral
	// Hook is "setup" or "teardown" for the file-level hooks (`test setup`,
	// `test teardown`); empty for regular tests.
	Hook string
	Body *BlockStatement
}

func (ts *TestStatement) statementNode()       {}
func (ts *TestStatement) TokenLiteral() string { return "test" }

type FnLiteral struct {
	Parameters []*Identifier
	ParamTypes []*TypeAnnotation
	ReturnType *TypeAnnotation
	Body       *BlockStatement
	Line       int
	Col        int
}

func (fl *FnLiteral) expressionNode()      {}
func (fl *FnLiteral) TokenLiteral() string { return "fn" }
func (fl *FnLiteral) String() string       { return "fn(...)" }
func (fl *FnLiteral) Pos() Position        { return Position{Line: fl.Line, Col: fl.Col} }

type SliceExpression struct {
	List  Expression
	Start Expression
	End   Expression
}

func (se *SliceExpression) expressionNode()      {}
func (se *SliceExpression) TokenLiteral() string { return "slice" }
func (se *SliceExpression) String() string       { return "slice" }

type TryExpression struct {
	TryBlock   *BlockStatement
	CatchParam *Identifier
	CatchBlock *BlockStatement
	AIFix      bool
}

func (te *TryExpression) expressionNode() {}
func (te *TryExpression) TokenLiteral() string {
	if te.AIFix {
		return "try_ai"
	}
	return "try"
}
func (te *TryExpression) String() string { return "try ... catch" }

type StructField struct {
	Name    string
	Default Expression
}

type StructStatement struct {
	Name   string
	Fields []StructField
	Line   int
	Col    int
}

func (ss *StructStatement) statementNode()       {}
func (ss *StructStatement) TokenLiteral() string { return "struct" }
func (ss *StructStatement) Pos() Position        { return Position{Line: ss.Line, Col: ss.Col} }

type SelectCase struct {
	Channel   Expression  // nil for default case
	Value     Expression  // nil for receive-only (<- ch)
	Variable  *Identifier // variable to bind received value (nil for send)
	Body      *BlockStatement
	IsDefault bool
}

type SelectExpression struct {
	Cases []*SelectCase
	Line  int
	Col   int
}

func (se *SelectExpression) expressionNode()      {}
func (se *SelectExpression) TokenLiteral() string { return "select" }
func (se *SelectExpression) String() string       { return "select { ... }" }
func (se *SelectExpression) Pos() Position        { return Position{Line: se.Line, Col: se.Col} }
