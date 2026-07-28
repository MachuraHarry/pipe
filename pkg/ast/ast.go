package ast

import "fmt"

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
	Body       *BlockStatement
}

func (fs *FnStatement) statementNode()       {}
func (fs *FnStatement) TokenLiteral() string { return "fn" }

type VarStatement struct {
	Name  *Identifier
	Value Expression
}

func (vs *VarStatement) statementNode()       {}
func (vs *VarStatement) TokenLiteral() string { return vs.Name.TokenLiteral() }

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
		s += fmt.Sprintf("\n  | %s -> %s", c.Pattern.String(), c.Body.String())
	}
	return s
}

type MatchCase struct {
	Pattern Expression
	Body    Expression
}

// ---- Expressions ----

type Identifier struct {
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Value }
func (i *Identifier) String() string       { return i.Value }

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
func (pe *PrefixExpression) String() string       { return fmt.Sprintf("(%s%s)", pe.Operator, pe.Right.String()) }

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
	Left  Expression
	Right Expression // the function or match to thread through
}

func (pe *PipelineExpression) expressionNode()      {}
func (pe *PipelineExpression) TokenLiteral() string { return ">" }
func (pe *PipelineExpression) String() string {
	return fmt.Sprintf("(%s > %s)", pe.Left.String(), pe.Right.String())
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

type MapLiteral struct {
	Pairs map[string]Expression
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return "{" }
func (ml *MapLiteral) String() string {
	var pairs string
	i := 0
	for k, v := range ml.Pairs {
		if i > 0 {
			pairs += ", "
		}
		pairs += fmt.Sprintf("%s: %s", k, v.String())
		i++
	}
	return fmt.Sprintf("{%s}", pairs)
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
	Update    Expression
	Body      *BlockStatement
	// For-in variant:
	Iterator  *Identifier   // loop variable name
	Iterable  Expression    // list/map to iterate over
	IsForIn   bool
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
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return "import" }

type DeferStatement struct {
	Expression Expression
}

func (ds *DeferStatement) statementNode()       {}
func (ds *DeferStatement) TokenLiteral() string { return "defer" }

type ExportStatement struct {
	FnName string
	Fn     *FnStatement
}

func (es *ExportStatement) statementNode()       {}
func (es *ExportStatement) TokenLiteral() string { return "export" }

type EnumStatement struct {
	Name   string
	Values []string
}

func (es *EnumStatement) statementNode()       {}
func (es *EnumStatement) TokenLiteral() string { return "enum" }

type FnLiteral struct {
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fl *FnLiteral) expressionNode()      {}
func (fl *FnLiteral) TokenLiteral() string { return "fn" }
func (fl *FnLiteral) String() string       { return "fn(...)" }

type SliceExpression struct {
	List   Expression
	Start  Expression
	End    Expression
}

func (se *SliceExpression) expressionNode()      {}
func (se *SliceExpression) TokenLiteral() string { return "slice" }
func (se *SliceExpression) String() string       { return "slice" }

type TryExpression struct {
	TryBlock    *BlockStatement
	CatchParam  *Identifier
	CatchBlock  *BlockStatement
}

func (te *TryExpression) expressionNode()      {}
func (te *TryExpression) TokenLiteral() string { return "try" }
func (te *TryExpression) String() string       { return "try ... catch" }
