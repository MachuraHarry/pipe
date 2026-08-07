package compiler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/object"
)

type SymbolScope int

const (
	GlobalScope SymbolScope = iota
	LocalScope
	BuiltinScope
	FreeScope
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	store          map[string]Symbol
	numDefinitions int
	Outer          *SymbolTable
	FreeSymbols    []Symbol
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{store: make(map[string]Symbol)}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

func (s *SymbolTable) Define(name string) Symbol {
	if existing, ok := s.store[name]; ok {
		return existing
	}
	sym := Symbol{Name: name, Index: s.numDefinitions}
	if s.Outer == nil {
		sym.Scope = GlobalScope
	} else {
		sym.Scope = LocalScope
	}
	s.store[name] = sym
	s.numDefinitions++
	return sym
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	sym, ok := s.store[name]
	if !ok && s.Outer != nil {
		outerSym, outerOk := s.Outer.Resolve(name)
		if !outerOk {
			return Symbol{}, false
		}
		if outerSym.Scope == GlobalScope || outerSym.Scope == BuiltinScope {
			return outerSym, true
		}
		free := Symbol{Name: name, Scope: FreeScope, Index: len(s.FreeSymbols)}
		s.store[name] = free
		s.FreeSymbols = append(s.FreeSymbols, outerSym)
		return free, true
	}
	return sym, ok
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	sym := Symbol{Name: name, Index: index, Scope: BuiltinScope}
	s.store[name] = sym
	return sym
}

type Compiler struct {
	constants   []object.Object
	symbolTable *SymbolTable
	scopes      []CompilationScope
	scopeIndex  int
	loopStack   []LoopContext
	importCache map[string]*ast.Program
	importedSet map[string]struct{}
	sourceFile  string
}

type LoopContext struct {
	continueTarget  int   // position to jump to on continue (condition check)
	continuePatches []int // positions of continue jumps to patch (C-style for)
	breakPatches    []int // positions of break jump instructions to patch
}

type CompilationScope struct {
	instructions Instructions
	deferred     []Instructions
}

type CompiledScope struct {
	Instructions Instructions
	NumLocals    int
	FreeSymbols  []Symbol
}

type Bytecode struct {
	Instructions Instructions
	Constants    []object.Object
}

func New() *Compiler {
	return NewWithFile("")
}

func NewWithFile(sourceFile string) *Compiler {
	mainScope := CompilationScope{
		instructions: Instructions{},
	}

	symbolTable := NewSymbolTable()
	for i, b := range object.Builtins {
		symbolTable.DefineBuiltin(i, b.Name)
	}

	return &Compiler{
		constants:   []object.Object{},
		symbolTable: symbolTable,
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
		importCache: make(map[string]*ast.Program),
		importedSet: make(map[string]struct{}),
		sourceFile:  sourceFile,
	}
}

func (c *Compiler) currentScope() *CompilationScope {
	return &c.scopes[c.scopeIndex]
}

func (c *Compiler) currentInstructions() Instructions {
	return c.currentScope().instructions
}

func (c *Compiler) Bytecode() *Bytecode {
	scope := c.currentScope()
	// Emit top-level deferred expressions before returning bytecode
	for i := len(scope.deferred) - 1; i >= 0; i-- {
		scope.instructions = append(scope.instructions, scope.deferred[i]...)
	}
	scope.deferred = nil

	return &Bytecode{
		Instructions: scope.instructions,
		Constants:    c.constants,
	}
}

func (c *Compiler) emit(op Opcode, operands ...int) int {
	ins := Make(op, operands...)
	pos := len(c.currentScope().instructions)
	c.currentScope().instructions = append(c.currentScope().instructions, ins...)
	return pos
}

func (c *Compiler) emitUint16(v uint16) {
	ins := encodeUint16(v)
	c.currentScope().instructions = append(c.currentScope().instructions, ins...)
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) addString(s string) int {
	return c.addConstant(&object.String{Value: s})
}

func (c *Compiler) addInteger(v int64) int {
	return c.addConstant(&object.Integer{Value: v})
}

func (c *Compiler) addFloat(v float64) int {
	return c.addConstant(&object.Float{Value: v})
}

func (c *Compiler) Compile(node ast.Node) error {
	switch n := node.(type) {
	case *ast.Program:
		// Two-pass: compile definitions first, then expressions
		c.defineTopLevelSymbols(n.Statements)
		return c.compileProgram(n.Statements)

	case *ast.BlockStatement:
		return c.compileStatements(n.Statements, true)

	case *ast.ExpressionStatement:
		if err := c.Compile(n.Expression); err != nil {
			return err
		}
		c.emit(OpPop)

	case *ast.VarStatement:
		symbol, exists := c.symbolTable.Resolve(n.Name.Value)
		if !exists {
			symbol = c.symbolTable.Define(n.Name.Value)
		}
		if err := c.Compile(n.Value); err != nil {
			return err
		}
		if symbol.Scope == GlobalScope {
			c.emit(OpSetGlobal, symbol.Index)
		} else {
			c.emit(OpSetLocal, symbol.Index)
		}

	case *ast.IntegerLiteral:
		idx := c.addInteger(n.Value)
		c.emit(OpConstant, idx)

	case *ast.FloatLiteral:
		idx := c.addFloat(n.Value)
		c.emit(OpConstant, idx)

	case *ast.StringLiteral:
		idx := c.addString(n.Value)
		c.emit(OpConstant, idx)

	case *ast.BooleanLiteral:
		if n.Value {
			c.emit(OpTrue)
		} else {
			c.emit(OpFalse)
		}

	case *ast.NilLiteral:
		c.emit(OpNil)

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(n.Value)
		if !ok {
			return fmt.Errorf("undefined variable: %s", n.Value)
		}
		c.loadSymbol(symbol)
		if symbol.Scope == BuiltinScope && isZeroArityBuiltin(n.Value) {
			c.emit(OpCall, 0)
		}

	case *ast.PrefixExpression:
		if err := c.Compile(n.Right); err != nil {
			return err
		}
		if n.Operator == "-" {
			c.emit(OpMinus)
		} else if n.Operator == "!" {
			c.emit(OpNot)
		} else {
			return fmt.Errorf("unknown prefix: %s", n.Operator)
		}

	case *ast.InfixExpression:
		if n.Operator == "&&" {
			if err := c.Compile(n.Left); err != nil {
				return err
			}
			c.emit(OpDup)
			jumpFalsy := c.emit(OpJumpNotTruthy, 9999)
			c.emit(OpPop)
			if err := c.Compile(n.Right); err != nil {
				return err
			}
			afterRight := len(c.currentInstructions())
			c.patchJump(jumpFalsy, afterRight)
			return nil
		}
		if n.Operator == "||" {
			if err := c.Compile(n.Left); err != nil {
				return err
			}
			c.emit(OpDup)
			jumpEnd := c.emit(OpJumpNotTruthy, 9999) // if falsy, skip to eval right
			jumpEnd2 := c.emit(OpJump, 9999)         // if truthy, skip right entirely
			c.patchJump(jumpEnd, len(c.currentInstructions()))
			c.emit(OpPop)
			if err := c.Compile(n.Right); err != nil {
				return err
			}
			c.patchJump(jumpEnd2, len(c.currentInstructions()))
			return nil
		}
		if err := c.Compile(n.Left); err != nil {
			return err
		}
		if err := c.Compile(n.Right); err != nil {
			return err
		}
		switch n.Operator {
		case "+":
			c.emit(OpAdd)
		case "-":
			c.emit(OpSub)
		case "*":
			c.emit(OpMul)
		case "/":
			c.emit(OpDiv)
		case "%":
			c.emit(OpMod)
		case "**":
			c.emit(OpPow)
		case "==":
			c.emit(OpEqual)
		case "!=":
			c.emit(OpNotEqual)
		case ">":
			c.emit(OpGreater)
		case "<":
			c.emit(OpLess)
		case ">=":
			c.emit(OpGte)
		case "<=":
			c.emit(OpLte)
		case "++":
			c.emit(OpConcat)
		case "[]":
			c.emit(OpGetBuiltin, c.resolveBuiltin("at").Index)
			if err := c.Compile(n.Left); err != nil {
				return err
			}
			if err := c.Compile(n.Right); err != nil {
				return err
			}
			c.emit(OpCall, 2)
		default:
			return fmt.Errorf("unknown operator: %s", n.Operator)
		}

	case *ast.IfExpression:
		if err := c.Compile(n.Condition); err != nil {
			return err
		}
		jumpNotTruthyPos := c.emit(OpJumpNotTruthy, 9999)
		if err := c.compileBlockLastReturn(n.Consequence); err != nil {
			return err
		}

		if n.Alternative != nil {
			jumpPos := c.emit(OpJump, 9999)
			afterConsequence := len(c.currentInstructions())
			c.patchJump(jumpNotTruthyPos, afterConsequence)
			if err := c.compileBlockLastReturn(n.Alternative); err != nil {
				return err
			}
			afterAlternative := len(c.currentInstructions())
			c.patchJump(jumpPos, afterAlternative)
		} else {
			jumpPos := c.emit(OpJump, 9999)
			falseBranch := len(c.currentInstructions())
			c.patchJump(jumpNotTruthyPos, falseBranch)
			c.emit(OpNil)
			afterIf := len(c.currentInstructions())
			c.patchJump(jumpPos, afterIf)
		}

	case *ast.WhileExpression:
		conditionPos := len(c.currentInstructions())
		if err := c.Compile(n.Condition); err != nil {
			return err
		}
		jumpFalsePos := c.emit(OpJumpNotTruthy, 9999)
		c.enterLoop(conditionPos)
		if err := c.compileBlockLastReturn(n.Body); err != nil {
			return err
		}
		c.emit(OpJumpBackward, conditionPos)
		afterLoop := len(c.currentInstructions())
		c.patchJump(jumpFalsePos, afterLoop)
		c.patchBreaks(afterLoop)
		c.leaveLoop()

	case *ast.ForExpression:
		if n.IsForIn {
			return c.compileForIn(n)
		}
		return c.compileCStyleFor(n)

	case *ast.BreakStatement:
		c.addBreak()

	case *ast.ContinueStatement:
		c.emitContinue()

	case *ast.ReturnStatement:
		if n.Value != nil {
			if err := c.Compile(n.Value); err != nil {
				return err
			}
		}
		c.emit(OpReturnValue)

	case *ast.MatchExpression:
		if err := c.compileMatch(n); err != nil {
			return err
		}

	case *ast.FnStatement:
		fnSymbol := c.symbolTable.Define(n.Name.Value)

		c.enterScope()
		for _, p := range n.Parameters {
			c.symbolTable.Define(p.Value)
		}

		if err := c.compileFnBody(n.Body); err != nil {
			return err
		}

		compiledFn := c.leaveScope()

		idx := c.addConstant(&object.CompiledFunction{
			Instructions: compiledFn.Instructions,
			NumLocals:    compiledFn.NumLocals,
			NumFree:      len(compiledFn.FreeSymbols),
		})

		for _, freeSym := range compiledFn.FreeSymbols {
			c.loadSymbol(freeSym)
		}

		c.emit(OpClosure, idx, len(compiledFn.FreeSymbols))

		if fnSymbol.Scope == GlobalScope {
			c.emit(OpSetGlobal, fnSymbol.Index)
		} else {
			c.emit(OpSetLocal, fnSymbol.Index)
		}

	case *ast.FnLiteral:
		c.enterScope()
		for _, p := range n.Parameters {
			c.symbolTable.Define(p.Value)
		}

		if err := c.compileFnBody(n.Body); err != nil {
			return err
		}

		compiledFn := c.leaveScope()

		idx := c.addConstant(&object.CompiledFunction{
			Instructions: compiledFn.Instructions,
			NumLocals:    compiledFn.NumLocals,
			NumFree:      len(compiledFn.FreeSymbols),
		})

		for _, freeSym := range compiledFn.FreeSymbols {
			c.loadSymbol(freeSym)
		}

		c.emit(OpClosure, idx, len(compiledFn.FreeSymbols))

	case *ast.SliceExpression:
		c.emit(OpGetBuiltin, c.resolveBuiltin("slice").Index)
		if err := c.Compile(n.List); err != nil {
			return err
		}
		if n.Start != nil {
			if err := c.Compile(n.Start); err != nil {
				return err
			}
		} else {
			c.emit(OpConstant, c.addInteger(0))
		}
		if n.End != nil {
			if err := c.Compile(n.End); err != nil {
				return err
			}
		} else {
			c.emit(OpGetBuiltin, c.resolveBuiltin("len").Index)
			if err := c.Compile(n.List); err != nil {
				return err
			}
			c.emit(OpCall, 1)
		}
		c.emit(OpCall, 3)

	case *ast.CallExpression:
		if ident, ok := n.Function.(*ast.Identifier); ok && len(n.Arguments) > 0 && isZeroArityBuiltin(ident.Value) {
			if symbol, ok := c.symbolTable.Resolve(ident.Value); ok && symbol.Scope == BuiltinScope {
				c.loadSymbol(symbol)
			} else if err := c.Compile(n.Function); err != nil {
				return err
			}
		} else if err := c.Compile(n.Function); err != nil {
			return err
		}
		for _, arg := range n.Arguments {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}
		c.emit(OpCall, len(n.Arguments))

	case *ast.PipelineExpression:
		if err := c.compilePipeline(n); err != nil {
			return err
		}

	case *ast.ListLiteral:
		for _, elem := range n.Elements {
			if err := c.Compile(elem); err != nil {
				return err
			}
		}
		c.emit(OpList, len(n.Elements))

	case *ast.MapLiteral:
		keys := make([]string, 0, len(n.Pairs))
		for k := range n.Pairs {
			keys = append(keys, k)
		}
		for _, k := range keys {
			if err := c.Compile(n.Pairs[k]); err != nil {
				return err
			}
		}
		c.emit(OpMap, len(n.Pairs))
		for _, k := range keys {
			ki := c.addString(k)
			c.emitUint16(uint16(ki))
		}

	case *ast.DotExpression:
		if err := c.Compile(n.Left); err != nil {
			return err
		}
		idx := c.addString(n.Field)
		c.emit(OpDot, idx)

	case *ast.TryExpression:
		if err := c.compileTryExpression(n); err != nil {
			return err
		}

	case *ast.EnumStatement:
		for i, name := range n.Values {
			sym := c.symbolTable.Define(name)
			c.emit(OpConstant, c.addInteger(int64(i)))
			if sym.Scope == GlobalScope {
				c.emit(OpSetGlobal, sym.Index)
			} else {
				c.emit(OpSetLocal, sym.Index)
			}
		}

	case *ast.StructStatement:
		def := &object.StructDef{
			Name:     n.Name,
			Fields:   make([]string, 0, len(n.Fields)),
			Defaults: make(map[string]object.Object),
		}
		for _, f := range n.Fields {
			def.Fields = append(def.Fields, f.Name)
			if f.Default != nil {
				def.Defaults[f.Name] = &object.NilObject{}
			}
		}
		sym := c.symbolTable.Define(n.Name)
		c.emit(OpConstant, c.addConstant(def))
		if sym.Scope == GlobalScope {
			c.emit(OpSetGlobal, sym.Index)
		} else {
			c.emit(OpSetLocal, sym.Index)
		}

	case *ast.ImportStatement:
		if err := c.compileImport(n); err != nil {
			return err
		}

	case *ast.ExportStatement:
		if n.Fn != nil {
			if err := c.Compile(n.Fn); err != nil {
				return err
			}
		}
		if n.Var != nil {
			if err := c.Compile(n.Var); err != nil {
				return err
			}
		}
		if n.Enum != nil {
			if err := c.Compile(n.Enum); err != nil {
				return err
			}
		}

	case *ast.DeferStatement:
		if err := c.compileDefer(n); err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) compileProgram(stmts []ast.Statement) error {
	// Pass 1: compile only declarations (closures, enums, exports, imports)
	// VarStatements are NOT compiled here — their values may have side effects
	// that depend on execution order (e.g., ai_provider before ai_batch).
	for _, stmt := range stmts {
		switch stmt.(type) {
		case *ast.FnStatement, *ast.EnumStatement, *ast.StructStatement,
			*ast.ExportStatement, *ast.ImportStatement:
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}
	}

	// Pass 2: compile values (VarStatements) and expressions in source order
	for _, stmt := range stmts {
		switch stmt.(type) {
		case *ast.FnStatement, *ast.EnumStatement, *ast.StructStatement,
			*ast.ExportStatement, *ast.ImportStatement:
			continue
		}
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileStatements(stmts []ast.Statement, popLast bool) error {
	for i, stmt := range stmts {
		if i == len(stmts)-1 && !popLast {
			if es, ok := stmt.(*ast.ExpressionStatement); ok {
				return c.Compile(es.Expression)
			}
		}
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileBlockLastReturn(block *ast.BlockStatement) error {
	return c.compileStatements(block.Statements, false)
}

func (c *Compiler) compileFnBody(body *ast.BlockStatement) error {
	if err := c.compileBlockLastReturn(body); err != nil {
		return err
	}
	if !c.lastIsReturn() {
		c.emit(OpReturnValue)
	}
	return nil
}

func (c *Compiler) compileMatch(me *ast.MatchExpression) error {
	if err := c.Compile(me.Value); err != nil {
		return err
	}

	if len(me.Cases) == 0 {
		c.emit(OpPop)
		c.emit(OpNil)
		return nil
	}

	jumpsToEnd := []int{}

	for _, cs := range me.Cases {
		if ident, ok := cs.Pattern.(*ast.Identifier); ok && ident.Value == "_" {
			c.emit(OpPop) // pop match value
			if err := c.Compile(cs.Body); err != nil {
				return err
			}
			jumpsToEnd = append(jumpsToEnd, c.emit(OpJump, 9999))
			break
		}

		c.emit(OpDup)
		if err := c.Compile(cs.Pattern); err != nil {
			return err
		}
		c.emit(OpEqual)
		jumpNotMatchPos := c.emit(OpJumpNotTruthy, 9999)

		c.emit(OpPop) // pop dup of match value

		if err := c.Compile(cs.Body); err != nil {
			return err
		}
		jumpsToEnd = append(jumpsToEnd, c.emit(OpJump, 9999))

		afterBody := len(c.currentInstructions())
		c.patchJump(jumpNotMatchPos, afterBody)
	}

	// Default: pop match value, push nil
	c.emit(OpPop)
	c.emit(OpNil)

	afterMatch := len(c.currentInstructions())
	for _, jmp := range jumpsToEnd {
		c.patchJump(jmp, afterMatch)
	}

	return nil
}

func (c *Compiler) compileTryExpression(te *ast.TryExpression) error {
	catchSym := c.symbolTable.Define(te.CatchParam.Value)

	if err := c.compileStatements(te.TryBlock.Statements, false); err != nil {
		return err
	}

	c.emit(OpCheckError)

	if te.AIFix {
		skipFixPos := c.emit(OpJumpNotTruthy, 9999)
		c.emit(OpPop)
		src := blockSourceFromStatements(te.TryBlock.Statements)
		c.emit(OpConstant, c.addConstant(&object.String{Value: src}))
		c.emit(OpTryAIFix)
		c.emit(OpCheckError)
		afterFix := len(c.currentInstructions())
		c.patchJump(skipFixPos, afterFix)
	}

	skipCatchPos := c.emit(OpJumpNotTruthy, 9999)

	if catchSym.Scope == GlobalScope {
		c.emit(OpSetGlobal, catchSym.Index)
	} else {
		c.emit(OpSetLocal, catchSym.Index)
	}
	c.emit(OpPop)

	for _, stmt := range te.CatchBlock.Statements[:len(te.CatchBlock.Statements)-1] {
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	if len(te.CatchBlock.Statements) > 0 {
		last := te.CatchBlock.Statements[len(te.CatchBlock.Statements)-1]
		if es, ok := last.(*ast.ExpressionStatement); ok {
			if err := c.Compile(es.Expression); err != nil {
				return err
			}
		} else {
			if err := c.Compile(last); err != nil {
				return err
			}
			c.emit(OpPop)
		}
	}

	endPos := c.emit(OpJump, 9999)

	afterCatch := len(c.currentInstructions())
	c.patchJump(skipCatchPos, afterCatch)

	c.patchJump(endPos, len(c.currentInstructions()))

	return nil
}

func blockSourceFromStatements(stmts []ast.Statement) string {
	if len(stmts) == 0 {
		return "(empty)"
	}
	if len(stmts) == 1 {
		if es, ok := stmts[0].(*ast.ExpressionStatement); ok {
			s := es.Expression.String()
			if len(s) > 2 && s[0] == '(' && s[len(s)-1] == ')' {
				s = s[1 : len(s)-1]
			}
			return s
		}
	}
	var parts []string
	for _, stmt := range stmts {
		parts = append(parts, fmt.Sprintf("%s", stmt))
	}
	return strings.Join(parts, "; ")
}

func (c *Compiler) compilePipeline(pe *ast.PipelineExpression) error {
	callOp := OpCall
	if pe.Parallel {
		callOp = OpSpawn
	}

	switch right := pe.Right.(type) {
	case *ast.Identifier:
		sym, ok := c.symbolTable.Resolve(right.Value)
		if !ok {
			return fmt.Errorf("undefined function: %s", right.Value)
		}
		c.loadSymbol(sym)
		if err := c.Compile(pe.Left); err != nil {
			return err
		}
		c.emit(callOp, 1)

	case *ast.CallExpression:
		if err := c.Compile(right.Function); err != nil {
			return err
		}
		if err := c.Compile(pe.Left); err != nil {
			return err
		}
		for _, arg := range right.Arguments {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}
		c.emit(callOp, len(right.Arguments)+1)

	default:
		if err := c.Compile(pe.Right); err != nil {
			return err
		}
		if err := c.Compile(pe.Left); err != nil {
			return err
		}
		c.emit(callOp, 1)
	}

	return nil
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(OpGetFree, s.Index)
	}
}

func (c *Compiler) enterLoop(continueTarget int) {
	c.loopStack = append(c.loopStack, LoopContext{
		continueTarget: continueTarget,
		breakPatches:   []int{},
	})
}

// enterLoopPending registers a loop whose continue target (the update clause)
// is only known after the body compiles. Continues are emitted as patchable
// forward jumps and resolved with patchContinues.
func (c *Compiler) enterLoopPending() {
	c.loopStack = append(c.loopStack, LoopContext{
		continueTarget:  -1,
		continuePatches: []int{},
		breakPatches:    []int{},
	})
}

func (c *Compiler) leaveLoop() {
	c.loopStack = c.loopStack[:len(c.loopStack)-1]
}

func (c *Compiler) addBreak() {
	if len(c.loopStack) == 0 {
		return
	}
	loop := &c.loopStack[len(c.loopStack)-1]
	loop.breakPatches = append(loop.breakPatches, c.emit(OpJump, 9999))
}

func (c *Compiler) patchBreaks(afterLoop int) {
	if len(c.loopStack) == 0 {
		return
	}
	loop := &c.loopStack[len(c.loopStack)-1]
	for _, pos := range loop.breakPatches {
		c.patchJump(pos, afterLoop)
	}
	loop.breakPatches = nil
}

func (c *Compiler) emitContinue() {
	if len(c.loopStack) == 0 {
		return
	}
	loop := &c.loopStack[len(c.loopStack)-1]
	if loop.continueTarget < 0 {
		loop.continuePatches = append(loop.continuePatches, c.emit(OpJump, 9999))
		return
	}
	c.emit(OpJumpBackward, loop.continueTarget)
}

func (c *Compiler) patchContinues(target int) {
	if len(c.loopStack) == 0 {
		return
	}
	loop := &c.loopStack[len(c.loopStack)-1]
	for _, pos := range loop.continuePatches {
		c.patchJump(pos, target)
	}
	loop.continuePatches = nil
}

func (c *Compiler) compileForIn(fe *ast.ForExpression) error {
	iterSym := c.symbolTable.Define(fe.Iterator.Value)
	listSym := c.symbolTable.Define("__list__")
	idxSym := c.symbolTable.Define("__idx__")

	if err := c.Compile(fe.Iterable); err != nil {
		return err
	}
	c.emitSet(listSym)

	c.emit(OpConstant, c.addInteger(0))
	c.emitSet(idxSym)

	loopStart := len(c.currentInstructions())

	// idx < len(list): push idx, then builtin len, then list
	c.emitGet(idxSym)
	c.emit(OpGetBuiltin, c.resolveBuiltin("len").Index)
	c.emitGet(listSym)
	c.emit(OpCall, 1)
	c.emit(OpLess)
	jumpFalsePos := c.emit(OpJumpNotTruthy, 9999)

	// list[idx]: push builtin at, then args
	c.emit(OpGetBuiltin, c.resolveBuiltin("at").Index)
	c.emitGet(listSym)
	c.emitGet(idxSym)
	c.emit(OpCall, 2)
	c.emitSet(iterSym)

	c.enterLoop(loopStart)
	if err := c.compileBlockLastReturn(fe.Body); err != nil {
		return err
	}
	c.leaveLoop()

	c.emitGet(idxSym)
	c.emit(OpConstant, c.addInteger(1))
	c.emit(OpAdd)
	c.emitSet(idxSym)

	c.emit(OpJumpBackward, loopStart)

	afterLoop := len(c.currentInstructions())
	c.patchJump(jumpFalsePos, afterLoop)
	c.patchBreaks(afterLoop)

	return nil
}

func (c *Compiler) compileCStyleFor(fe *ast.ForExpression) error {
	// Init clause: for i: 0; cond; update
	if fe.Init != nil {
		if err := c.Compile(fe.Init); err != nil {
			return err
		}
	}

	conditionPos := len(c.currentInstructions())
	if fe.Condition != nil {
		if err := c.Compile(fe.Condition); err != nil {
			return err
		}
	} else {
		c.emit(OpConstant, c.addInteger(1)) // no condition → infinite loop
	}
	jumpFalsePos := c.emit(OpJumpNotTruthy, 9999)

	c.enterLoopPending()
	if err := c.compileBlockLastReturn(fe.Body); err != nil {
		return err
	}

	updatePos := len(c.currentInstructions())
	if fe.Update != nil {
		if err := c.Compile(fe.Update); err != nil {
			return err
		}
	}
	c.emit(OpJumpBackward, conditionPos)

	afterLoop := len(c.currentInstructions())
	c.patchJump(jumpFalsePos, afterLoop)
	c.patchContinues(updatePos)
	c.patchBreaks(afterLoop)
	c.leaveLoop()

	return nil
}

func (c *Compiler) emitGet(s Symbol) {
	if s.Scope == GlobalScope {
		c.emit(OpGetGlobal, s.Index)
	} else {
		c.emit(OpGetLocal, s.Index)
	}
}

func (c *Compiler) emitSet(s Symbol) {
	if s.Scope == GlobalScope {
		c.emit(OpSetGlobal, s.Index)
	} else {
		c.emit(OpSetLocal, s.Index)
	}
}

func (c *Compiler) resolveBuiltin(name string) Symbol {
	for i, bi := range object.Builtins {
		if bi.Name == name {
			return Symbol{Name: name, Scope: BuiltinScope, Index: i}
		}
	}
	return Symbol{}
}

func (c *Compiler) patchJump(pos int, target int) {
	ins := c.currentScope().instructions
	ins[pos+1] = byte(target >> 8)
	ins[pos+2] = byte(target)
}

func (c *Compiler) lastIsReturn() bool {
	ins := c.currentScope().instructions
	if len(ins) < 1 {
		return false
	}
	last := Opcode(ins[len(ins)-1])
	return last == OpReturn || last == OpReturnValue
}

func (c *Compiler) compileDefer(de *ast.DeferStatement) error {
	prevLen := len(c.currentScope().instructions)
	c.currentScope().instructions = c.currentScope().instructions[:prevLen]

	if err := c.Compile(de.Expression); err != nil {
		return err
	}

	deferCode := make(Instructions, len(c.currentScope().instructions)-prevLen)
	copy(deferCode, c.currentScope().instructions[prevLen:])
	c.currentScope().instructions = c.currentScope().instructions[:prevLen]

	c.currentScope().deferred = append(c.currentScope().deferred, deferCode)
	return nil
}

func (c *Compiler) compileImport(is *ast.ImportStatement) error {
	cacheKey := is.Path

	// Flat imports: skip if already imported (deduplication)
	if is.Alias == "" {
		if _, ok := c.importedSet[cacheKey]; ok {
			return nil
		}
	}

	// Use cached parse result or parse fresh
	program, ok := c.importCache[cacheKey]
	if !ok {
		var resolvedPath string
		var err error
		resolvedPath, program, err = object.ResolveAndParse(is.Path, c.sourceFile)
		if err != nil {
			return fmt.Errorf("import %s: %w", is.Path, err)
		}
		// Use resolved path as cache key (may differ from is.Path)
		cacheKey = resolvedPath
		c.importCache[cacheKey] = program
	}

	if is.Alias == "" {
		c.importedSet[cacheKey] = struct{}{}
	}

	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *ast.FnStatement:
			if err := c.Compile(s); err != nil {
				return fmt.Errorf("import %s: %w", is.Path, err)
			}
		case *ast.VarStatement:
			if err := c.Compile(s); err != nil {
				return fmt.Errorf("import %s: %w", is.Path, err)
			}
		case *ast.EnumStatement:
			if err := c.Compile(s); err != nil {
				return fmt.Errorf("import %s: %w", is.Path, err)
			}
		case *ast.StructStatement:
			if err := c.Compile(s); err != nil {
				return fmt.Errorf("import %s: %w", is.Path, err)
			}
		case *ast.ExportStatement:
			if s.Fn != nil {
				if err := c.Compile(s.Fn); err != nil {
					return fmt.Errorf("import %s: %w", is.Path, err)
				}
			}
			if s.Var != nil {
				if err := c.Compile(s.Var); err != nil {
					return fmt.Errorf("import %s: %w", is.Path, err)
				}
			}
			if s.Enum != nil {
				if err := c.Compile(s.Enum); err != nil {
					return fmt.Errorf("import %s: %w", is.Path, err)
				}
			}
		case *ast.ExpressionStatement:
			// skip standalone expressions in imports
		}
	}
	return nil
}

func (c *Compiler) resolveImport(path string) (string, *ast.Program, error) {
	return object.ResolveAndParse(path, c.sourceFile)
}

func (c *Compiler) defineTopLevelSymbols(stmts []ast.Statement) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.FnStatement:
			c.symbolTable.Define(s.Name.Value)
		case *ast.VarStatement:
			c.symbolTable.Define(s.Name.Value)
		case *ast.EnumStatement:
			for _, name := range s.Values {
				c.symbolTable.Define(name)
			}
		case *ast.StructStatement:
			c.symbolTable.Define(s.Name)
		case *ast.ExportStatement:
			if s.Fn != nil {
				c.symbolTable.Define(s.Fn.Name.Value)
			}
			if s.Var != nil {
				c.symbolTable.Define(s.Var.Name.Value)
			}
			if s.Enum != nil {
				for _, name := range s.Enum.Values {
					c.symbolTable.Define(name)
				}
			}
		case *ast.ImportStatement:
			_, program, err := c.resolveImport(s.Path)
			if err == nil {
				c.defineTopLevelSymbols(program.Statements)
			}
		}
	}
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{instructions: Instructions{}, deferred: nil}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() CompiledScope {
	scope := c.scopes[c.scopeIndex]

	// Emit all deferred expressions in LIFO order
	for i := len(scope.deferred) - 1; i >= 0; i-- {
		scope.instructions = append(scope.instructions, scope.deferred[i]...)
	}

	nl := c.symbolTable.numDefinitions
	free := c.symbolTable.FreeSymbols
	c.scopes = c.scopes[:c.scopeIndex]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer
	return CompiledScope{Instructions: scope.instructions, NumLocals: nl, FreeSymbols: free}
}

// Pretty-print instructions for debugging
func (ins Instructions) String() string {
	var out string
	i := 0
	for i < len(ins) {
		op := Opcode(ins[i])
		switch op {
		case OpClosure:
			out += fmt.Sprintf("%04d %-14s %d %d\n", i, op, ReadUint16(ins, i+1), ReadUint16(ins, i+3))
			i += 5
		case OpConstant, OpGetGlobal, OpSetGlobal, OpGetLocal, OpSetLocal,
			OpGetBuiltin, OpDot, OpGetFree:
			out += fmt.Sprintf("%04d %-14s %d\n", i, op, ReadUint16(ins, i+1))
			i += 3
		case OpCall, OpList:
			out += fmt.Sprintf("%04d %-14s %d\n", i, op, ReadUint16(ins, i+1))
			i += 3
		case OpMap:
			numPairs := int(ReadUint16(ins, i+1))
			out += fmt.Sprintf("%04d %-14s %d pairs\n", i, op, numPairs)
			i += 3
			for j := 0; j < numPairs; j++ {
				ki := ReadUint16(ins, i)
				out += fmt.Sprintf("%04d   key=%d\n", i, ki)
				i += 2
			}
		case OpJump, OpJumpNotTruthy, OpJumpBackward:
			out += fmt.Sprintf("%04d %-14s %d\n", i, op, ReadUint16(ins, i+1))
			i += 3
		default:
			out += fmt.Sprintf("%04d %s\n", i, op)
			i++
		}
	}
	return out
}

func ParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

var zeroArityBuiltins = map[string]bool{
	"now":              true,
	"random":           true,
	"secure_random_int": true,
	"try_ai_log":       true,
	"args":            true,
	"read_stdin":      true,
	"ai_cost":         true,
	"ai_tokens":       true,
	"ai_cache_hits":   true,
	"ai_cache_misses": true,
	"audit_log":       true,
	"budget_spent":    true,
}

func isZeroArityBuiltin(name string) bool {
	return zeroArityBuiltins[name]
}
