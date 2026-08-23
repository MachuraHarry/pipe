package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/lexer"
)

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

const (
	_ int = iota
	PrecedenceLowest
	PrecedenceOr
	PrecedenceAnd
	PrecedencePipeline
	PrecedenceEquals
	PrecedenceCompare
	PrecedenceSum
	PrecedenceProduct
	PrecedencePower
	PrecedenceConcat
	PrecedencePrefix
	PrecedenceCall
	PrecedenceDot
)

var precedences = map[lexer.TokenType]int{
	lexer.OR:       PrecedenceOr,
	lexer.AND:      PrecedenceAnd,
	lexer.ARROW:    PrecedencePipeline,
	lexer.ARROW2:   PrecedencePipeline,
	lexer.EQ:       PrecedenceEquals,
	lexer.NOT_EQ:   PrecedenceEquals,
	lexer.LT:       PrecedenceCompare,
	lexer.GT:       PrecedenceCompare,
	lexer.LTE:      PrecedenceCompare,
	lexer.GTE:      PrecedenceCompare,
	lexer.PLUS:     PrecedenceSum,
	lexer.MINUS:    PrecedenceSum,
	lexer.STAR:     PrecedenceProduct,
	lexer.SLASH:    PrecedenceProduct,
	lexer.PERCENT:  PrecedenceProduct,
	lexer.POWER:    PrecedencePower,
	lexer.CONCAT:   PrecedenceConcat,
	lexer.DOT:      PrecedenceDot,
	lexer.LBRACKET: PrecedenceCall,
}

type Parser struct {
	l          *lexer.Lexer
	curToken   lexer.Token
	peekToken  lexer.Token
	peekToken2 lexer.Token
	errors     []string

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = map[lexer.TokenType]prefixParseFn{
		lexer.IDENT:    p.parseIdentifier,
		lexer.INT:      p.parseIntegerLiteral,
		lexer.FLOAT:    p.parseFloatLiteral,
		lexer.STRING:   p.parseStringLiteral,
		lexer.TRUE:     p.parseBooleanLiteral,
		lexer.FALSE:    p.parseBooleanLiteral,
		lexer.NIL:      p.parseNilLiteral,
		lexer.MINUS:    p.parsePrefixExpression,
		lexer.BANG:     p.parsePrefixExpression,
		lexer.LPAREN:   p.parseGroupedExpression,
		lexer.LBRACKET: p.parseListLiteral,
		lexer.LBRACE:   p.parseMapLiteral,
		lexer.IF:       p.parseIfExpression,
		lexer.MATCH_KW: p.parseMatchExpression,
		lexer.SELECT:   p.parseSelectExpression,
		lexer.WHILE:    p.parseWhileExpression,
		lexer.FOR:      p.parseForExpression,
		lexer.FN:       p.parseFnLiteral,
		lexer.TRY:      p.parseTryExpression,
		lexer.TRYAI:    p.parseTryAIExpression,
	}

	p.infixParseFns = map[lexer.TokenType]infixParseFn{
		lexer.PLUS:     p.parseInfixExpression,
		lexer.MINUS:    p.parseInfixExpression,
		lexer.STAR:     p.parseInfixExpression,
		lexer.SLASH:    p.parseInfixExpression,
		lexer.PERCENT:  p.parseInfixExpression,
		lexer.POWER:    p.parseInfixExpression,
		lexer.EQ:       p.parseInfixExpression,
		lexer.NOT_EQ:   p.parseInfixExpression,
		lexer.LT:       p.parseInfixExpression,
		lexer.ARROW:    p.parsePipelineExpression,
		lexer.ARROW2:   p.parseParallelPipelineExpression,
		lexer.GT:       p.parseInfixExpression,
		lexer.LTE:      p.parseInfixExpression,
		lexer.GTE:      p.parseInfixExpression,
		lexer.CONCAT:   p.parseInfixExpression,
		lexer.AND:      p.parseInfixExpression,
		lexer.OR:       p.parseInfixExpression,
		lexer.DOT:      p.parseDotExpression,
		lexer.LPAREN:   p.parseCallExpression,
		lexer.LBRACKET: p.parseIndexOrSlice,
	}

	p.nextToken()
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) error(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("line %d col %d: %s", p.curToken.Line, p.curToken.Col, msg))
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.peekToken2
	p.peekToken2 = p.l.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) peekTokenIs2(t lexer.TokenType) bool {
	return p.peekToken2.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	var hint string
	expected := lexer.TokenName(t)
	got := lexer.TokenName(p.peekToken.Type)
	switch t {
	case lexer.NEWLINE:
		hint = fmt.Sprintf("expected a line break here, got %s", got)
	case lexer.INDENT:
		hint = fmt.Sprintf("expected an indented block here, got %s", got)
	case lexer.COLON:
		hint = fmt.Sprintf("expected ':', got %s (%q)", got, p.peekToken.Literal)
	case lexer.ARROW, lexer.ARROW2:
		hint = fmt.Sprintf("expected '%s', got %s (%q)", expected, got, p.peekToken.Literal)
	case lexer.RPAREN:
		hint = fmt.Sprintf("expected ')', got %s (%q) — did you forget to close a parenthesis?", got, p.peekToken.Literal)
	case lexer.RBRACKET:
		hint = fmt.Sprintf("expected ']', got %s (%q) — did you forget to close a bracket?", got, p.peekToken.Literal)
	case lexer.RBRACE:
		hint = fmt.Sprintf("expected '}', got %s (%q) — did you forget to close a brace?", got, p.peekToken.Literal)
	case lexer.DEDENT:
		hint = fmt.Sprintf("expected block end (dedent), got %s", got)
	case lexer.LBRACE:
		hint = fmt.Sprintf("expected '{', got %s (%q)", got, p.peekToken.Literal)
	default:
		if p.peekToken.Type == lexer.NEWLINE {
			hint = fmt.Sprintf("expected %s, but the line ended here — did you forget something?", expected)
		} else {
			hint = fmt.Sprintf("expected %s, got %s (%q)", expected, got, p.peekToken.Literal)
		}
	}
	p.errors = append(p.errors, fmt.Sprintf(
		"line %d col %d: %s",
		p.peekToken.Line, p.peekToken.Col, hint))
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	var hint string
	switch t {
	case lexer.NEWLINE:
		hint = "unexpected line break — expected an expression. Did you forget to complete a value or add an operator?"
	case lexer.COLON:
		hint = "unexpected ':' — a colon starts a variable assignment (x: value) or a map entry ({key: value}), but cannot appear here"
	case lexer.ARROW:
		hint = "unexpected '>' — did you mean a pipeline? If so, indent the next line:\n    > your_function"
	case lexer.ARROW2:
		hint = "unexpected '>>' — did you mean a parallel pipeline? If so, indent:\n    >> your_function"
	case lexer.PIPE:
		hint = "unexpected '|' — Pipe uses '>' for pipelines, not '|'"
	case lexer.INDENT:
		hint = "unexpected indentation increase — this block is not expected here"
	case lexer.DEDENT:
		hint = "unexpected dedent — check that all blocks are properly aligned"
	case lexer.ILLEGAL:
		switch p.curToken.Literal {
		case "@":
			hint = "'@' is not a valid character in Pipe"
		case "&":
			hint = "unexpected '&' — did you mean 'and' (&&)?"
		case "unterminated string":
			hint = "string is missing the closing quote (\"...\")"
		case "unterminated backtick string":
			hint = "backtick string is missing the closing backtick (`...`)"
		case "unexpected indent":
			hint = "unexpected indentation increase — check that your indentation is consistent"
		default:
			if len(p.curToken.Literal) == 1 {
				hint = fmt.Sprintf("invalid character %q", p.curToken.Literal)
			} else {
				hint = fmt.Sprintf("%s", p.curToken.Literal)
			}
		}
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.IDENT,
		lexer.TRUE, lexer.FALSE, lexer.NIL, lexer.LPAREN,
		lexer.LBRACKET, lexer.LBRACE, lexer.IF, lexer.FN,
		lexer.MATCH_KW, lexer.SELECT, lexer.WHILE, lexer.FOR, lexer.MINUS, lexer.BANG:
		hint = fmt.Sprintf("unexpected '%s' here — check your expression order", lexer.TokenName(t))
	default:
		hint = fmt.Sprintf("unexpected token '%s'", p.curToken.Literal)
	}
	p.errors = append(p.errors,
		fmt.Sprintf("line %d col %d: %s",
			p.curToken.Line, p.curToken.Col, hint))
}

// ---- Entry points ----

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.FN:
		return p.parseFnStatement()
	case lexer.WHILE:
		return p.parseWhileStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.BREAK:
		return &ast.BreakStatement{}
	case lexer.CONTINUE:
		return &ast.ContinueStatement{}
	case lexer.IMPORT:
		return p.parseImportStatement()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.DEFER:
		return p.parseDeferStatement()
	case lexer.EXPORT:
		return p.parseExportStatement()
	case lexer.ENUM:
		return p.parseEnumStatement()
	case lexer.STRUCT:
		return p.parseStructStatement()
	case lexer.TEST:
		return p.parseTestStatement()
	case lexer.NEWLINE:
		return nil
	case lexer.DEDENT:
		return nil
	case lexer.INDENT:
		return nil
	default:
		return p.parseExpressionOrVarStatement()
	}
}

func (p *Parser) parseExpressionOrVarStatement() ast.Statement {
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
		return p.parseVarStatement()
	}
	// Common mistake: `name = value` instead of `name: value`.
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.ASSIGN) {
		p.error(fmt.Sprintf("use ':' instead of '=' for assignment: '%s'", p.curToken.Literal))
		return p.parseVarStatement()
	}
	// Compound assignment: x += expr
	if p.curTokenIs(lexer.IDENT) && isCompoundAssign(p.peekToken.Type) {
		return p.parseCompoundAssign()
	}

	stmt := &ast.ExpressionStatement{}
	stmt.Expression = p.parseExpression(PrecedenceLowest)

	p.tryPipelineContinuation(&stmt.Expression)

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) tryPipelineContinuation(expr *ast.Expression) {
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
		if p.peekTokenIs(lexer.INDENT) {
			p.nextToken()
			if p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.ARROW2) {
				for p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.ARROW2) || p.peekTokenIs(lexer.PIPE) {
					if p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.ARROW2) {
						parallel := p.peekTokenIs(lexer.ARROW2)
						p.nextToken()
						p.nextToken()
						right := p.parseExpression(PrecedenceCall)
						*expr = &ast.PipelineExpression{
							Left:     *expr,
							Right:    p.insertPipelinePlaceholder(right, *expr),
							Parallel: parallel,
						}
					} else if p.peekTokenIs(lexer.PIPE) {
						p.nextToken()
						p.nextToken()
					}
					if p.peekTokenIs(lexer.NEWLINE) {
						p.nextToken()
					}
				}
				if p.peekTokenIs(lexer.DEDENT) {
					p.nextToken()
				}
			}
		}
	}
}

func isCompoundAssign(t lexer.TokenType) bool {
	switch t {
	case lexer.PLUSEQ, lexer.MINUSEQ, lexer.STAREQ, lexer.SLASHEQ, lexer.PERCENTEQ:
		return true
	}
	return false
}

func (p *Parser) parseCompoundAssign() ast.Statement {
	stmt := &ast.VarStatement{}
	stmt.Name = &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}

	opToken := p.peekToken
	op := ""
	switch opToken.Type {
	case lexer.PLUSEQ:
		op = "+"
	case lexer.MINUSEQ:
		op = "-"
	case lexer.STAREQ:
		op = "*"
	case lexer.SLASHEQ:
		op = "/"
	case lexer.PERCENTEQ:
		op = "%"
	}

	p.nextToken() // skip to +=
	p.nextToken() // skip to value

	rightExpr := p.parseExpression(PrecedenceLowest)
	stmt.Value = &ast.InfixExpression{
		Operator: op,
		Left:     stmt.Name,
		Right:    rightExpr,
	}
	return stmt
}

func (p *Parser) parseVarStatement() *ast.VarStatement {
	stmt := &ast.VarStatement{}
	stmt.Name = &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}

	p.nextToken() // skip name
	p.nextToken() // skip ':'

	// Type annotation: name: type = value
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.ASSIGN) {
		stmt.TypeAnnotation = &ast.TypeAnnotation{
			Name: p.curToken.Literal,
			Line: p.curToken.Line,
			Col:  p.curToken.Col,
		}
		p.nextToken() // skip type name
		p.nextToken() // skip '='
	}

	stmt.Value = p.parseExpression(PrecedenceLowest)

	p.tryPipelineContinuation(&stmt.Value)

	return stmt
}

func (p *Parser) parseFnStatement() ast.Statement {
	p.nextToken() // skip 'fn'

	if !p.curTokenIs(lexer.IDENT) {
		p.error("expected function name or parameter after 'fn'")
		return nil
	}

	firstLine := p.curToken.Line
	firstCol := p.curToken.Col

	var idents []*ast.Identifier
	for p.curTokenIs(lexer.IDENT) {
		idents = append(idents, &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col})
		p.nextToken()
	}

	// Inline lambda: fn param...: expression
	if p.curTokenIs(lexer.COLON) && len(idents) > 0 {
		// Check if this is typed params: fn name(param: type, ...) -> type
		// vs inline lambda: fn x: expr or fn a b: expr
		// If we have exactly 1 ident and peek is LPAREN, it's typed params
		if len(idents) == 1 && p.peekTokenIs(lexer.LPAREN) {
			// fall through to named function with typed params
		} else {
			p.nextToken() // skip colon
			expr := p.parseExpression(PrecedenceLowest)
			if expr == nil {
				return nil
			}
			return &ast.ExpressionStatement{
				Expression: &ast.FnLiteral{
					Parameters: idents,
					Body: &ast.BlockStatement{
						Statements: []ast.Statement{
							&ast.ExpressionStatement{Expression: expr},
						},
					},
					Line: firstLine,
					Col:  firstCol,
				},
			}
		}
	}

	// Named function with typed params: fn name(param: type, ...) -> type
	if len(idents) > 0 && p.curTokenIs(lexer.LPAREN) {
		name := idents[0]
		params, paramTypes := p.parseTypedParams()
		var retType *ast.TypeAnnotation
		if p.curTokenIs(lexer.FAT_ARROW) {
			p.nextToken() // skip ->
			if p.curTokenIs(lexer.IDENT) {
				retType = &ast.TypeAnnotation{Name: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}
				p.nextToken()
			}
		}
		stmt := &ast.FnStatement{
			Name:       name,
			Parameters: params,
			ParamTypes: paramTypes,
			ReturnType: retType,
		}
		stmt.Body = p.parseBlock()
		return stmt
	}

	// Named function: first ident is name, rest are params (existing syntax)
	if len(idents) == 0 {
		p.error("expected function name after 'fn'")
		return nil
	}
	stmt := &ast.FnStatement{
		Name:       idents[0],
		Parameters: idents[1:],
	}
	stmt.Body = p.parseBlock()
	return stmt
}

func (p *Parser) parseTypedParams() ([]*ast.Identifier, []*ast.TypeAnnotation) {
	p.nextToken() // skip '('
	var params []*ast.Identifier
	var types []*ast.TypeAnnotation

	for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENT) {
			p.error("expected parameter name")
			break
		}
		param := &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}
		params = append(params, param)
		p.nextToken()

		// Optional type annotation: param: type
		var typ *ast.TypeAnnotation
		if p.curTokenIs(lexer.COLON) {
			p.nextToken() // skip ':'
			if p.curTokenIs(lexer.IDENT) {
				typ = &ast.TypeAnnotation{Name: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}
				p.nextToken()
			}
		}
		types = append(types, typ)

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // skip ','
		}
	}

	if p.curTokenIs(lexer.RPAREN) {
		p.nextToken() // skip ')'
	}
	return params, types
}

func (p *Parser) parseParameters() []*ast.Identifier {
	var params []*ast.Identifier

	for p.curTokenIs(lexer.IDENT) && !p.peekTokenIs(lexer.COLON) {
		params = append(params, &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col})
		p.nextToken()
	}

	return params
}

func (p *Parser) parseBlock() *ast.BlockStatement {
	block := &ast.BlockStatement{}
	block.Statements = []ast.Statement{}

	if p.curTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	if !p.curTokenIs(lexer.INDENT) {
		expr := p.parseExpression(PrecedenceLowest)
		if expr != nil {
			block.Statements = append(block.Statements, &ast.ExpressionStatement{Expression: expr})
		}
		return block
	}

	p.nextToken()

	for !p.curTokenIs(lexer.DEDENT) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// ---- Expression parsing ----

func (p *Parser) parseExpression(precedence int) ast.Expression {
	return p.parseExpr(precedence, true)
}

func (p *Parser) parseExpressionNoSpace(precedence int) ast.Expression {
	return p.parseExpr(precedence, false)
}

func (p *Parser) parseExpr(precedence int, allowSpaceCalls bool) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	// FnLiteral: don't continue parsing (it's self-contained with its own block)
	if _, ok := leftExp.(*ast.FnLiteral); ok {
		return leftExp
	}

	// TryExpression: don't continue (self-contained)
	if _, ok := leftExp.(*ast.TryExpression); ok {
		return leftExp
	}

	// IfExpression, WhileExpression, MatchExpression, ForExpression: don't continue (self-contained blocks)
	if _, ok := leftExp.(*ast.IfExpression); ok {
		return leftExp
	}
	if _, ok := leftExp.(*ast.WhileExpression); ok {
		return leftExp
	}
	if _, ok := leftExp.(*ast.MatchExpression); ok {
		return leftExp
	}
	if _, ok := leftExp.(*ast.ForExpression); ok {
		return leftExp
	}

	for {
		if allowSpaceCalls &&
			!p.peekTokenIs(lexer.NEWLINE) &&
			!p.peekTokenIs(lexer.DEDENT) &&
			!p.peekTokenIs(lexer.EOF) &&
			!p.peekTokenIs(lexer.INDENT) &&
			p.peekStartsCallArg() {
			call := &ast.CallExpression{Function: leftExp}

			for !p.peekTokenIs(lexer.NEWLINE) &&
				!p.peekTokenIs(lexer.DEDENT) &&
				!p.peekTokenIs(lexer.EOF) &&
				!p.peekTokenIs(lexer.INDENT) &&
				p.peekStartsCallArg() {
				p.nextToken()
				arg := p.parseExpressionNoSpace(PrecedenceSum)
				call.Arguments = append(call.Arguments, arg)
			}

			leftExp = call
			continue
		}

		if !p.peekTokenIs(lexer.NEWLINE) &&
			!p.peekTokenIs(lexer.DEDENT) &&
			!p.peekTokenIs(lexer.EOF) &&
			!p.peekTokenIs(lexer.INDENT) &&
			precedence < p.peekPrecedence() {
			infix := p.infixParseFns[p.peekToken.Type]
			if infix != nil {
				p.nextToken()
				leftExp = infix(leftExp)
				continue
			}
		}

		break
	}

	return leftExp
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return PrecedenceLowest
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return PrecedenceLowest
}

func isValueToken(t lexer.TokenType) bool {
	switch t {
	case lexer.IDENT, lexer.INT, lexer.FLOAT, lexer.STRING,
		lexer.TRUE, lexer.FALSE, lexer.NIL,
		lexer.LPAREN, lexer.LBRACE:
		return true
	}
	return false
}

// peekAdjacent reports whether the peek token starts immediately after the
// current token with no whitespace in between (same line).
func (p *Parser) peekAdjacent() bool {
	return p.peekToken.Line == p.curToken.Line &&
		p.peekToken.Col == p.curToken.Col+len(p.curToken.Literal)
}

// peekStartsCallArg reports whether the peek token should be consumed as the
// next space-separated call argument. A '[' qualifies only when whitespace
// separates it from the current token: an adjacent '[' is an index/slice
// postfix (`xs[0]`, `xs[1:3]`), while `map [1, 2] f` passes a fresh list
// literal as the first argument.
func (p *Parser) peekStartsCallArg() bool {
	if p.peekTokenIs(lexer.LBRACKET) {
		return !p.peekAdjacent()
	}
	return isValueToken(p.peekToken.Type)
}

// ---- Prefix parsers ----

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{}
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.error(fmt.Sprintf("invalid number: %s", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.error(fmt.Sprintf("invalid number: %s", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Value: p.curTokenIs(lexer.TRUE)}
}

func (p *Parser) parseNilLiteral() ast.Expression {
	return &ast.NilLiteral{}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	op := p.curToken.Literal
	if op == "not" {
		op = "!"
	}
	expr := &ast.PrefixExpression{Operator: op}
	p.nextToken()
	expr.Right = p.parseExpression(PrecedencePrefix)
	return expr
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken() // skip (
	exp := p.parseExpression(PrecedenceLowest)
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{}

	p.nextToken() // skip 'if'
	expr.Condition = p.parseExpression(PrecedenceLowest)

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()

	expr.Consequence = p.parseBlock()

	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken()
		p.nextToken()
		expr.Alternative = p.parseBlock()
	}

	return expr
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expr := &ast.MatchExpression{}

	p.nextToken() // skip 'match'
	expr.Value = p.parseExpression(PrecedenceLowest)

	if !p.expectPeek(lexer.NEWLINE) {
		return nil
	}
	if !p.expectPeek(lexer.INDENT) {
		return nil
	}
	p.nextToken() // skip INDENT

	for p.curTokenIs(lexer.PIPE) {
		p.nextToken() // skip |

		// Check for binding pattern: | name: pattern -> body
		var bindName string
		if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
			bindName = p.curToken.Literal
			p.nextToken() // skip binding name
			p.nextToken() // skip ':'
		}

		// Check for list destructuring: | [a, b, ...rest] -> body
		if p.curTokenIs(lexer.LBRACKET) {
			pattern := p.parseListDestructurePattern()
			var guard ast.Expression
			if p.peekTokenIs(lexer.IF) {
				p.nextToken()
				p.nextToken()
				guard = p.parseExpression(PrecedenceLowest)
			}
			if !p.expectPeek(lexer.FAT_ARROW) {
				return nil
			}
			p.nextToken()
			body := p.parseExpression(PrecedenceLowest)
			expr.Cases = append(expr.Cases, ast.MatchCase{Pattern: pattern, Body: body, Guard: guard, Bind: bindName})
			for p.peekTokenIs(lexer.NEWLINE) {
				p.nextToken()
			}
			if p.peekTokenIs(lexer.DEDENT) {
				p.nextToken()
				break
			}
			p.nextToken()
			continue
		}

		// Check for map destructuring: | {name: n, age: a} -> body
		if p.curTokenIs(lexer.LBRACE) {
			pattern := p.parseMapDestructurePattern()
			var guard ast.Expression
			if p.peekTokenIs(lexer.IF) {
				p.nextToken()
				p.nextToken()
				guard = p.parseExpression(PrecedenceLowest)
			}
			if !p.expectPeek(lexer.FAT_ARROW) {
				return nil
			}
			p.nextToken()
			body := p.parseExpression(PrecedenceLowest)
			expr.Cases = append(expr.Cases, ast.MatchCase{Pattern: pattern, Body: body, Guard: guard, Bind: bindName})
			for p.peekTokenIs(lexer.NEWLINE) {
				p.nextToken()
			}
			if p.peekTokenIs(lexer.DEDENT) {
				p.nextToken()
				break
			}
			p.nextToken()
			continue
		}

		// Multi-pattern: | 1 | 2 | 3 -> body (each pattern shares the body)
		patterns := []ast.Expression{p.parseExpression(PrecedenceLowest)}
		for p.peekTokenIs(lexer.PIPE) {
			p.nextToken() // consume '|'
			p.nextToken() // move to next pattern
			patterns = append(patterns, p.parseExpression(PrecedenceLowest))
		}

		// Guard: | pattern if condition -> body (shared by all patterns)
		var guard ast.Expression
		if p.peekTokenIs(lexer.IF) {
			p.nextToken() // consume 'if' keyword
			p.nextToken() // move to the guard expression
			guard = p.parseExpression(PrecedenceLowest)
		}

		if !p.expectPeek(lexer.FAT_ARROW) { // ->
			return nil
		}
		p.nextToken() // skip ->
		body := p.parseExpression(PrecedenceLowest)

		for _, pattern := range patterns {
			expr.Cases = append(expr.Cases, ast.MatchCase{Pattern: pattern, Body: body, Guard: guard, Bind: bindName})
		}

		for p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
		if p.peekTokenIs(lexer.DEDENT) {
			p.nextToken()
			break
		}
		p.nextToken()
	}

	return expr
}

func (p *Parser) parseListDestructurePattern() *ast.ListDestructurePattern {
	ld := &ast.ListDestructurePattern{Line: p.curToken.Line, Col: p.curToken.Col}
	p.nextToken() // skip '['

	for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENT) {
			p.error("expected identifier in list destructuring pattern")
			break
		}
		name := p.curToken.Literal
		p.nextToken() // skip identifier

		// Check for rest: name.. (postfix dotdot)
		if p.curTokenIs(lexer.DOTDOT) {
			ld.Rest = name
			p.nextToken() // skip ..
			if p.curTokenIs(lexer.COMMA) {
				p.nextToken() // skip trailing comma
			}
			break
		}

		ld.Elements = append(ld.Elements, &ast.Identifier{Value: name, Line: p.curToken.Line, Col: p.curToken.Col})

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // skip ','
		}
	}

	if p.curTokenIs(lexer.RBRACKET) {
		// Don't consume ']' — caller will advance
	}
	return ld
}

func (p *Parser) parseMapDestructurePattern() *ast.MapDestructurePattern {
	md := &ast.MapDestructurePattern{Line: p.curToken.Line, Col: p.curToken.Col}
	p.nextToken() // skip '{'

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENT) {
			p.error("expected key name in map destructuring pattern")
			break
		}
		key := &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}
		p.nextToken() // skip key

		if !p.curTokenIs(lexer.COLON) {
			p.error("expected ':' after key in map destructuring pattern")
			break
		}
		p.nextToken() // skip ':'

		if !p.curTokenIs(lexer.IDENT) {
			p.error("expected variable name after ':' in map destructuring pattern")
			break
		}
		val := &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}
		md.Keys = append(md.Keys, key)
		md.Values = append(md.Values, val)
		p.nextToken() // skip value name

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // skip ','
		}
	}

	if p.curTokenIs(lexer.RBRACE) {
		// Don't consume '}' — caller will advance
	}
	return md
}

func (p *Parser) parseSelectExpression() ast.Expression {
	expr := &ast.SelectExpression{Line: p.curToken.Line, Col: p.curToken.Col}
	p.nextToken() // skip 'select'

	// After 'select', curToken is at NEWLINE
	if p.curTokenIs(lexer.NEWLINE) {
		p.nextToken() // skip NEWLINE
	}

	if !p.curTokenIs(lexer.INDENT) {
		p.error("expected indented block after 'select'")
		return nil
	}
	p.nextToken() // skip INDENT

	for p.curTokenIs(lexer.PIPE) {
		p.nextToken() // skip |

		sc := &ast.SelectCase{}

		if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "default" {
			sc.IsDefault = true
		} else {
			// Parse channel case: [var <-] channel -> body
			// Check for variable binding: var <- channel
			if p.peekTokenIs(lexer.PIPE) || p.peekTokenIs(lexer.NEWLINE) || p.peekTokenIs(lexer.DEDENT) {
				// This is tricky - we need to distinguish between:
				// | channel -> body  (receive only)
				// | var <- channel -> body  (receive with binding)
				// | channel <- value -> body  (send)
				// Let's parse the first expression, then check for <-
			}
			leftExpr := p.parseExpression(PrecedenceLowest)
			if leftExpr == nil {
				return nil
			}

			// Check for <- (receive or send)
			if p.peekTokenIs(lexer.ASSIGN) {
				// Could be: channel <- value (send) or var <- channel (receive with binding)
				// Actually <-  isn't a token in Pipe. We use arrow syntax.
				// Let's use: recv var from channel, or channel <- value
				// For now: treat as channel expression
				sc.Channel = leftExpr
			} else {
				sc.Channel = leftExpr
			}
		}

		if !p.expectPeek(lexer.FAT_ARROW) { // ->
			return nil
		}
		p.nextToken() // skip ->

		sc.Body = &ast.BlockStatement{
			Statements: []ast.Statement{
				&ast.ExpressionStatement{Expression: p.parseExpression(PrecedenceLowest)},
			},
		}

		expr.Cases = append(expr.Cases, sc)

		for p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
		if p.peekTokenIs(lexer.DEDENT) {
			p.nextToken()
			break
		}
		p.nextToken()
	}

	return expr
}

func (p *Parser) parseListLiteral() ast.Expression {
	list := &ast.ListLiteral{}
	list.Elements = p.parseExpressionList(lexer.RBRACKET)
	return list
}

func (p *Parser) parseMapLiteral() ast.Expression {
	m := &ast.MapLiteral{Pairs: make(map[string]ast.Expression)}

	if p.peekTokenIs(lexer.RBRACE) {
		p.nextToken()
		return m
	}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENT) {
			p.error("expected key in map")
			return nil
		}
		key := p.curToken.Literal

		if !p.expectPeek(lexer.COLON) {
			return nil
		}

		p.nextToken()
		val := p.parseExpression(PrecedenceLowest)
		m.Pairs[key] = val

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
		} else if p.peekTokenIs(lexer.RBRACE) {
			p.nextToken()
			break
		} else {
			p.nextToken()
		}
	}

	return m
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(PrecedenceLowest))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(PrecedenceLowest))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

// ---- Infix parsers ----

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) parsePipelineExpression(left ast.Expression) ast.Expression {
	if !p.isPipelineContext() {
		return p.parseInfixExpression(left)
	}

	expr := &ast.PipelineExpression{
		Left: left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)
	expr.Right = p.insertPipelinePlaceholder(right, left)

	return expr
}

func (p *Parser) parseParallelPipelineExpression(left ast.Expression) ast.Expression {
	if !p.isPipelineContext() {
		return p.parseInfixExpression(left)
	}

	expr := &ast.PipelineExpression{
		Left:     left,
		Parallel: true,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)
	expr.Right = p.insertPipelinePlaceholder(right, left)

	return expr
}

func (p *Parser) isPipelineContext() bool {
	switch p.peekToken.Type {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.TRUE, lexer.FALSE, lexer.NIL:
		return false
	case lexer.LPAREN:
		return false
	case lexer.MINUS, lexer.BANG:
		return false
	case lexer.IDENT:
		switch p.peekToken2.Type {
		case lexer.IDENT, lexer.INT, lexer.FLOAT, lexer.STRING, lexer.TRUE, lexer.FALSE, lexer.NIL,
			lexer.LPAREN, lexer.LBRACKET, lexer.LBRACE:
			return true
		default:
			return false
		}
	case lexer.IF, lexer.WHILE, lexer.FOR, lexer.FN, lexer.MATCH_KW, lexer.SELECT, lexer.TRY, lexer.TRYAI:
		return true
	default:
		return false
	}
}

func (p *Parser) insertPipelinePlaceholder(right ast.Expression, pipedValue ast.Expression) ast.Expression {
	callExpr, ok := right.(*ast.CallExpression)
	if !ok {
		return right
	}
	for i, arg := range callExpr.Arguments {
		if ident, ok := arg.(*ast.Identifier); ok && ident.Value == "_" {
			callExpr.Arguments[i] = pipedValue
			callExpr.PipedArg = true
			return callExpr
		}
	}
	return right
}

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	p.nextToken()
	if !p.curTokenIs(lexer.IDENT) {
		p.error("expected field name after '.'")
		return nil
	}
	return &ast.DotExpression{
		Left:  left,
		Field: p.curToken.Literal,
	}
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	call := &ast.CallExpression{Function: function}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return call
	}

	p.nextToken()
	call.Arguments = append(call.Arguments, p.parseExpression(PrecedenceLowest))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		call.Arguments = append(call.Arguments, p.parseExpression(PrecedenceLowest))
	}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
	}

	return call
}

// ---- while/for/break/continue/import ----

func (p *Parser) parseWhileStatement() ast.Statement {
	expr := p.parseWhileExpression()
	return &ast.ExpressionStatement{Expression: expr}
}

func (p *Parser) parseForStatement() ast.Statement {
	expr := p.parseForExpression()
	return &ast.ExpressionStatement{Expression: expr}
}

func (p *Parser) parseWhileExpression() ast.Expression {
	expr := &ast.WhileExpression{}
	p.nextToken() // skip 'while'
	expr.Condition = p.parseExpression(PrecedenceLowest)

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()
	expr.Body = p.parseBlock()
	return expr
}

func (p *Parser) parseForExpression() ast.Expression {
	expr := &ast.ForExpression{}
	p.nextToken() // skip 'for'

	// C-style: for ; cond ; update (empty init)
	if p.curTokenIs(lexer.SEMICOLON) {
		return p.parseCStyleBody(expr)
	}

	if !p.curTokenIs(lexer.IDENT) {
		p.error("for expects an iterator variable or ';'")
		return nil
	}
	iterName := p.curToken.Literal
	iterPos := ast.Position{Line: p.curToken.Line, Col: p.curToken.Col}

	// for-in: for IDENT in expr
	if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "in" {
		expr.IsForIn = true
		expr.Iterator = &ast.Identifier{Value: iterName, Line: iterPos.Line, Col: iterPos.Col}
		p.nextToken() // skip 'in'
		p.nextToken() // move to iterable expression
		expr.Iterable = p.parseExpression(PrecedenceLowest)

		if p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
		p.nextToken()
		expr.Body = p.parseBlock()
		return expr
	}

	// C-style for: for IDENT : init ; cond ; update
	if !p.peekTokenIs(lexer.COLON) {
		p.error("expected 'in' or ':' after 'for' variable")
		return nil
	}

	p.nextToken() // skip ':'
	p.nextToken() // move to init value
	expr.Init = &ast.VarStatement{
		Name:  &ast.Identifier{Value: iterName, Line: iterPos.Line, Col: iterPos.Col},
		Value: p.parseExpression(PrecedenceLowest),
	}

	if !p.expectPeek(lexer.SEMICOLON) {
		return nil
	}

	return p.parseCStyleBody(expr)
}

func (p *Parser) parseCStyleBody(expr *ast.ForExpression) ast.Expression {
	// Parse condition
	p.nextToken() // move to condition start
	if !p.curTokenIs(lexer.SEMICOLON) {
		expr.Condition = p.parseExpression(PrecedenceLowest)
		if !p.expectPeek(lexer.SEMICOLON) {
			return nil
		}
	} else {
		// empty condition: for ;; update → infinite loop
		expr.Condition = nil
	}

	// Parse update clause
	p.nextToken() // move to update start
	if !p.curTokenIs(lexer.SEMICOLON) && !p.peekTokenIs(lexer.NEWLINE) && !p.peekTokenIs(lexer.INDENT) {
		if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
			name := p.curToken.Literal
			namePos := ast.Position{Line: p.curToken.Line, Col: p.curToken.Col}
			p.nextToken() // skip ':'
			p.nextToken() // move to value
			expr.Update = &ast.VarStatement{
				Name:  &ast.Identifier{Value: name, Line: namePos.Line, Col: namePos.Col},
				Value: p.parseExpression(PrecedenceLowest),
			}
		} else {
			expr.Update = &ast.ExpressionStatement{
				Expression: p.parseExpression(PrecedenceLowest),
			}
		}
	}

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()
	expr.Body = p.parseBlock()
	return expr
}

func (p *Parser) parseReturnStatement() ast.Statement {
	p.nextToken() // skip 'return'
	return &ast.ReturnStatement{Value: p.parseExpression(PrecedenceLowest)}
}

func (p *Parser) parseDeferStatement() ast.Statement {
	p.nextToken() // skip 'defer'
	return &ast.DeferStatement{Expression: p.parseExpression(PrecedenceLowest)}
}

func (p *Parser) parseExportStatement() ast.Statement {
	p.nextToken() // skip 'export'

	switch p.curToken.Type {
	case lexer.FN:
		stmt := p.parseFnStatement()
		if fnStmt, ok := stmt.(*ast.FnStatement); ok {
			return &ast.ExportStatement{FnName: fnStmt.Name.Value, Fn: fnStmt}
		}
		p.error("cannot export inline lambda; use a named function")
	case lexer.IDENT:
		if p.peekTokenIs(lexer.COLON) {
			varStmt := p.parseVarStatement()
			if varStmt != nil {
				return &ast.ExportStatement{VarName: varStmt.Name.Value, Var: varStmt}
			}
		}
	case lexer.ENUM:
		enumStmt := p.parseEnumStatement()
		if es, ok := enumStmt.(*ast.EnumStatement); ok {
			return &ast.ExportStatement{EnumName: es.Name, Enum: es}
		}
	}

	p.error("export expects 'fn', variable, or 'enum'")
	return nil
}

func (p *Parser) parseEnumStatement() ast.Statement {
	stmt := &ast.EnumStatement{}

	p.nextToken() // skip 'enum'
	if !p.curTokenIs(lexer.IDENT) {
		p.error("enum expects a name")
		return nil
	}
	stmt.Name = p.curToken.Literal
	stmt.Line = p.curToken.Line
	stmt.Col = p.curToken.Col

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken() // skip colon

	for p.curTokenIs(lexer.IDENT) {
		stmt.Values = append(stmt.Values, p.curToken.Literal)
		stmt.ValuePos = append(stmt.ValuePos, ast.Position{Line: p.curToken.Line, Col: p.curToken.Col})
		p.nextToken()
		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // skip comma
		}
	}

	return stmt
}

func (p *Parser) parseStructStatement() ast.Statement {
	stmt := &ast.StructStatement{}

	p.nextToken() // skip 'struct'
	if !p.curTokenIs(lexer.IDENT) {
		p.error("struct expects a name")
		return nil
	}
	stmt.Name = p.curToken.Literal
	stmt.Line = p.curToken.Line
	stmt.Col = p.curToken.Col

	p.nextToken()

	if p.curTokenIs(lexer.COLON) {
		// Inline: struct Point: x, y
		p.nextToken()
		for p.curTokenIs(lexer.IDENT) {
			stmt.Fields = append(stmt.Fields, ast.StructField{Name: p.curToken.Literal})
			p.nextToken()
			if p.curTokenIs(lexer.COMMA) {
				p.nextToken()
			}
		}
		return stmt
	}

	// Block form with indentation
	if !p.curTokenIs(lexer.INDENT) && !p.curTokenIs(lexer.NEWLINE) {
		p.error("struct expects indented body or ':' after name")
		return nil
	}

	// Skip optional newline before INDENT
	if p.curTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	if p.curTokenIs(lexer.INDENT) {
		p.nextToken()
	}

	for !p.curTokenIs(lexer.DEDENT) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENT) {
			p.error("struct field must be an identifier")
			return nil
		}
		fieldName := p.curToken.Literal

		var defExpr ast.Expression

		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // skip field name
			p.nextToken() // skip colon
			defExpr = p.parseExpression(PrecedenceLowest)
		}

		stmt.Fields = append(stmt.Fields, ast.StructField{
			Name:    fieldName,
			Default: defExpr,
		})

		p.nextToken()
		if p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
	}

	return stmt
}

func (p *Parser) parseTestStatement() ast.Statement {
	stmt := &ast.TestStatement{}

	p.nextToken() // skip 'test'

	// File-level hooks use the bare keywords `test setup` / `test teardown`
	// (no string description). These are not reserved globally, so existing
	// code that uses setup/teardown as identifiers keeps parsing.
	if p.curTokenIs(lexer.IDENT) && (p.curToken.Literal == "setup" || p.curToken.Literal == "teardown") {
		stmt.Hook = p.curToken.Literal
		p.nextToken()
		stmt.Body = p.parseBlock()
		return stmt
	}

	if !p.curTokenIs(lexer.STRING) {
		p.error("test expects a string description or a setup/teardown hook")
		return nil
	}
	stmt.Name = &ast.StringLiteral{Value: p.curToken.Literal}

	p.nextToken()
	stmt.Body = p.parseBlock()

	return stmt
}

func (p *Parser) parseImportStatement() ast.Statement {
	p.nextToken() // skip 'import'
	if !p.curTokenIs(lexer.STRING) {
		p.error("import expects a string path")
		return nil
	}
	stmt := &ast.ImportStatement{Path: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}

	// Optional: as <alias>
	if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "as" {
		p.nextToken() // skip 'as'
		p.nextToken() // move to alias name
		if p.curTokenIs(lexer.IDENT) {
			stmt.Alias = p.curToken.Literal
		}
	}

	return stmt
}

func (p *Parser) parseFnLiteral() ast.Expression {
	lit := &ast.FnLiteral{Line: p.curToken.Line, Col: p.curToken.Col}

	p.nextToken() // skip 'fn'

	// Collect all identifiers as parameters
	for p.curTokenIs(lexer.IDENT) {
		lit.Parameters = append(lit.Parameters, &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col})
		p.nextToken()
	}

	// Inline form: fn params...: expression
	if p.curTokenIs(lexer.COLON) {
		p.nextToken() // skip colon
		expr := p.parseExpression(PrecedenceLowest)
		if expr != nil {
			lit.Body = &ast.BlockStatement{
				Statements: []ast.Statement{
					&ast.ExpressionStatement{Expression: expr},
				},
			}
		}
		return lit
	}

	// Body (indented block form)
	lit.Body = p.parseBlock()
	return lit
}

func (p *Parser) parseIndexOrSlice(left ast.Expression) ast.Expression {
	p.nextToken() // skip [

	// Empty index? []
	if p.curTokenIs(lexer.RBRACKET) {
		return left
	}

	startExp := p.parseExpression(PrecedenceLowest)

	// Slice: start..end or ..end
	if p.peekTokenIs(lexer.DOTDOT) {
		p.nextToken() // skip ..
		p.nextToken() // move to end expression
		var endExp ast.Expression
		if !p.curTokenIs(lexer.RBRACKET) {
			endExp = p.parseExpression(PrecedenceLowest)
		}
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
		return &ast.SliceExpression{List: left, Start: startExp, End: endExp}
	}

	// Simple index: [expr]
	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	// Return a call-like expression for index access (treated as at in evaluator)
	return &ast.InfixExpression{Operator: "[]", Left: left, Right: startExp}
}

func (p *Parser) parseTryExpression() ast.Expression {
	return p.parseTryBase(false)
}

func (p *Parser) parseTryAIExpression() ast.Expression {
	expr := p.parseTryBase(true)
	if expr == nil {
		return nil
	}
	expr.(*ast.TryExpression).AIFix = true
	return expr
}

func (p *Parser) parseTryBase(aiFix bool) ast.Expression {
	expr := &ast.TryExpression{}

	p.nextToken() // skip 'try' or 'try_ai'
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()
	expr.TryBlock = p.parseBlock()

	// catch is optional for try_ai, required for try
	if !p.peekTokenIs(lexer.CATCH) {
		if !aiFix {
			p.error("try without catch")
			return nil
		}
		return expr
	}
	p.nextToken() // consume 'catch'
	p.nextToken() // move to catch param name
	if p.curTokenIs(lexer.IDENT) {
		expr.CatchParam = &ast.Identifier{Value: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col}
	}
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()
	expr.CatchBlock = p.parseBlock()

	return expr
}

func QuoteString(s string) string {
	var buf strings.Builder
	buf.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '\n':
			buf.WriteString("\\n")
		case '\t':
			buf.WriteString("\\t")
		case '\r':
			buf.WriteString("\\r")
		case '\\':
			buf.WriteString("\\\\")
		case '"':
			buf.WriteString("\\\"")
		default:
			buf.WriteRune(ch)
		}
	}
	buf.WriteByte('"')
	return buf.String()
}
